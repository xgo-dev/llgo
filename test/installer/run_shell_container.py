#!/usr/bin/env python3
"""Bound the Docker client as well as the installer commands inside it."""

import json
import math
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import tempfile
import time
import uuid


OWNER_LABEL = "org.llgo.installer-shell-test"
FULL_ID = re.compile(r"[0-9a-f]{64}")
ROOT = Path(__file__).resolve().parents[2]


def seconds(name, default):
    value = float(os.environ.get(name, default))
    if not math.isfinite(value) or value <= 0:
        raise ValueError(f"{name} must be a positive finite number")
    return value


class Stopped(Exception):
    def __init__(self, status, reason):
        self.status = status
        super().__init__(reason)


def terminate_client(process):
    # Docker commands get their own session. Never signal the caller's group.
    # Keep the group leader unreaped through the grace period: it prevents PID
    # reuse, and its exit does not imply that its children handled TERM.
    for sig in (signal.SIGTERM, signal.SIGKILL):
        try:
            os.killpg(process.pid, sig)
        except ProcessLookupError:
            pass
        if sig == signal.SIGTERM:
            time.sleep(0.1)
    try:
        process.wait(timeout=0.5)
    except subprocess.TimeoutExpired:
        print("installer container: Docker client did not reap after KILL", file=sys.stderr)


class ContainerRun:
    def __init__(self, image, setup, directory):
        self.image = image
        self.setup = setup
        self.token = uuid.uuid4().hex
        self.name = "llgo-installer-" + self.token
        self.cidfile = Path(directory) / "container.cid"
        self.cid = None
        self.signal = None
        # Include image pulls/package setup, but leave time below the 20 minute
        # CI deadline for diagnostics and cleanup (each independently bounded).
        self.deadline = time.monotonic() + seconds("LLGO_SHELL_CONTAINER_TIMEOUT", 900)
        self.cleanup_timeout = seconds("LLGO_SHELL_CONTAINER_CLEANUP_TIMEOUT", 5)

    def interrupted(self, signum, _frame):
        if self.signal is None:
            self.signal = signum

    def command(self, arguments, *, cleanup=False, capture=False):
        deadline = time.monotonic() + self.cleanup_timeout if cleanup else self.deadline
        if not cleanup and self.signal is not None:
            raise Stopped(128 + self.signal, "interrupted by signal")
        if not cleanup and time.monotonic() >= deadline:
            raise Stopped(124, f"host deadline exceeded before docker {arguments[0]}")
        with tempfile.TemporaryFile() as output:
            # A regular file, not an inherited pipe, prevents a stranded client
            # descendant from keeping the watchdog's own stdout open forever.
            process = subprocess.Popen(
                ["docker", *arguments], stdin=subprocess.DEVNULL,
                stdout=output, stderr=subprocess.STDOUT, start_new_session=True,
            )
            offset = 0

            def forward():
                nonlocal offset
                if not capture:
                    data = os.pread(output.fileno(), 65536, offset)
                    if data:
                        sys.stdout.buffer.write(data)
                        sys.stdout.buffer.flush()
                        offset += len(data)
                    return bool(data)
                return False

            stopped = None
            while process.poll() is None:
                forward()
                if self.signal is not None and not cleanup:
                    stopped = Stopped(128 + self.signal, "interrupted by signal")
                    break
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    label = "cleanup deadline" if cleanup else "host deadline"
                    stopped = Stopped(124, f"{label} exceeded during docker {arguments[0]}")
                    break
                try:
                    process.wait(timeout=min(0.05, remaining))
                except subprocess.TimeoutExpired:
                    pass
            if stopped is not None:
                terminate_client(process)
            # Do not let an unexpectedly noisy descendant extend the deadline
            # indefinitely after its client has been terminated.
            for _ in range(16):
                if not forward():
                    break
            else:
                print("installer container: remaining client output truncated", file=sys.stderr)
            if stopped is not None:
                if not cleanup:
                    raise stopped
                print(f"installer container: {stopped}", file=sys.stderr)
                return stopped.status, ""
            status = process.returncode
            if status < 0:
                status = 128 - status
            text = ""
            if capture:
                text = os.pread(output.fileno(), 1024 * 1024, 0).decode("utf-8", "replace")
            return status, text

    def recorded_id(self):
        if not self.cidfile.exists():
            return ""
        with self.cidfile.open() as source:
            value = source.read(66)
        return value.strip() if len(value) <= 65 else ""

    def identify(self, candidate, *, cleanup=False):
        status, output = self.command(
            ["inspect", "--format", "{{json .}}", candidate], cleanup=cleanup, capture=True,
        )
        if status:
            return None
        try:
            info = json.loads(output)
            cid = info["Id"]
            owned = (
                isinstance(cid, str) and FULL_ID.fullmatch(cid)
                and (candidate == self.name or candidate == cid)
                and info["Name"] == "/" + self.name
                and info["Config"]["Labels"].get(OWNER_LABEL) == self.token
            )
            if owned:
                self.cid = cid
                return info
        except (ValueError, KeyError, TypeError, AttributeError):
            pass
        print("installer container: refused unverified container identity", file=sys.stderr)
        return None

    def steps(self):
        script = (
            f"{self.setup} && fish --version && timeout --verbose -k 5 120 "
            "bash test/installer/install_shell_test.sh bash zsh fish sh"
        )
        status, _ = self.command([
            "create", "--init", "--name", self.name,
            "--label", f"{OWNER_LABEL}={self.token}", "--cidfile", str(self.cidfile),
            "-e", "LLGO_SHELL_TEST_TIMEOUT=10", "-v", f"{ROOT}:/src:ro", "-w", "/src",
            self.image, "sh", "-ec", script,
        ])
        if status:
            return status
        cid = self.recorded_id()
        if not FULL_ID.fullmatch(cid):
            print("installer container: missing or invalid full container ID", file=sys.stderr)
            return 125
        if self.identify(cid) is None:
            return 125
        status, _ = self.command(["start", "--attach", self.cid])
        if status:
            return status
        info = self.identify(self.cid)
        if info is None or info["State"]["Running"] or info["State"]["Status"] != "exited":
            print("installer container: attach exited without confirmed container completion", file=sys.stderr)
            return 125
        status = info["State"]["ExitCode"]
        if type(status) is not int or not 0 <= status <= 255:
            print("installer container: invalid container exit status", file=sys.stderr)
            return 125
        return status

    def cleanup(self, status):
        if self.cid is None:
            candidate = self.recorded_id()
            # Create may time out before publishing its CID file. The unique
            # name is only a read-only lookup; mutations still require a full
            # ID with both this invocation's label and its name verified.
            if not FULL_ID.fullmatch(candidate):
                candidate = self.name
            self.identify(candidate, cleanup=True)
        if self.cid is None:
            print("installer container: no verified container to clean up", file=sys.stderr)
            return False
        if status:
            self.command(["logs", "--tail", "100", self.cid], cleanup=True)
            self.command(["inspect", "--format", "{{json .State}}", self.cid], cleanup=True)
            self.command(["top", self.cid, "-eo", "pid,ppid,pgid,sid,stat,wchan,args"], cleanup=True)
            self.command(["kill", self.cid], cleanup=True)
        removed, _ = self.command(["rm", "--force", self.cid], cleanup=True)
        return removed == 0

    def run(self):
        for sig in (signal.SIGTERM, signal.SIGINT):
            signal.signal(sig, self.interrupted)
        status = 125
        try:
            status = self.steps()
        except Stopped as error:
            print(f"installer container: {error}", file=sys.stderr)
            status = error.status
        except (OSError, ValueError, KeyError, TypeError) as error:
            print(f"installer container: {error}", file=sys.stderr)
        finally:
            try:
                cleaned = self.cleanup(status)
            except (OSError, ValueError, KeyError, TypeError) as error:
                print(f"installer container: cleanup failed: {error}", file=sys.stderr)
                cleaned = False
            if status == 0 and not cleaned:
                status = 125
        return 128 + self.signal if self.signal is not None else status


def main():
    if len(sys.argv) != 3:
        print("usage: run_shell_container.py IMAGE SETUP_COMMAND", file=sys.stderr)
        return 2
    try:
        with tempfile.TemporaryDirectory(prefix="llgo-installer-container-") as directory:
            return ContainerRun(sys.argv[1], sys.argv[2], directory).run()
    except ValueError as error:
        print(f"installer container: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

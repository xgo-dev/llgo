#!/usr/bin/env python3
"""Stream a command to stdout and a log through a native subprocess pipe."""

from pathlib import Path
import argparse
import datetime
import json
import math
import os
import signal
import subprocess
import sys
import threading
import time


def stream(process, output):
    while chunk := process.stdout.read(16384):
        output.write(chunk)
        output.flush()
        sys.stdout.buffer.write(chunk)
        sys.stdout.buffer.flush()


def run(log, command, timeout=None, label=None):
    if timeout is not None:
        return bounded(log, command, timeout, label or "command")
    # Git Bash's tee pipe can deadlock a native Windows writer after sustained
    # output. Create the child pipe in Python, and forward bytes without a shell
    # pipe or text conversion so quarantine matching sees the original output.
    with Path(log).open("wb") as output:
        with subprocess.Popen(command, stdout=subprocess.PIPE,
                              stderr=subprocess.STDOUT, bufsize=0) as process:
            stream(process, output)
            status = process.wait()
    return status if status >= 0 else 128 - status


class PosixGroup:
    def __init__(self, command):
        self.terminated, self.exited = False, False
        self.process = subprocess.Popen(command, stdout=subprocess.PIPE,
                                        stderr=subprocess.STDOUT, bufsize=0,
                                        start_new_session=True)

    def wait(self, timeout):
        # Observe exit without reaping the group leader. Its PID cannot be
        # reused until close() has terminated the remaining owned processes.
        expires = time.monotonic() + timeout
        while True:
            result = os.waitid(os.P_PID, self.process.pid,
                               os.WEXITED | os.WNOWAIT | os.WNOHANG)
            if result is not None:
                self.exited = True
                return result.si_status if result.si_code == os.CLD_EXITED else -result.si_status
            remaining = expires - time.monotonic()
            if remaining <= 0:
                raise subprocess.TimeoutExpired(self.process.args, timeout)
            time.sleep(min(remaining, 0.01))

    def terminate(self):
        if self.terminated:
            return
        self.terminated = True
        try:
            os.killpg(self.process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        except PermissionError:
            # Darwin reports EPERM for a group containing only the unreaped
            # zombie leader. Keep that PID leased while proving there are no
            # live members; never suppress real signal/query permission errors.
            if sys.platform != "darwin" or not self.exited:
                raise
            result = subprocess.run(["/bin/ps", "-A", "-o", "pgid=", "-o", "stat="],
                                    capture_output=True, text=True, timeout=2)
            if result.returncode != 0:
                raise OSError("could not inspect exited process group")
            rows = result.stdout.splitlines()
            if not rows:
                raise OSError("empty process-group inspection output")
            for line in rows:
                group, state = line.split()
                if int(group) == self.process.pid and not state.startswith("Z"):
                    raise

    def close(self):
        # Never signal a reaped PID: its process-group identity may be reused.
        if self.process.returncode is None:
            self.terminate()


def notice(message):
    sys.stderr.buffer.write((message + "\n").encode("utf-8"))
    sys.stderr.buffer.flush()


def bounded(log, command, timeout, label):
    if not math.isfinite(timeout) or timeout <= 0:
        raise ValueError("command timeout must be finite and positive")
    path = Path(log)
    path.parent.mkdir(parents=True, exist_ok=True)
    metadata = path.with_suffix(path.suffix + ".json")
    start = time.monotonic()
    expires = start + timeout
    record = {"label": label, "command": command, "timeout_seconds": timeout,
              "started_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "cwd": os.getcwd(), "status": "starting", "python": sys.executable,
              "host_os": os.name,
              "owner": "windows-job" if os.name == "nt" else "posix-process-group"}

    def save():
        temporary = metadata.with_suffix(metadata.suffix + ".tmp")
        temporary.write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
        temporary.replace(metadata)

    save()
    notice(f"Starting {label} (timeout {timeout:g}s; log {path})")
    owner, reader, output = None, None, None
    drained = threading.Event()
    interrupted, cleanup_errors, reader_errors, cancellations = [], [], [], []
    old_handlers = {}
    status = 125

    def cleanup(operation):
        try:
            return operation()
        except Exception as error:
            cleanup_errors.append(f"{type(error).__name__}: {error}")

    def stop(reason, code):
        if interrupted:
            return
        interrupted.append((reason, code))
        notice(f"{label}: {reason}")
        if owner is not None:
            try:
                owner.terminate()
            except Exception as error:
                cleanup_errors.append(f"terminate: {error}")
                cleanup(owner.close)

    def read_output():
        try:
            stream(owner.process, output)
        except Exception as error:
            reader_errors.append(str(error))
        finally:
            drained.set()

    try:
        if os.environ.get("RUNNER_OS") == "Windows" and os.name != "nt":
            raise OSError("Windows demo ownership requires the native host Python interpreter")
        for signum in (signal.SIGTERM, signal.SIGINT):
            old_handlers[signum] = signal.signal(
                signum, lambda number, frame: cancellations.append(number))
        output = path.open("wb", buffering=0)
        if os.name == "nt":
            from windows_job import WindowsJob
            owner = WindowsJob(command)
        else:
            owner = PosixGroup(command)
        record.update(status="running", pid=owner.process.pid)
        save()
        notice(f"Started {label} (pid {owner.process.pid})")
        if cancellations:
            stop(f"canceled by signal {cancellations[0]}", 128 + cancellations[0])
        reader = threading.Thread(target=read_output, daemon=True)
        reader.start()
        # Startup, pipe draining and the actual process share one deadline.
        # Signal handlers only queue cancellation; avoid I/O and locks there.
        while not interrupted:
            if cancellations:
                stop(f"canceled by signal {cancellations[0]}", 128 + cancellations[0])
                break
            remaining = expires - time.monotonic()
            if remaining <= 0:
                stop("timed out", 124)
                break
            if reader_errors:
                stop("output logging failed", 125)
                break
            if not drained.wait(min(remaining, 0.1)):
                continue
            try:
                status = owner.wait(timeout=max(0, min(expires - time.monotonic(), 0.1)))
                break
            except subprocess.TimeoutExpired:
                continue
        if interrupted:
            if not drained.wait(2):
                cleanup_errors.append("command output pipe did not close after termination")
            cleanup(lambda: owner.process.wait(timeout=2))
            record["reason"], status = interrupted[0]
        status = status if status >= 0 else 128 - status
    except Exception as error:
        record["error"] = str(error)
        notice(f"{label}: {error}")
        status = 125
    finally:
        if owner is not None:
            cleanup(owner.close)
            cleanup(lambda: owner.process.wait(timeout=2))
        if reader is not None:
            reader.join(timeout=0.1)
        if owner is not None:
            if reader is None or drained.is_set():
                cleanup(owner.process.stdout.close)
            else:
                # Closing a pipe while another thread is reading can itself
                # block on Windows. Record failure and let supervisor exit
                # close this descriptor instead of losing the deadline here.
                cleanup_errors.append("output reader did not finish; pipe left for process exit")
        if output is not None:
            cleanup(output.close)
        for signum, handler in old_handlers.items():
            signal.signal(signum, handler)
    if cleanup_errors or reader_errors:
        record["cleanup_errors"] = cleanup_errors + reader_errors
        status = 125
    record.update(status="passed" if status == 0 else "failed",
                  exit_code=status, elapsed_seconds=time.monotonic() - start)
    save()
    notice(f"Finished {label} (exit {status}; {record['elapsed_seconds']:.3f}s)")
    return status


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout", type=float)
    parser.add_argument("--label")
    parser.add_argument("log", type=Path)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if not args.command:
        parser.error("a command is required")
    if args.timeout is not None and (not math.isfinite(args.timeout) or args.timeout <= 0):
        parser.error("timeout must be finite and positive")
    sys.exit(run(args.log, args.command, timeout=args.timeout, label=args.label))

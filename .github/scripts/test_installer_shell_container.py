"""Exercise the host watchdog without Docker, downloads, or compiler builds."""

import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[2]
WATCHDOG = ROOT / "test/installer/run_shell_container.py"
CID = "a" * 64

# Each invocation is a real subprocess. In particular, the stalled commands
# ignore TERM, so tests exercise the watchdog's forced client termination.
FAKE_DOCKER = r'''
import json
import os
from pathlib import Path
import signal
import sys
import time

root = Path(os.environ["FAKE_DOCKER_ROOT"])
mode = os.environ["FAKE_DOCKER_MODE"]
args = sys.argv[1:]
with (root / "calls.jsonl").open("a") as output:
    output.write(json.dumps(args) + "\n")

def stall():
    signal.signal(signal.SIGTERM, signal.SIG_IGN)
    while True:
        time.sleep(1)

if args[0] == "create":
    cid = "a" * 64
    label = args[args.index("--label") + 1]
    name = args[args.index("--name") + 1]
    state = {"Id": cid, "Name": "/" + name,
             "Config": {"Labels": dict([label.split("=", 1)])},
             "State": {"Running": False, "Status": "created", "ExitCode": 0}}
    (root / "state.json").write_text(json.dumps(state))
    if mode in ("empty-cid", "create-hang-no-id"):
        cid = ""
    elif mode == "malicious-cid":
        cid = "--all"
    elif mode == "foreign-cid":
        cid = "b" * 64
    Path(args[args.index("--cidfile") + 1]).write_text(cid)
    if mode in ("create-hang", "create-hang-no-id"):
        stall()
    print(cid)
    sys.exit(0)

if args[0] == "inspect":
    if mode in ("empty-cid", "malicious-cid"):
        sys.exit(1)
    state = json.loads((root / "state.json").read_text())
    if mode == "foreign-cid":
        state["Id"] = "b" * 64
        state["Name"] = "/unrelated-user-container"
        state["Config"]["Labels"] = {}
    print(json.dumps(state))
elif args[0] == "start":
    state = json.loads((root / "state.json").read_text())
    state["State"] = {"Running": False, "Status": "exited", "ExitCode": 0}
    if mode in ("failure", "lost-exit-code"):
        state["State"]["ExitCode"] = 37
    elif mode == "invalid-exit-code":
        state["State"]["ExitCode"] = 256
    elif mode == "still-running":
        state["State"].update(Running=True, Status="running")
    (root / "state.json").write_text(json.dumps(state))
    (root / "started").write_text(str(os.getpid()))
    (root / "watchdog.pid").write_text(str(os.getppid()))
    print("CONTAINER_START_DIAGNOSTIC", flush=True)
    if mode == "orphan-child":
        if os.fork() == 0:
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
            (root / "child.json").write_text(json.dumps({"pid": os.getpid(), "pgid": os.getpgrp()}))
            stall()
        while True:
            time.sleep(1)
    if mode == "cleanup-hang":
        # Enter cleanup after identity verification, independent of the host
        # watchdog deadline. Other cases exercise a stalled attach.
        sys.exit(124)
    if mode in ("attach-hang", "signal"):
        stall()
    sys.exit(37 if mode == "failure" else 0)
elif args[0] == "logs":
    print("CONTAINER_FAILURE_LOG", flush=True)
    if mode == "cleanup-hang":
        stall()
elif args[0] == "top":
    print("CONTAINER_PROCESS_DIAGNOSTIC", flush=True)
elif args[0] in ("kill", "rm"):
    if mode == "cleanup-hang":
        stall()
    if mode == "remove-failure" and args[0] == "rm":
        sys.exit(1)
else:
    sys.exit(2)
'''


class InstallerShellContainerTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(prefix="installer watchdog ")
        self.addCleanup(self.directory.cleanup)
        self.temporary = Path(self.directory.name)
        docker = self.temporary / "docker"
        docker.write_text(f"#!{sys.executable}\n" + FAKE_DOCKER)
        docker.chmod(0o755)

    def start(self, mode, *, distro=False, timeout="5", cleanup_timeout="1"):
        # Successful fixture commands need startup headroom on loaded runners;
        # only dedicated stall cases consume their explicit short host budget.
        environment = dict(
            os.environ,
            PATH=f"{self.temporary}{os.pathsep}{os.environ['PATH']}",
            FAKE_DOCKER_ROOT=str(self.temporary),
            FAKE_DOCKER_MODE=mode,
            LLGO_SHELL_CONTAINER_TIMEOUT=timeout,
            LLGO_SHELL_CONTAINER_CLEANUP_TIMEOUT=cleanup_timeout,
        )
        command = [sys.executable, str(WATCHDOG), "fedora:fixture", "true"]
        if distro:
            command = ["bash", str(ROOT / "test/installer/install_shell_distro_test.sh"),
                       "fedora:fixture", "debian:fixture"]
        process = subprocess.Popen(
            command,
            env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
        self.addCleanup(self.stop, process)
        return process

    def wait_started(self, process, filename="started"):
        deadline = time.monotonic() + 3
        while not (self.temporary / filename).exists():
            if process.poll() is not None or time.monotonic() >= deadline:
                self.fail("watchdog did not start the fixture container")
            time.sleep(0.01)

    @staticmethod
    def stop(process):
        if process.poll() is None:
            process.kill()
        process.communicate(timeout=3)

    def finish(self, process):
        stdout, stderr = process.communicate(timeout=8)
        calls_file = self.temporary / "calls.jsonl"
        calls = []
        if calls_file.exists():
            calls = [json.loads(line) for line in calls_file.read_text().splitlines()]
        # Every mutating cleanup command must target the full, owned ID, never
        # a name, an empty string, a user-supplied option, or a broad selector.
        for call in calls:
            if call[0] in ("kill", "rm"):
                self.assertEqual(call[-1], CID)
                self.assertNotIn("--all", call)
        return process.returncode, stdout + stderr, calls

    def test_success_preserves_container_exit_and_removes_only_owned_id(self):
        status, output, calls = self.finish(self.start("success"))
        self.assertEqual(status, 0, output)
        self.assertIn("CONTAINER_START_DIAGNOSTIC", output)
        self.assertTrue(any(call[0] == "rm" for call in calls))
        self.assertFalse(any(call[0] == "kill" for call in calls))
        create = next(call for call in calls if call[0] == "create")
        self.assertIn("LLGO_SHELL_TEST_TIMEOUT=10", create)
        self.assertIn("timeout --verbose -k 5 120", create[-1])

    def test_failed_container_preserves_exit_and_failure_logs(self):
        status, output, calls = self.finish(self.start("failure"))
        self.assertEqual(status, 37, output)
        self.assertIn("CONTAINER_FAILURE_LOG", output)
        self.assertIn("CONTAINER_PROCESS_DIAGNOSTIC", output)
        self.assertTrue(any(call[0] == "inspect" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_stalled_attach_is_bounded_and_not_reported_as_success(self):
        status, output, calls = self.finish(self.start("attach-hang", timeout="2"))
        self.assertEqual(status, 124, output)
        self.assertIn("host deadline", output)
        self.assertIn("CONTAINER_FAILURE_LOG", output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_client_exit_does_not_leave_term_ignoring_child_alive(self):
        process = self.start("orphan-child", timeout="2")
        self.wait_started(process, "child.json")
        child = json.loads((self.temporary / "child.json").read_text())
        self.assertNotEqual(child["pgid"], os.getpgrp())

        def child_alive():
            try:
                os.kill(child["pid"], 0)
            except ProcessLookupError:
                return False
            # A reparented zombie cannot execute or retain open descriptors.
            # Reaping it belongs to the host init process, not this watchdog.
            state = subprocess.run(["ps", "-o", "stat=", "-p", str(child["pid"])],
                                   capture_output=True, text=True, timeout=2).stdout.strip()
            return bool(state) and not state.startswith("Z")

        def cleanup_child():
            if child_alive():
                os.kill(child["pid"], signal.SIGKILL)

        self.addCleanup(cleanup_child)
        status, output, _ = self.finish(process)
        self.assertEqual(status, 124, output)
        deadline = time.monotonic() + 1
        while child_alive() and time.monotonic() < deadline:
            time.sleep(0.01)
        self.assertFalse(child_alive(), "Docker client's TERM-ignoring child survived")

    def test_stalled_create_with_published_id_is_also_cleaned_up(self):
        status, output, calls = self.finish(self.start("create-hang", timeout="2"))
        self.assertEqual(status, 124, output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_stalled_create_without_id_uses_verified_name_lookup_only(self):
        status, output, calls = self.finish(self.start("create-hang-no-id", timeout="2"))
        self.assertEqual(status, 124, output)
        self.assertTrue(any(call[0] == "inspect" and call[-1].startswith("llgo-installer-")
                            for call in calls))
        self.assertTrue(any(call[0] == "rm" and call[-1] == CID for call in calls))

    def test_attach_zero_does_not_hide_a_failed_container(self):
        status, output, _ = self.finish(self.start("lost-exit-code"))
        self.assertEqual(status, 37, output)

    def test_running_invalid_status_or_failed_cleanup_is_not_success(self):
        for mode in ("still-running", "invalid-exit-code", "remove-failure"):
            with self.subTest(mode=mode):
                status, output, _ = self.finish(self.start(mode))
                self.assertEqual(status, 125, output)

    def test_cleanup_commands_have_their_own_deadlines(self):
        status, output, calls = self.finish(self.start("cleanup-hang", timeout="5", cleanup_timeout="1"))
        self.assertEqual(status, 124, output)
        for command in ("logs", "kill", "rm"):
            self.assertIn(f"cleanup deadline exceeded during docker {command}", output)
            self.assertTrue(any(call[0] == command for call in calls), output)

    def test_signal_preserves_failure_and_cleans_owned_container(self):
        process = self.start("signal")
        self.wait_started(process)
        process.send_signal(signal.SIGTERM)
        status, output, calls = self.finish(process)
        self.assertEqual(status, 128 + signal.SIGTERM, output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_real_distro_entrypoint_uses_host_deadline(self):
        status, output, calls = self.finish(self.start("attach-hang", distro=True, timeout="2"))
        self.assertEqual(status, 1, output)
        self.assertIn("FAIL fedora:fixture (exit 124)", output)
        self.assertNotIn("PASS fedora:fixture", output)
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_distro_does_not_start_another_image_after_interruption(self):
        process = self.start("signal", distro=True)
        self.wait_started(process, "watchdog.pid")
        os.kill(int((self.temporary / "watchdog.pid").read_text()), signal.SIGTERM)
        status, output, calls = self.finish(process)
        self.assertEqual(status, 143, output)
        self.assertEqual(sum(call[0] == "create" for call in calls), 1)

    def test_invalid_or_foreign_ids_are_never_started_or_removed(self):
        for mode in ("empty-cid", "malicious-cid", "foreign-cid"):
            with self.subTest(mode=mode):
                calls_file = self.temporary / "calls.jsonl"
                calls_file.unlink(missing_ok=True)
                status, output, calls = self.finish(self.start(mode))
                self.assertNotEqual(status, 0, output)
                self.assertTrue(any(call[0] == "create" for call in calls))
                self.assertFalse(any(call[0] in ("start", "kill", "rm") for call in calls))


if __name__ == "__main__":
    unittest.main()

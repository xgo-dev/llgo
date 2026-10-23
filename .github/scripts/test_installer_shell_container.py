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
    if mode == "empty-cid":
        cid = ""
    elif mode == "malicious-cid":
        cid = "--all"
    elif mode == "foreign-cid":
        cid = "b" * 64
    Path(args[args.index("--cidfile") + 1]).write_text(cid)
    if mode == "create-hang":
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
    state["State"] = {"Running": False, "Status": "exited",
                      "ExitCode": 37 if mode == "failure" else 0}
    (root / "state.json").write_text(json.dumps(state))
    (root / "started").write_text(str(os.getpid()))
    print("CONTAINER_START_DIAGNOSTIC", flush=True)
    if mode in ("attach-hang", "signal", "cleanup-hang"):
        stall()
    sys.exit(37 if mode == "failure" else 0)
elif args[0] == "logs":
    print("CONTAINER_FAILURE_LOG", flush=True)
    if mode == "cleanup-hang":
        stall()
elif args[0] in ("kill", "rm"):
    if mode == "cleanup-hang":
        stall()
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

    def start(self, mode):
        environment = dict(
            os.environ,
            PATH=f"{self.temporary}{os.pathsep}{os.environ['PATH']}",
            FAKE_DOCKER_ROOT=str(self.temporary),
            FAKE_DOCKER_MODE=mode,
            LLGO_SHELL_CONTAINER_TIMEOUT="1",
            LLGO_SHELL_CONTAINER_CLEANUP_TIMEOUT="0.25",
        )
        process = subprocess.Popen(
            [sys.executable, str(WATCHDOG), "fedora:fixture", "true"],
            env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
        )
        self.addCleanup(self.stop, process)
        return process

    @staticmethod
    def stop(process):
        if process.poll() is None:
            process.kill()
            process.communicate(timeout=3)

    def finish(self, process):
        stdout, stderr = process.communicate(timeout=8)
        calls_file = self.temporary / "calls.jsonl"
        calls = [json.loads(line) for line in calls_file.read_text().splitlines()] if calls_file.exists() else []
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
        self.assertTrue(any(call[0] == "inspect" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_stalled_attach_is_bounded_and_not_reported_as_success(self):
        status, output, calls = self.finish(self.start("attach-hang"))
        self.assertEqual(status, 124, output)
        self.assertIn("host deadline", output)
        self.assertIn("CONTAINER_FAILURE_LOG", output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_stalled_create_with_published_id_is_also_cleaned_up(self):
        status, output, calls = self.finish(self.start("create-hang"))
        self.assertEqual(status, 124, output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_cleanup_commands_have_their_own_deadlines(self):
        status, output, calls = self.finish(self.start("cleanup-hang"))
        self.assertEqual(status, 124, output)
        self.assertIn("cleanup deadline", output)
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_signal_preserves_failure_and_cleans_owned_container(self):
        process = self.start("signal")
        deadline = time.monotonic() + 3
        while not (self.temporary / "started").exists():
            if process.poll() is not None or time.monotonic() >= deadline:
                self.fail("watchdog did not start the fixture container")
            time.sleep(0.01)
        process.send_signal(signal.SIGTERM)
        status, output, calls = self.finish(process)
        self.assertEqual(status, 128 + signal.SIGTERM, output)
        self.assertTrue(any(call[0] == "kill" for call in calls))
        self.assertTrue(any(call[0] == "rm" for call in calls))

    def test_invalid_or_foreign_ids_are_never_started_or_removed(self):
        for mode in ("empty-cid", "malicious-cid", "foreign-cid"):
            with self.subTest(mode=mode):
                calls_file = self.temporary / "calls.jsonl"
                calls_file.unlink(missing_ok=True)
                status, output, calls = self.finish(self.start(mode))
                self.assertNotEqual(status, 0, output)
                self.assertFalse(any(call[0] in ("start", "kill", "rm") for call in calls))


if __name__ == "__main__":
    unittest.main()

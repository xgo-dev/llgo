import json
import io
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import time
import types
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "dev" / "run_logged_command.py"
sys.path.insert(0, str(ROOT / "dev"))
import windows_job
import run_logged_command


class LoggedCommandTests(unittest.TestCase):
    def fake_owner(self):
        owner = mock.Mock()
        owner.process.stdout = io.BytesIO(b"command-live\n")
        owner.process.wait.return_value = 0
        owner.wait.return_value = 0
        owner.process.pid = 123
        return owner

    def owner_patch(self, factory):
        target = "windows_job.WindowsJob" if os.name == "nt" else "run_logged_command.PosixGroup"
        return mock.patch(target, side_effect=factory)

    def test_startup_time_consumes_the_same_absolute_deadline(self):
        owner = self.fake_owner()

        def delayed(command):
            time.sleep(0.06)
            return owner

        with tempfile.TemporaryDirectory() as temporary, self.owner_patch(delayed):
            log = Path(temporary) / "run.log"
            status = run_logged_command.run(log, ["fake"], timeout=0.01, label="run delayed")
            self.assertEqual(status, 124)
            self.assertEqual(json.loads(log.with_suffix(".log.json").read_text())["reason"], "timed out")

    def test_cleanup_failure_is_terminal_metadata_and_never_a_pass(self):
        owner = self.fake_owner()
        owner.close.side_effect = OSError("cleanup-rejected")
        with tempfile.TemporaryDirectory() as temporary, self.owner_patch(lambda command: owner):
            log = Path(temporary) / "run.log"
            status = run_logged_command.run(log, ["fake"], timeout=1, label="run cleanup")
            self.assertEqual(status, 125)
            record = json.loads(log.with_suffix(".log.json").read_text())
            self.assertEqual(record["status"], "failed")
            self.assertIn("cleanup-rejected", " ".join(record["cleanup_errors"]))

    def test_failed_terminate_and_close_cannot_leave_the_reader_waiting_forever(self):
        release = threading.Event()

        class BlockedPipe:
            def read(self, size):
                release.wait(30)
                return b""

            def close(self):
                release.wait(30)

        owner = self.fake_owner()
        owner.process.stdout = BlockedPipe()
        owner.terminate.side_effect = OSError("terminate-rejected")
        owner.close.side_effect = OSError("close-rejected")
        with tempfile.TemporaryDirectory() as temporary, self.owner_patch(lambda command: owner):
            log = Path(temporary) / "run.log"
            start = time.monotonic()
            try:
                status = run_logged_command.run(log, ["fake"], timeout=0.01, label="run broken cleanup")
            finally:
                release.set()
            self.assertLess(time.monotonic() - start, 4)
            self.assertEqual(status, 125)
            record = json.loads(log.with_suffix(".log.json").read_text())
            errors = " ".join(record["cleanup_errors"])
            self.assertIn("terminate-rejected", errors)
            self.assertIn("close-rejected", errors)

    def test_darwin_empty_group_requires_positive_bounded_evidence(self):
        cases = [("", 0, True), ("123 Z\n", 0, False), ("999 S\n", 0, False),
                 ("123 S\n", 0, True), ("", 1, True), ("unparseable", 0, True)]
        for output, code, rejected in cases:
            with self.subTest(output=output, code=code):
                owner = object.__new__(run_logged_command.PosixGroup)
                owner.terminated, owner.exited = False, True
                owner.process = mock.Mock(pid=123)
                result = subprocess.CompletedProcess([], code, stdout=output)
                with mock.patch.object(run_logged_command.sys, "platform", "darwin"), \
                        mock.patch.object(run_logged_command.os, "killpg", side_effect=PermissionError("denied"), create=True), \
                        mock.patch.object(run_logged_command.subprocess, "run", return_value=result) as probe:
                    if rejected:
                        with self.assertRaises((OSError, ValueError)):
                            owner.terminate()
                    else:
                        owner.terminate()
                        self.assertEqual(probe.call_args.kwargs["timeout"], 2)
                        self.assertEqual(probe.call_args.args[0],
                                         ["/bin/ps", "-A", "-o", "pgid=", "-o", "stat="])

    def test_windows_profile_requires_real_native_python_ownership(self):
        with tempfile.TemporaryDirectory() as temporary:
            log = Path(temporary) / "run.log"
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--timeout", "2", str(log),
                 sys.executable, "-c", "print('native-probe')"],
                env=dict(os.environ, RUNNER_OS="Windows"), capture_output=True, timeout=6)
            record = json.loads(log.with_suffix(".log.json").read_text())
            if os.name == "nt":
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(record["owner"], "windows-job")
            else:
                self.assertEqual(result.returncode, 125, result.stderr)
                self.assertNotIn(b"native-probe", result.stdout)
                self.assertIn("native host Python", record["error"])

    def test_default_native_pipe_keeps_bytes_and_exit_status(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            payload = ("stdout 中文\r\n".encode("utf-8") * 100000)
            source = root / "payload"
            source.write_bytes(payload)
            result = subprocess.run(
                [sys.executable, str(SCRIPT), str(root / "plain.log"), sys.executable,
                 "-c", "import pathlib,sys; sys.stdout.buffer.write(pathlib.Path(sys.argv[1]).read_bytes()); sys.exit(23)",
                 str(source)], capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 23, result.stderr)
            self.assertEqual(result.stdout, payload)
            self.assertEqual((root / "plain.log").read_bytes(), payload)

    def tree_command(self, root, parent_exits):
        beat = root / "heartbeat"
        child = ("import pathlib,time; p=pathlib.Path(" + repr(str(beat)) + "); "
                 "exec('while True:\\n with p.open(\"ab\",buffering=0) as f: f.write(b\".\")\\n time.sleep(0.02)')")
        parent = ("import subprocess,sys,time; subprocess.Popen([sys.executable,'-c'," +
                  repr(child) + "]); print('command-live',flush=True); " +
                  ("" if parent_exits else "time.sleep(30)"))
        return [sys.executable, "-c", parent], beat

    def assert_stopped(self, beat):
        self.assertTrue(beat.exists(), "grandchild must actually have run")
        size = beat.stat().st_size
        time.sleep(0.15)
        self.assertEqual(beat.stat().st_size, size, "owned grandchild survived termination")

    def check_tree_timeout(self, parent_exits):
        with tempfile.TemporaryDirectory(prefix="owned command ") as temporary:
            root = Path(temporary)
            command, beat = self.tree_command(root, parent_exits)
            log = root / "run.log"
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--timeout", "1", "--label", "run probe", str(log), *command],
                capture_output=True, timeout=6)
            self.assertEqual(result.returncode, 124, result.stderr)
            self.assertIn(b"command-live", result.stdout)
            self.assertEqual(log.read_bytes(), result.stdout)
            record = json.loads((root / "run.log.json").read_text())
            self.assertEqual((record["label"], record["status"], record["exit_code"]),
                             ("run probe", "failed", 124))
            self.assertEqual(record["reason"], "timed out")
            self.assert_stopped(beat)

    def test_deadline_kills_owned_grandchild(self):
        self.check_tree_timeout(False)

    def test_deadline_still_applies_after_parent_exits_with_inherited_pipe(self):
        self.check_tree_timeout(True)

    def test_success_also_closes_descendants_with_redirected_output(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            beat, child_pid = root / "heartbeat", root / "pid"
            child = f"""import os,pathlib,time
pathlib.Path({str(child_pid)!r}).write_text(str(os.getpid()))
p=pathlib.Path({str(beat)!r})
while True:
    with p.open('ab',buffering=0) as output:
        output.write(b'.')
    time.sleep(0.02)
"""
            parent = ("import pathlib,subprocess,sys,time; "
                      f"subprocess.Popen([sys.executable,'-c',{child!r}], stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL); "
                      f"p=pathlib.Path({str(beat)!r}); "
                      "exec('while not p.exists(): time.sleep(0.01)'); print('parent-complete')")
            result = subprocess.run(
                [sys.executable, str(SCRIPT), "--timeout", "3", str(root / "run.log"),
                 sys.executable, "-c", parent], capture_output=True, timeout=7)
            try:
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn(b"parent-complete", result.stdout)
                self.assert_stopped(beat)
            finally:
                if beat.exists():
                    previous = beat.stat().st_size
                    time.sleep(0.05)
                    if beat.stat().st_size != previous:
                        os.kill(int(child_pid.read_text()), 9)

    def test_cancel_closes_owned_tree_without_a_false_pass(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            command, beat = self.tree_command(root, False)
            log = root / "run.log"
            with subprocess.Popen([sys.executable, str(SCRIPT), "--timeout", "10", str(log), *command],
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE) as process:
                deadline = time.monotonic() + 4
                while not beat.exists() and time.monotonic() < deadline:
                    time.sleep(0.02)
                self.assertTrue(beat.exists(), "child did not reach its actual execution")
                process.terminate()
                stdout, stderr = process.communicate(timeout=4)
                self.assertNotEqual(process.returncode, 0, stdout + stderr)
            self.assertNotEqual(json.loads((root / "run.log.json").read_text())["status"], "passed")
            self.assert_stopped(beat)

    def test_invalid_deadline_never_starts_command(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            marker = root / "unexpected"
            for value in ("0", "-1", "nan", "inf"):
                with self.subTest(value=value):
                    result = subprocess.run(
                        [sys.executable, str(SCRIPT), "--timeout", value, str(root / "run.log"),
                         sys.executable, "-c", f"import pathlib; pathlib.Path({str(marker)!r}).touch()"],
                        capture_output=True, timeout=3)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertFalse(marker.exists())


class WindowsJobContractTests(unittest.TestCase):
    def kernel(self, assignment=1):
        kernel = mock.Mock()
        kernel.CreateJobObjectW.return_value = 101
        kernel.CreateEventW.return_value = 202
        kernel.SetInformationJobObject.return_value = 1
        kernel.AssignProcessToJobObject.return_value = assignment
        kernel.SetEvent.return_value = 1
        return kernel

    def patches(self, kernel):
        return mock.patch.multiple(windows_job.ctypes, WinDLL=mock.Mock(return_value=kernel),
                                   WinError=mock.Mock(return_value=OSError("job rejected")),
                                   get_last_error=mock.Mock(return_value=5), create=True)

    def test_gate_releases_only_after_assignment_and_handle_scope_closes(self):
        kernel = self.kernel()
        process = mock.Mock(_handle=303)
        order = []
        kernel.AssignProcessToJobObject.side_effect = lambda *args: order.append("assigned") or 1
        kernel.SetEvent.side_effect = lambda *args: order.append("released") or 1
        with self.patches(kernel), mock.patch.object(windows_job.os, "set_handle_inheritable", create=True), \
                mock.patch.object(windows_job.subprocess, "STARTUPINFO", create=True,
                                  side_effect=lambda: types.SimpleNamespace()), \
                mock.patch.object(windows_job.subprocess, "Popen", return_value=process) as spawn:
            owner = windows_job.WindowsJob(["tool.exe", "argument with spaces"])
            self.assertEqual(order, ["assigned", "released"])
            self.assertEqual(spawn.call_args.kwargs["startupinfo"].lpAttributeList, {"handle_list": [202]})
            self.assertTrue(spawn.call_args.kwargs["close_fds"])
            self.assertEqual(json.loads(spawn.call_args.args[0][-1]), ["tool.exe", "argument with spaces"])
            owner.close()
            owner.close()
        self.assertEqual(kernel.CloseHandle.call_args_list, [mock.call(202), mock.call(101)])
        kernel.SetInformationJobObject.assert_called_once()

    def test_assignment_failure_never_releases_or_falls_back_to_unowned_command(self):
        kernel = self.kernel(assignment=0)
        process = mock.Mock(_handle=303)
        with self.patches(kernel), mock.patch.object(windows_job.os, "set_handle_inheritable", create=True), \
                mock.patch.object(windows_job.subprocess, "STARTUPINFO", create=True,
                                  side_effect=lambda: types.SimpleNamespace()), \
                mock.patch.object(windows_job.subprocess, "Popen", return_value=process) as spawn:
            with self.assertRaises(OSError):
                windows_job.WindowsJob(["tool.exe"])
        kernel.SetEvent.assert_not_called()
        self.assertEqual(spawn.call_count, 1)
        process.kill.assert_called_once()
        process.wait.assert_called_once()
        process.stdout.close.assert_called_once()
        self.assertCountEqual(kernel.CloseHandle.call_args_list, [mock.call(202), mock.call(101)])

    def test_close_failure_is_reported_and_handle_is_not_forgotten(self):
        kernel = self.kernel()
        process = mock.Mock(_handle=303)
        with self.patches(kernel), mock.patch.object(windows_job.os, "set_handle_inheritable", create=True), \
                mock.patch.object(windows_job.subprocess, "STARTUPINFO", create=True,
                                  side_effect=lambda: types.SimpleNamespace()), \
                mock.patch.object(windows_job.subprocess, "Popen", return_value=process):
            owner = windows_job.WindowsJob(["tool.exe"])
            kernel.CloseHandle.return_value = 0
            with self.assertRaises(OSError):
                owner.close()
            self.assertEqual(owner.job, 101)


if __name__ == "__main__":
    unittest.main()

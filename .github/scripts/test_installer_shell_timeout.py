"""Exercise the real installer harness with a shell that ignores SIGTERM."""

import os
from pathlib import Path
import shlex
import shutil
import signal
import subprocess
import tempfile
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parents[2]


class InstallerShellTimeoutTests(unittest.TestCase):
    def test_missing_timeout_is_a_failure_not_a_skip(self):
        case = InstallerShellTimeoutTests("test_stuck_shell_is_killed_and_reports_its_output")
        result = unittest.TestResult()
        with mock.patch("shutil.which", return_value=None):
            case.run(result)
        self.assertEqual(len(result.failures), 1)
        self.assertEqual(result.errors, [])
        self.assertEqual(result.skipped, [])
        self.assertIn("GNU timeout is required", result.failures[0][1])

    def test_stuck_shell_is_killed_and_reports_its_output(self):
        self.assertTrue(shutil.which("timeout") or shutil.which("gtimeout"),
                        "GNU timeout is required")
        with tempfile.TemporaryDirectory(prefix="llgo shell timeout ") as directory:
            temporary = Path(directory)
            pidfile = temporary / "shell.pid"
            shell = temporary / "fish"
            shell.write_text(
                "#!/bin/sh\n"
                "trap '' TERM\n"
                f"echo $$ > {shlex.quote(str(pidfile))}\n"
                "echo STUCK_SHELL_DIAGNOSTIC >&2\n"
                "exec sleep 60\n"
            )
            shell.chmod(0o755)
            environment = dict(os.environ, PATH=f"{temporary}{os.pathsep}{os.environ['PATH']}",
                               LLGO_SHELL_TEST_TIMEOUT="1")
            process = subprocess.Popen(
                ["bash", str(ROOT / "test/installer/install_shell_test.sh"), "fish"],
                env=environment, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
            )
            try:
                stdout, stderr = process.communicate(timeout=10)
                self.assertNotEqual(process.returncode, 0)
                self.assertIn("START fish -ic (default)", stdout)
                self.assertIn("STUCK_SHELL_DIAGNOSTIC", stderr)
                self.assertIn("timeout:", stderr)
                self.assertIn("fish -ic (default)", stderr)
                self.assertIn("startup or validation failed", stderr)
                self.assertNotIn("did not find exactly one installed PATH entry", stderr)
                with self.assertRaises(ProcessLookupError):
                    os.kill(int(pidfile.read_text()), 0)
            finally:
                if pidfile.exists():
                    try:
                        os.kill(int(pidfile.read_text()), signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                if process.poll() is None:
                    process.kill()
                    process.communicate()


if __name__ == "__main__":
    unittest.main()

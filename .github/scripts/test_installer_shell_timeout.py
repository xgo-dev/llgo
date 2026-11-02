"""Exercise the real installer harness with a shell that ignores SIGTERM."""

import os
from pathlib import Path
import shlex
import shutil
import signal
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


class InstallerShellTimeoutTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("timeout") or shutil.which("gtimeout"),
                         "GNU timeout is required")
    def test_stuck_shell_is_killed_and_reports_its_output(self):
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

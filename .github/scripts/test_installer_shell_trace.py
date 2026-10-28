"""Check fish-only startup diagnostics through the real installer harness."""

import os
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
BASH = shutil.which("bash")


class InstallerShellTraceTests(unittest.TestCase):
    def check_shell(self, name, expected_trace):
        with tempfile.TemporaryDirectory(prefix="installer shell trace ") as directory:
            temporary = Path(directory)
            record = temporary / "startup.txt"
            shell = temporary / name
            shell.write_text(
                "#!/bin/sh\n"
                f"printf '%s %s\\n' \"$1\" \"${{fish_trace-unset}}\" >> {shlex.quote(str(record))}\n"
                "printf 'STARTUP_DIAGNOSTIC mode=%s trace=%s\\n' \"$1\" \"${fish_trace-unset}\" >&2\n"
                "if [ \"$1\" = -lic ]; then\n"
                "    echo ORIGINAL_STARTUP_FAILURE >&2\n"
                "    exit 23\n"
                "fi\n"
            )
            shell.chmod(0o755)
            environment = dict(os.environ, PATH=f"{temporary}{os.pathsep}{os.environ['PATH']}")
            # Prove that the clean startup environment, not a caller's trace
            # setting, determines what each shell receives.
            environment["fish_trace"] = "caller-value"
            environment.pop("LLGO_SHELL_TEST_TIMEOUT", None)
            result = subprocess.run(
                [BASH, str(ROOT / "test/installer/install_shell_test.sh"), name],
                env=environment, capture_output=True, text=True, timeout=10,
            )
            self.assertEqual(result.returncode, 1, result.stdout + result.stderr)
            self.assertEqual(record.read_text().splitlines(),
                             [f"-ic {expected_trace}", f"-lic {expected_trace}"])
            self.assertIn("ORIGINAL_STARTUP_FAILURE", result.stderr)
            self.assertIn(f"STARTUP_DIAGNOSTIC mode=-lic trace={expected_trace}", result.stderr)
            self.assertIn("startup or validation failed (exit 23)", result.stderr)
            self.assertNotIn(f"PASS {name} (default)", result.stdout)

    def test_fish_login_and_nonlogin_receive_startup_trace(self):
        self.check_shell("fish", "1")

    def test_other_shell_does_not_receive_fish_trace(self):
        self.check_shell("bash", "unset")


if __name__ == "__main__":
    unittest.main()

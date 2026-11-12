import os
from pathlib import Path
import runpy
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock


SCRIPT = Path(__file__).resolve().parents[1] / "workflows" / "test_demo.sh"
BASH = os.environ.get("LLGO_TEST_BASH") or shutil.which("bash")


class DemoSchedulerTests(unittest.TestCase):
    def setUp(self):
        self.assertIsNotNone(BASH, "required CI tool is missing: bash")

    def run_demos(self, statuses, missing_wait_pid=False):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            compiler = bin_dir / "llgo"
            # Exercise the real harness without compiling Go. Each fake
            # executable keeps its own output and exit status.
            compiler.write_text("""#!/bin/bash
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then output=$2; shift; fi
  shift
done
printf '#!/bin/sh\\nsleep 0.05\\necho child-%s\\nexit %s\\n' "${PWD##*_}" "${PWD##*_}" > "$output"
chmod +x "$output"
""")
            compiler.chmod(0o755)
            for index, status in enumerate(statuses):
                case = root / "_demo" / "go" / f"case{index}_{status}"
                case.mkdir(parents=True)
                (case / "main.go").write_text("package main\n")
            environment = dict(os.environ, PATH=str(bin_dir) + os.pathsep + os.environ["PATH"],
                               LLGO_DEMO_JOBS="4", LLGO_DEMO_OUTPUT_DIR=str(root / "logs"))
            environment.pop("LLGO_DEMO_LLGORUN_FLAGS", None)
            environment.pop("BASH_ENV", None)
            if missing_wait_pid:
                # Model Bash's no-unreported-job result even on Bash
                # versions where wait -n handles completed jobs correctly.
                # Explicit PID waits still execute the real builtin.
                hook = root / "bash-env"
                hook.write_text("""wait() {
  if [ "${1:-}" = -n ]; then
    unset "$3"
    return 127
  fi
  builtin wait "$@"
}
""")
                environment["BASH_ENV"] = str(hook)
            result = subprocess.run([BASH, str(SCRIPT)], cwd=root, env=environment,
                                    text=True, capture_output=True, timeout=5)
            self.assertEqual(result.returncode, int(any(statuses)), result.stdout + result.stderr)
            lines = result.stdout.splitlines()
            for index, status in enumerate(statuses):
                self.assertEqual(lines.count(f"Testing ./_demo/go/case{index}_{status}"), 1)
            self.assertEqual(lines.count("PASS"), statuses.count(0), result.stdout)
            self.assertEqual(lines.count("FAIL"), len(statuses) - statuses.count(0), result.stdout)
            self.assertIn(f"{statuses.count(0)}/{len(statuses)} tests passed", result.stdout)

    def test_missing_wait_pid_preserves_success(self):
        self.run_demos([0, 0, 0], missing_wait_pid=True)

    def test_missing_wait_pid_preserves_failures(self):
        self.run_demos([0, 7, 127], missing_wait_pid=True)

    def test_reported_child_exit_127_is_a_failure(self):
        self.run_demos([0, 7, 127])


class DemoSchedulerDependenciesTests(unittest.TestCase):
    def test_missing_bash_fails_before_starting_a_command(self):
        environment = {key: value for key, value in os.environ.items() if key != "LLGO_TEST_BASH"}
        with mock.patch.dict(os.environ, environment, clear=True), \
                mock.patch.object(shutil, "which", return_value=None), \
                mock.patch.object(subprocess, "run",
                                  side_effect=AssertionError("must fail before starting a command")):
            namespace = runpy.run_path(__file__)
            result = unittest.TestResult()
            namespace["DemoSchedulerTests"]("test_missing_wait_pid_preserves_success").run(result)
            self.assertEqual(result.skipped, [], "missing CI tools must not become skips")
            self.assertEqual(result.errors, [])
            self.assertEqual(len(result.failures), 1)
            self.assertIn("required CI tool is missing: bash", result.failures[0][1])


if __name__ == "__main__":
    unittest.main()

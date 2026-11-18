import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "workflows" / "test_demo.sh"
BASH = os.environ.get("LLGO_TEST_BASH") or shutil.which("bash")


@unittest.skipUnless(BASH, "Bash is required for the demo scheduler")
class DemoSchedulerTests(unittest.TestCase):
    def run_demos(self, statuses, missing_wait_pid=False, cmp_statuses=()):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            bin_dir = root / "bin"
            bin_dir.mkdir()
            compiler = bin_dir / "llgo"
            # Exercise the real harness without compiling Go. Each fake
            # executable keeps its own output and exit status.
            compiler.write_text("""#!/bin/bash
if [ "$1" = cmptest ]; then
  echo comparison-child-${PWD##*_}
  exit "${PWD##*_}"
fi
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
            for index, status in enumerate(cmp_statuses):
                case = root / "_cmptest" / f"_case{index}_{status}"
                case.mkdir(parents=True)
                (case / "main.go").write_text("package main\n")
            support = root / "_cmptest" / "support" / "nested"
            support.mkdir(parents=True)
            (support / "helper.go").write_text("package helper\n")
            environment = dict(os.environ, PATH=str(bin_dir) + os.pathsep + os.environ["PATH"],
                               LLGO_DEMO_JOBS="4")
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
            all_statuses = list(statuses) + list(cmp_statuses)
            self.assertEqual(result.returncode, int(any(all_statuses)), result.stdout + result.stderr)
            lines = result.stdout.splitlines()
            for index, status in enumerate(statuses):
                self.assertEqual(lines.count(f"Testing ./_demo/go/case{index}_{status}"), 1)
            for index, status in enumerate(cmp_statuses):
                self.assertEqual(lines.count(f"Testing ./_cmptest/_case{index}_{status}"), 1)
                self.assertIn(f"comparison-child-{status}", lines)
            self.assertNotIn("Testing ./_cmptest/support", lines)
            self.assertEqual(lines.count("PASS"), all_statuses.count(0), result.stdout)
            self.assertEqual(lines.count("FAIL"), len(all_statuses) - all_statuses.count(0), result.stdout)
            self.assertIn(f"{all_statuses.count(0)}/{len(all_statuses)} tests passed", result.stdout)

    def test_missing_wait_pid_preserves_success(self):
        self.run_demos([0, 0, 0], missing_wait_pid=True)

    def test_missing_wait_pid_preserves_failures(self):
        self.run_demos([0, 7, 127], missing_wait_pid=True)

    def test_reported_child_exit_127_is_a_failure(self):
        self.run_demos([0, 7, 127])

    def test_comparison_failure_fails_the_existing_demo_job(self):
        self.run_demos([0], cmp_statuses=[0, 7, 127])

    def test_comparison_fixtures_run_once_and_skip_support_directories(self):
        self.run_demos([0], cmp_statuses=[0, 0])


if __name__ == "__main__":
    unittest.main()

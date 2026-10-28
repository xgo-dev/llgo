import os
import json
from pathlib import Path
import queue
import shutil
import signal
import subprocess
import tempfile
import threading
import time
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "workflows" / "test_demo.sh"
BASH = os.environ.get("LLGO_TEST_BASH") or shutil.which("bash")


class DemoDeadlineTests(unittest.TestCase):
    def fixture(self, root, phase, timeout="0.2"):
        binary = root / "bin"
        binary.mkdir()
        compiler = binary / "llgo"
        compiler.write_text("""#!/bin/bash
while [ "$#" -gt 0 ]; do
  if [ "$1" = -o ]; then output=$2; shift; fi
  shift
done
echo compiler-live
if [ "${PWD##*/}" = probe ] && [ "$FAKE_PHASE" = build ]; then sleep 30; fi
printf '#!/bin/bash\necho runtime-live\n' > "$output"
if [ "${PWD##*/}" = probe ] && [ "$FAKE_PHASE" = run ]; then
  printf 'sleep 30\n' >> "$output"
fi
chmod +x "$output"
""")
        compiler.chmod(0o755)
        for case in ("fast", "probe"):
            directory = root / "_demo" / "go" / case
            directory.mkdir(parents=True)
            (directory / "main.go").write_text("package main\n")
        environment = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"],
                           LLGO_DEMO_JOBS="2", FAKE_PHASE=phase,
                           LLGO_DEMO_BUILD_TIMEOUT=timeout, LLGO_DEMO_RUN_TIMEOUT=timeout,
                           LLGO_DEMO_OUTPUT_DIR=str(root / "logs"))
        for name in ("LLGO_DEMO_LLGORUN_FLAGS", "BASH_ENV", "RUNNER_OS"):
            environment.pop(name, None)
        return environment

    def stop(self, process):
        if process.poll() is None:
            os.killpg(process.pid, signal.SIGKILL)
        return process.communicate(timeout=3)

    def check_deadline(self, phase):
        self.assertTrue(BASH, "Bash is required")
        with tempfile.TemporaryDirectory(prefix="demo deadline ") as temporary:
            root = Path(temporary)
            environment = self.fixture(root, phase)
            with subprocess.Popen([BASH, str(SCRIPT)], cwd=root, env=environment,
                                  stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  text=True, start_new_session=True) as process:
                try:
                    output, _ = process.communicate(timeout=3)
                except subprocess.TimeoutExpired:
                    output, _ = self.stop(process)
                    self.fail(f"unbounded {phase} case outlived its deadline; original output:\n{output}")
            self.assertNotEqual(process.returncode, 0, output)
            self.assertIn(f"Starting {phase} ./_demo/go/probe", output)
            self.assertIn("timed out", output.lower())
            self.assertIn("1/2 tests passed", output)
            logs = list((root / "logs").rglob(f"{phase}.log"))
            self.assertTrue(logs, "failed stage logs were discarded")
            marker = "compiler-live" if phase == "build" else "runtime-live"
            self.assertTrue(any(marker in log.read_text() for log in logs))

    def test_stuck_build_is_bounded_and_retains_build_log(self):
        self.check_deadline("build")

    def test_stuck_run_is_bounded_and_retains_run_log(self):
        self.check_deadline("run")

    def test_embedded_cold_budget_is_independent_of_native_build_budget(self):
        self.assertTrue(BASH, "Bash is required")
        with tempfile.TemporaryDirectory(prefix="demo embedded budget ") as temporary:
            root = Path(temporary)
            environment = self.fixture(root, "none", timeout="0.01")
            compiler = root / "bin" / "llgo"
            compiler.write_text("#!/bin/bash\necho embedded-live\nsleep 0.04\n")
            for category, names in (("c", ("catomic", "hello", "qsort")),
                                    ("go", ("ifaceconv", "linkname", "cabi", "issue1538"))):
                for name in names:
                    case = root / "_demo" / category / name
                    case.mkdir(parents=True)
                    (case / "main.go").write_text("package main\n")
            environment.update(LLGO_DEMO_JOBS="1", LLGO_DEMO_EMBEDDED_TIMEOUT="1")
            result = subprocess.run([BASH, str(SCRIPT), "--embedded"], cwd=root,
                                    env=environment, text=True, capture_output=True, timeout=8)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn("11/11 tests passed", result.stdout)
            records = [json.loads(path.read_text()) for path in (root / "logs").rglob("*.json")]
            self.assertEqual(len(records), 11)
            self.assertTrue(all(record["timeout_seconds"] == 1 for record in records))

    def test_running_case_exposes_phase_and_output_before_completion(self):
        self.assertTrue(BASH, "Bash is required")
        with tempfile.TemporaryDirectory(prefix="demo live ") as temporary:
            root = Path(temporary)
            environment = self.fixture(root, "run", timeout="4")
            with subprocess.Popen([BASH, str(SCRIPT)], cwd=root, env=environment,
                                  stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  text=True, start_new_session=True) as process:
                lines = queue.Queue()
                reader = threading.Thread(target=lambda: [lines.put(line) for line in process.stdout], daemon=True)
                reader.start()
                observed = ""
                deadline = time.monotonic() + 2
                try:
                    while "Starting run ./_demo/go/probe" not in observed or observed.count("runtime-live") < 2:
                        remaining = deadline - time.monotonic()
                        if remaining <= 0:
                            self.fail("active case phase/output remained buffered:\n" + observed)
                        try:
                            observed += lines.get(timeout=remaining)
                        except queue.Empty:
                            self.fail("active case phase/output remained buffered:\n" + observed)
                    self.assertIsNone(process.poll(), observed)
                finally:
                    if process.poll() is None:
                        os.killpg(process.pid, signal.SIGTERM)
                    process.wait(timeout=3)
                    reader.join(timeout=3)
                    self.assertFalse(reader.is_alive(), "canceled case retained its output pipe")
                    probe = [json.loads(path.read_text()) for path in (root / "logs").rglob("run.log.json")
                             if "probe" in json.loads(path.read_text())["label"]]
                    self.assertEqual(len(probe), 1)
                    self.assertEqual(probe[0].get("reason"), "canceled by signal 15")


if __name__ == "__main__":
    unittest.main()

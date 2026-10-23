import contextlib
import io
import json
import os
from pathlib import Path
import queue
import subprocess
import sys
import tempfile
import threading
import unittest

import run_go_tests


class GoTestReportTests(unittest.TestCase):
    def invoke(self, lines, status=0):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        summary = root / "summary.md"
        command = [sys.executable, "-c", "import sys; print(" +
                   repr("\n".join(lines)) + f"); sys.exit({status})"]
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            result = run_go_tests.run(command, root, "build", summary)
        return root, result, output.getvalue(), summary.read_text()

    def test_preserves_package_identity_nested_cases_and_null_timings(self):
        events = [
            {"Package": "p", "Test": "TestSame/sub|case", "Action": "pass", "Elapsed": 2},
            {"Package": "p", "Test": "TestSame", "Action": "pass", "Elapsed": 3},
            {"Package": "q", "Test": "TestSame", "Action": "skip"},
            {"Package": "p", "Action": "pass", "Elapsed": 4},
        ]
        root, status, _, summary = self.invoke([json.dumps(event) for event in events])
        self.assertEqual(status, 0)
        cases = json.loads((root / "build-times.json").read_text())
        self.assertEqual([(case["Package"], case["Test"]) for case in cases],
                         [("p", "TestSame"), ("p", "TestSame/sub|case"), ("q", "TestSame")])
        self.assertIn("Slowest tests", summary)
        self.assertIn("Slowest subtests", summary)
        self.assertIn("sub&#124;case", summary)
        self.assertEqual(len((root / "build.jsonl").read_text().splitlines()), 4)

    def test_prints_every_case_beyond_the_summary_limit(self):
        events = [{"Package": "p", "Test": f"TestCase{i:02}", "Action": "pass", "Elapsed": i}
                  for i in range(20)]
        root, status, output, summary = self.invoke([json.dumps(event) for event in events])
        self.assertEqual(status, 0)
        timings = (root / "build-times.txt").read_text()
        self.assertIn(timings, output)
        self.assertIn("Parent times include subtests", timings)
        self.assertLess(timings.index("TestCase19"), timings.index("TestCase00"))
        for event in events:
            self.assertIn(event["Test"], timings)
        self.assertNotIn("TestCase00", summary)

    def test_failure_status_and_plain_diagnostics_reach_the_outer_wrapper(self):
        events = [{"Action": "output", "Output": "fatal error: test failure\n"},
                  {"Package": "p", "Test": "TestBroken", "Action": "fail", "Elapsed": 5}]
        root, status, output, summary = self.invoke([json.dumps(event) for event in events], 7)
        self.assertEqual(status, 7)
        self.assertTrue(output.startswith("fatal error: test failure\n"))
        self.assertIn("5.00\tfail\tp\tTestBroken", output)
        self.assertIn("exit 7", summary)
        self.assertEqual(json.loads((root / "build-times.json").read_text())[0]["Action"], "fail")

    def test_unstructured_tool_output_is_visible(self):
        _, status, output, _ = self.invoke(["compiler diagnostic", "[1, 2]"])
        self.assertEqual(status, 0)
        self.assertTrue(output.startswith("compiler diagnostic\n[1, 2]\n"))

    def test_cli_preserves_unicode_and_child_status_with_cp1252_stdout(self):
        package, name = "example.com/\u4e2d\u6587\u2215", "TestUnicode/\u00b7\U0001f642"
        diagnostic = "compiler diagnostic: \u4e2d\u6587 \u2215 \U0001f642\n"
        script = Path(run_go_tests.__file__).resolve()
        for status in (0, 23):
            for timings_only in (False, True):
                with self.subTest(status=status, timings_only=timings_only), \
                        tempfile.TemporaryDirectory() as temporary:
                    root = Path(temporary)
                    terminal = {"Package": package, "Test": name, "Elapsed": 1.25,
                                "Action": "fail" if status else "pass"}
                    lines = [] if timings_only else [
                        json.dumps({"Action": "output", "Output": diagnostic}, ensure_ascii=False),
                        diagnostic.rstrip("\n"), json.dumps([diagnostic], ensure_ascii=False)]
                    lines.append(json.dumps(terminal, ensure_ascii=False))
                    payload = ("\n".join(lines) + "\n").encode("utf-8")
                    source = root / "events.jsonl"
                    source.write_bytes(payload)
                    summary = root / "summary.md"
                    child = ("import pathlib,sys; "
                             "sys.stdout.buffer.write(pathlib.Path(sys.argv[1]).read_bytes()); "
                             "sys.exit(int(sys.argv[2]))")
                    result = subprocess.run(
                        [sys.executable, str(script), "--output-dir", str(root), "build",
                         sys.executable, "-c", child, str(source), str(status)],
                        capture_output=True, timeout=15,
                        env={**os.environ, "PYTHONIOENCODING": "cp1252:strict", "PYTHONUTF8": "0",
                             "GITHUB_STEP_SUMMARY": str(summary)})
                    self.assertEqual(result.returncode, status, result.stderr)
                    rendered = result.stdout.decode("utf-8")
                    if not timings_only:
                        self.assertTrue(rendered.startswith(diagnostic + diagnostic))
                        self.assertIn(json.dumps([diagnostic], ensure_ascii=False), rendered)
                    self.assertEqual((root / "build.jsonl").read_bytes(), payload)
                    self.assertEqual(json.loads((root / "build-times.json").read_text(encoding="utf-8")),
                                     [terminal])
                    timings = (root / "build-times.txt").read_text(encoding="utf-8")
                    self.assertIn(timings, rendered)
                    self.assertIn(f"1.25\t{terminal['Action']}\t{package}\t{name}", timings)
                    report = summary.read_text(encoding="utf-8")
                    self.assertIn(f"exit {status}", report)
                    self.assertIn(package, report)
                    self.assertIn(name, report)

    def test_cli_preserves_unicode_command_diagnostics_with_cp1252_stderr(self):
        with tempfile.TemporaryDirectory() as temporary:
            diagnostic = "\u4e2d\u6587\u2215\U0001f642"
            result = subprocess.run(
                [sys.executable, str(Path(run_go_tests.__file__).resolve()),
                 "--output-dir", temporary, "build", sys.executable, "-c", "pass", diagnostic],
                capture_output=True, timeout=15,
                env={**os.environ, "PYTHONIOENCODING": "cp1252:strict", "PYTHONUTF8": "0"})
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn(diagnostic, result.stderr.decode("utf-8"))

    def test_native_log_pipe_preserves_large_output_stderr_and_exit_status(self):
        repo = Path(__file__).resolve().parents[2]
        payload = ("native output \u4e2d\u6587\r\n".encode("utf-8") * 400000)
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, log = root / "source", root / "log"
            source.write_bytes(payload)
            child = ("import pathlib,sys; "
                     f"sys.stdout.buffer.write(pathlib.Path({str(source)!r}).read_bytes()); "
                     "sys.stdout.buffer.flush(); "
                     "sys.stderr.buffer.write(b'final diagnostic\\n'); sys.exit(23)")
            result = subprocess.run(
                [sys.executable, str(repo / "dev/run_logged_command.py"), str(log),
                 sys.executable, "-c", child], capture_output=True, timeout=15)
            self.assertEqual(result.returncode, 23, result.stderr)
            self.assertEqual(result.stdout, payload + b"final diagnostic\n")
            self.assertEqual(log.read_bytes(), result.stdout)

    def test_progress_and_json_log_are_visible_before_child_exits(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            release = root / "release"
            event = json.dumps({"Action": "output", "Output": "probe running\n"})
            child = ("import pathlib,time; "
                     f"print({event!r},flush=True); "
                     f"release=pathlib.Path({str(release)!r}); "
                     "exec('while not release.exists(): time.sleep(0.01)')")
            script = Path(run_go_tests.__file__).resolve()
            command = [sys.executable, str(script), "--output-dir", str(root),
                       "build", sys.executable, "-c", child]
            with subprocess.Popen(command, stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE, text=True) as process:
                lines = queue.Queue()
                reader = threading.Thread(target=lambda: lines.put(process.stdout.readline()))
                reader.start()
                try:
                    self.assertEqual(lines.get(timeout=5), "probe running\n")
                    self.assertIsNone(process.poll())
                    self.assertEqual((root / "build.jsonl").read_text().strip(), event)
                finally:
                    release.touch()
                    reader.join(timeout=5)
                    stdout, stderr = process.communicate(timeout=5)
                self.assertEqual(process.returncode, 0, stdout + stderr)

    def test_windows_quarantine_still_rejects_unrelated_failures_after_json_rendering(self):
        repo = Path(__file__).resolve().parents[2]
        for extra in ("", "--- FAIL: TestRealFailure (0.01s)\n"):
            with self.subTest(extra=extra), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                go = root / "go"
                go.write_text(f"#!{sys.executable}\nprint('github.com/xgo-dev/llgo')\n")
                go.chmod(0o755)
                diagnostic = (
                    "fatal error: receive on synctest channel from outside bubble\n"
                    "testing.(*T).Run\nFAIL github.com/xgo-dev/llgo/test/runtime 0.1s\n" + extra)
                event = json.dumps({"Action": "output", "Output": diagnostic})
                command = ["bash", str(repo / "dev/go_test_windows.sh"), sys.executable,
                           str(repo / ".github/scripts/run_go_tests.py"), "main", sys.executable,
                           "-c", f"import sys; print({event!r}); sys.exit(1)"]
                output = root / "outputs"
                output.touch()
                result = subprocess.run(command, cwd=root, capture_output=True, text=True,
                                        env={**os.environ, "PATH": str(root) + os.pathsep + os.environ["PATH"],
                                             "GITHUB_OUTPUT": str(output),
                                             "GITHUB_STEP_SUMMARY": str(root / "summary")})
                self.assertEqual(result.returncode, 1 if extra else 0, result.stderr)
                self.assertEqual("windows_runtime_corruption=true" in output.read_text(), not extra)


if __name__ == "__main__":
    unittest.main()

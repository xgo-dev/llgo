import argparse
import contextlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import run_build_tests


class BuildShardsTest(unittest.TestCase):
    def test_native_atomic_acceptance_requires_exact_run_pass_and_actual_host(self):
        self.assertEqual(run_build_tests.ARM64_ATOMIC_TESTS, (
            "TestRuntimeAtomicARM64ActualWrappersMatchOriginalGo",
            "TestRuntimeAtomicARM64OriginalSDKTests",
        ))
        names = list(run_build_tests.ARM64_ATOMIC_TESTS)
        events = [{"Package": run_build_tests.PACKAGE, "Test": name, "Action": action}
                  for name in names for action in ("run", "pass")]
        # Unrelated SDK alignment and helper SKIPs retain their original meaning.
        events.append({"Package": run_build_tests.PACKAGE, "Test": "TestUnaligned64",
                       "Action": "skip"})
        for machine in ("arm64", "aarch64"):
            self.assertTrue(run_build_tests.arm64_atomic_result(machine, names, events)["success"])
        variants = {
            "skip": events[:1] + [{**events[1], "Action": "skip"}] + events[2:],
            "fail": events[:1] + [{**events[1], "Action": "fail"}] + events[2:],
            "missing-terminal": events[:1] + events[2:],
            "no-run": events[1:],
            "duplicate-run": events[:1] + events,
            "duplicate-terminal": events[:2] + events[1:],
            "terminal-before-run": [events[1], events[0]] + events[2:],
            "foreign-package": [{**event, "Package": "another/package"} for event in events],
            "child-only": [{**event, "Test": event["Test"] + "/child"} for event in events],
        }
        for label, changed in variants.items():
            with self.subTest(case=label):
                result = run_build_tests.arm64_atomic_result("arm64", names, changed)
                self.assertFalse(result["success"])
        for machine, selected in (("x86_64", names), ("arm64", names[1:]),
                                  ("arm64", names + names[:1])):
            with self.subTest(machine=machine, selected=selected):
                result = run_build_tests.arm64_atomic_result(machine, selected, events)
                self.assertFalse(result["success"])

    def test_native_atomic_gate_uses_the_same_drained_suite_report(self):
        # Fake only the subprocess boundary: no Go or LLVM command runs here.
        # The real runner still lists, partitions, drains and writes its report.
        for terminal in ("pass", "skip"):
            with self.subTest(terminal=terminal), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                names = list(run_build_tests.ARM64_ATOMIC_TESTS) + ["TestOptionalHelper"]
                args = argparse.Namespace(workers=2, gomaxprocs=2, overlay="", ldflags="",
                                          require_arm64_atomic=True)
                commands = []

                def start(command, **unused):
                    commands.append(command)
                    profile = next(word.split("=", 1)[1] for word in command
                                   if word.startswith("-coverprofile="))
                    Path(profile).write_text("mode: set\np/a.go:1.1,2.2 1 1\n")
                    selected = next(word for word in command if word.startswith("-run="))
                    events = []
                    for name in names:
                        if name not in selected:
                            continue
                        outcome = terminal if name in run_build_tests.ARM64_ATOMIC_TESTS else "skip"
                        events += [{"Package": run_build_tests.PACKAGE,
                                    "Test": name, "Action": action}
                                   for action in ("run", outcome)]
                    process = mock.MagicMock()
                    process.__enter__.return_value = process
                    process.stdout = iter(json.dumps(event) + "\n" for event in events)
                    process.wait.return_value = 0
                    return process

                with mock.patch.object(run_build_tests, "__file__", str(root / ".github/scripts/runner.py")), \
                        mock.patch.object(run_build_tests.platform, "machine", return_value="arm64"), \
                        mock.patch.object(run_build_tests.subprocess, "check_output",
                                          side_effect=["\n".join(names), b""]), \
                        mock.patch.object(run_build_tests.subprocess, "Popen", side_effect=start), \
                        mock.patch.dict(os.environ, {"RUNNER_TEMP": directory}), \
                        contextlib.redirect_stdout(io.StringIO()):
                    status = run_build_tests.run(args)
                self.assertEqual(status, int(terminal != "pass"))
                self.assertEqual(len(commands), 2)  # The gate does not rerun either root.
                report = json.loads((root / "go-test-timings/build-shards.json").read_text())
                self.assertEqual(len(report["results"]), 2)
                self.assertTrue(all(result["status"] == 0 for result in report["results"]))
                self.assertEqual(report["native_arm64_atomic"]["success"], terminal == "pass")
                for actions in report["native_arm64_atomic"]["actions"].values():
                    self.assertEqual(actions, ["run", terminal])

    def test_partition_keeps_existing_and_new_tests_exactly_once(self):
        names = [f"TestCase{i}" for i in range(100)] + ["ExampleBuild", "FuzzInputs", "BenchmarkGo126New"]
        groups = run_build_tests.partition(names, 2)
        self.assertEqual(sorted(sum(groups, [])), sorted(names))
        extended = run_build_tests.partition(names + ["TestNewRegression"], 2)
        for index, group in enumerate(groups):
            self.assertEqual(group, [name for name in extended[index] if name != "TestNewRegression"])
        with self.assertRaises(ValueError):
            run_build_tests.partition(["TestSame", "TestSame"], 2)
        self.assertEqual(run_build_tests.selection([]), "^$")

    def test_coverage_union_retains_zeros_and_rejects_missing_blocks(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            first, second, output = [root / name for name in ("first", "second", "output")]
            first.write_text("mode: set\np/a.go:1.1,2.2 2 1\np/a.go:3.1,4.2 3 0\np/b.go:1.1,2.2 1 0\n")
            second.write_text("mode: set\np/a.go:1.1,2.2 2 0\np/a.go:3.1,4.2 3 1\np/b.go:1.1,2.2 1 0\n")
            run_build_tests.merge_coverage([first, second], output)
            self.assertEqual(output.read_text(),
                             "mode: set\np/a.go:1.1,2.2 2 1\np/a.go:3.1,4.2 3 1\np/b.go:1.1,2.2 1 0\n")
            for bad in ("mode: count\n", "mode: set\np/a.go:1.1,2.2 2 1\n",
                        second.read_text().replace("4.2 3", "4.2 4")):
                second.write_text(bad)
                with self.assertRaises(ValueError):
                    run_build_tests.merge_coverage([first, second], output)

    @unittest.skipUnless(shutil.which("go") and shutil.which("git"), "requires Go and Git")
    def test_real_go_tests_keep_helpers_coverage_and_failures(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve() / "repo with spaces"
            root.mkdir()
            script = root / ".github/scripts/run_build_tests.py"
            script.parent.mkdir(parents=True)
            shutil.copy2(run_build_tests.__file__, script)
            package = root / "internal/build"
            package.mkdir(parents=True)
            (root / "go.mod").write_text("module github.com/xgo-dev/llgo\n\ngo 1.24\n")
            (package / "fixture.go").write_text("""package build
func First() int { return 11 }
func Second() int { return 22 }
func ChildOnly() int { return 33 }
func Uncovered() int { return 44 }
var Config = "original"
""")
            # Pick names on different shards so both test environments exercise
            # the same relative file without interfering with each other.
            names = [next(f"TestCase{i}" for i in range(100)
                          if run_build_tests.shard_index(f"TestCase{i}", 2) == index)
                     for index in range(2)]
            (package / "fixture_test.go").write_text("""package build
import (
 "os"
 "os/exec"
 "path/filepath"
 "testing"
 "time"
)
func lock(t *testing.T) {
 f, err := os.OpenFile("same-output", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
 if err != nil { t.Fatal(err) }
 f.Close()
 t.Cleanup(func() { os.Remove("same-output") })
 wd, err := os.Getwd()
 if err != nil { t.Fatal(err) }
 if filepath.Clean(wd) != filepath.Join(os.Getenv("LLGO_ROOT"), "internal", "build") { t.Fatal("wrong LLGO_ROOT") }
 time.Sleep(50*time.Millisecond)
}
func FIRST(t *testing.T) {
 lock(t)
 t.Log("Unicode output: 中文")
 if Config != "configured value" { t.Fatal(Config) }
 if First() != 11 { t.Fatal("first") }
 cmd := exec.Command(os.Args[0], "-test.run=^TestHelper$", "-test.gocoverdir="+os.Getenv("GOCOVERDIR"))
 cmd.Env = append(os.Environ(), "SHARD_HELPER=1")
 if out, err := cmd.CombinedOutput(); err != nil { t.Fatalf("helper: %v %s", err, out) }
}
func SECOND(t *testing.T) {
 lock(t)
 if Second() != 22 { t.Fatal("second") }
 if os.Getenv("SHARD_FAIL") == "1" { t.Fatal("intentional failure") }
}
func TestHelper(t *testing.T) {
 if os.Getenv("SHARD_HELPER") != "1" { t.Skip("subprocess only") }
 if ChildOnly() != 33 { t.Fatal("child") }
}
""".replace("FIRST", names[0]).replace("SECOND", names[1]), encoding="utf-8")
            unused = root / "unused"
            unused.mkdir()
            (unused / "unused.go").write_text("package unused\nfunc Never() int { return 99 }\n")
            for command in (["git", "init", "-q"], ["git", "add", "."]):
                subprocess.run(command, cwd=root, check=True, capture_output=True)
            runner_temp = root / "runner temp"
            runner_temp.mkdir()
            overlay = root / "runtime overlay.json"
            overlay.write_text('{"Replace":{}}')
            ldflags = "-X 'github.com/xgo-dev/llgo/internal/build.Config=configured value'"
            environment = {**os.environ, "GOFLAGS": "", "LLGO_ROOT": str(root),
                           "RUNNER_TEMP": str(runner_temp), "PYTHONIOENCODING": "cp1252"}
            baseline = subprocess.run(["go", "test", "-count=1", "-coverpkg=./...",
                                       "-covermode=set", "-coverprofile=baseline.txt",
                                       "-overlay=" + str(overlay), "-ldflags=" + ldflags,
                                       "./internal/build"], cwd=root, env=environment,
                                      capture_output=True, text=True, timeout=120)
            self.assertEqual(baseline.returncode, 0, baseline.stdout + baseline.stderr)
            results = []
            for workers, failing in ((1, False), (2, False), (2, True)):
                result = subprocess.run([sys.executable, str(script), "--workers", str(workers),
                                         "--overlay", str(overlay), "--ldflags=" + ldflags],
                                        cwd=root, env={**environment, "SHARD_FAIL": str(int(failing))},
                                        capture_output=True, text=True, encoding="utf-8", timeout=120)
                self.assertEqual(result.returncode, int(failing), result.stdout + result.stderr)
                events = [json.loads(line) for line in result.stdout.splitlines() if line.startswith("{")]
                self.assertTrue(any("Unicode output: 中文" in event.get("Output", "") for event in events))
                outcomes = [event["Test"] for event in events
                            if event.get("Test") and event.get("Action") in ("pass", "skip", "fail")]
                self.assertCountEqual(outcomes, names + ["TestHelper"])
                if not failing:
                    results.append((root / "coverage-build.txt").read_text())
            self.assertEqual(results[0], results[1])
            self.assertEqual(sorted((root / "baseline.txt").read_text().splitlines()),
                             sorted(results[1].splitlines()))
            # ChildOnly is executed only in a re-execed helper. It must still
            # reach the profile via cmd/go's GOCOVERDIR, while unused code stays.
            self.assertIn("fixture.go:4.", results[1])
            child = next(line for line in results[1].splitlines() if "fixture.go:4." in line)
            self.assertTrue(child.endswith(" 1"), child)
            unused = next(line for line in results[1].splitlines() if "fixture.go:5." in line)
            self.assertTrue(unused.endswith(" 0"), unused)


if __name__ == "__main__":
    unittest.main()

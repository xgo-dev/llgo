import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import run_build_tests


class BuildShardsTest(unittest.TestCase):
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

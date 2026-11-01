"""Keep worker tests in separate packages without compiling either package twice."""

import json
import os
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "dev/test_wasm_workers.sh"


class WorkerReuseTests(unittest.TestCase):
    def run_worker(self, target, failure=""):
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            trace = directory / "calls.jsonl"
            fake = directory / "fake-tool.py"
            fake.write_text(
                "import json, os, pathlib, sys\n"
                "args = sys.argv[1:]\n"
                "kind = 'isolated' if args[-1].endswith('/numgoroutine') else 'suite'\n"
                "with open(os.environ['TRACE'], 'a') as out:\n"
                "    out.write(json.dumps([kind, args, os.getenv('LLGO_WASM_WORKERS')]) + '\\n')\n"
                "if os.getenv('FAIL_TOOL') == kind: sys.exit(7)\n"
                "print('PASS')\n"
            )
            for name in ("compiler",):
                tool = directory / name
                tool.write_text(f"#!/bin/sh\nexec {shlex.quote(sys.executable)} "
                                f"{shlex.quote(str(fake))} \"$@\"\n")
                tool.chmod(0o755)
            # Load the real function definitions without running the full suite.
            script = SCRIPT.read_text().split('require_tool "${llgo_cmd}"')[0]
            script += '\nrepo_root="$1"\nrun_worker_go_test "$2"\n'
            harness = directory / "worker-test.sh"
            harness.write_text(script)
            result = subprocess.run(
                ["bash", str(harness), str(ROOT), target],
                env={**os.environ, "LLGO": str(directory / "compiler"),
                     "TRACE": str(trace),
                     "FAIL_TOOL": failure},
                capture_output=True, text=True,
            )
            calls = [json.loads(line) for line in trace.read_text().splitlines()]
            return result, calls

    def test_both_profiles_run_each_package_once_in_isolation(self):
        for target in ("emscripten", "emscripten-memory64"):
            with self.subTest(target=target):
                result, calls = self.run_worker(target)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual([call[0] for call in calls], ["suite", "isolated"])
                suite_args, isolated_args = calls[0][1], calls[1][1]
                self.assertEqual(suite_args[-1], str(ROOT / "test/go"))
                self.assertEqual(isolated_args[-1], str(ROOT / "test/go/numgoroutine"))
                self.assertNotIn("-skip", suite_args)
                self.assertNotIn("-run", suite_args)
                self.assertEqual(isolated_args[isolated_args.index("-run") + 1],
                                 "^TestRuntimeNumGoroutineIncludesNewProc$")
                for args in (suite_args, isolated_args):
                    self.assertEqual(args[0], "test")
                    self.assertEqual(args[args.index("-target") + 1], target)
                    self.assertIn("-emulator", args)
                    self.assertIn("-count=1", args)
                self.assertEqual([call[2] for call in calls], ["2", "2"])

    def test_suite_failure_stops_before_isolated_package(self):
        result, calls = self.run_worker("emscripten", "suite")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([call[0] for call in calls], ["suite"])

    def test_isolated_package_failure_fails_acceptance(self):
        result, calls = self.run_worker("emscripten", "isolated")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([call[0] for call in calls], ["suite", "isolated"])


if __name__ == "__main__":
    unittest.main()

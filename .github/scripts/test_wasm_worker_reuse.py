"""Keep worker test isolation without compiling the same package twice."""

import json
import os
from pathlib import Path
import re
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
                "kind = ('compile' if '-emulator' in args else\n"
                "        'gc' if '^TestGCConcurrentTypedMutation$' in args else\n"
                "        'stack' if '^TestEscapeTypedStack' in args else\n"
                "        'uintptr' if '^TestUintptrEscapes' in args else 'run')\n"
                "with open(os.environ['TRACE'], 'a') as out:\n"
                "    out.write(json.dumps([kind, args, os.getenv('LLGO_WASM_WORKERS')]) + '\\n')\n"
                "if os.getenv('FAIL_TOOL') == kind: sys.exit(7)\n"
                "if kind == 'compile': pathlib.Path(args[args.index('-o')+1]).touch()\n"
                "else: assert pathlib.Path(args[1]).is_file()\n"
                "print('PASS')\n"
            )
            for name in ("compiler", "node"):
                tool = directory / name
                tool.write_text(f"#!/bin/sh\nexec {shlex.quote(sys.executable)} "
                                f"{shlex.quote(str(fake))} \"$@\"\n")
                tool.chmod(0o755)
            # Load the real function definitions without running the full suite.
            script = SCRIPT.read_text().split('require_tool "${llgo_cmd}"')[0]
            script += '\nrepo_root="$1"\nrun_worker_go_test "$2"\n'
            result = subprocess.run(
                ["bash", "-c", script, str(SCRIPT), str(ROOT), target],
                env={**os.environ, "LLGO": str(directory / "compiler"),
                     "NODE": str(directory / "node"), "TRACE": str(trace),
                     "FAIL_TOOL": failure},
                capture_output=True, text=True,
            )
            calls = [json.loads(line) for line in trace.read_text().splitlines()]
            return result, calls

    def test_both_profiles_compile_once_and_start_a_fresh_process(self):
        for target in ("emscripten", "emscripten-memory64"):
            with self.subTest(target=target):
                result, calls = self.run_worker(target)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual([call[0] for call in calls], ["compile", "gc", "stack", "uintptr", "run"])
                compile_args, gc_args, stack_args, uintptr_args, run_args = [call[1] for call in calls]
                for args in (gc_args, stack_args, uintptr_args, run_args):
                    self.assertIn(f"{target}-runner.mjs", args[0])
                    self.assertEqual(compile_args[compile_args.index("-o") + 1], args[1])
                self.assertEqual(compile_args[compile_args.index("-skip") + 1],
                                 "^(TestRuntimeNumGoroutineIncludesNewProc|TestGCConcurrentTypedMutation|TestEscapeTypedStack.*|TestUintptrEscapes.*)$")
                self.assertEqual(gc_args[gc_args.index("-test.run") + 1],
                                 "^TestGCConcurrentTypedMutation$")
                self.assertIn("-test.timeout=480s", gc_args)
                self.assertEqual(stack_args[stack_args.index("-test.run") + 1],
                                 "^TestEscapeTypedStack")
                self.assertIn("-test.timeout=480s", stack_args)
                self.assertEqual(uintptr_args[uintptr_args.index("-test.run") + 1],
                                 "^TestUintptrEscapes")
                self.assertIn("-test.timeout=480s", uintptr_args)
                self.assertEqual(run_args[run_args.index("-test.run") + 1],
                                 "^TestRuntimeNumGoroutineIncludesNewProc$")
                self.assertEqual([call[2] for call in calls], ["2", "2", "2", "2", "2"])
                skip = compile_args[compile_args.index("-skip") + 1]
                isolated = [args[args.index("-test.run") + 1]
                            for args in (gc_args, stack_args, uintptr_args, run_args)]
                # Check the real suite inventory: every test still runs exactly
                # once, either in the main process or an isolated stress group.
                for source in (ROOT / "test/go").glob("*_test.go"):
                    for name in re.findall(r"^func (Test\w+)\(.*\*testing\.T", source.read_text(), re.M):
                        runs = int(re.search(skip, name) is None)
                        runs += sum(re.search(pattern, name) is not None for pattern in isolated)
                        self.assertEqual(runs, 1, f"{source.name}: {name}")

    def test_compile_failure_stops_before_isolated_process(self):
        result, calls = self.run_worker("emscripten", "compile")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([call[0] for call in calls], ["compile"])

    def test_isolated_process_failure_fails_acceptance(self):
        stages = ["compile", "gc", "stack", "uintptr", "run"]
        for failure in stages[1:]:
            with self.subTest(failure=failure):
                result, calls = self.run_worker("emscripten", failure)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual([call[0] for call in calls],
                                 stages[:stages.index(failure) + 1])


if __name__ == "__main__":
    unittest.main()

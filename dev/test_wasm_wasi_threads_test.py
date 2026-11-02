import contextlib
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import test_wasm_wasi_threads as check


class MemoryProbeTests(unittest.TestCase):
    def test_positive_and_negative_builds_isolate_linker_environment(self):
        env = dict(os.environ, CCFLAGS="@missing-compiler.rsp", CFLAGS="@missing-cflags.rsp",
                   LDFLAGS="@missing-linker.rsp -Wl,--initial-memory=33554432")
        builds = []

        def run(command, **kwargs):
            if command[0] == check.LLGO:
                actual = kwargs["env"]
                self.assertEqual(actual["CCFLAGS"], "")
                self.assertEqual(actual["CFLAGS"], "")
                self.assertNotIn("@", actual["LDFLAGS"])
                self.assertNotIn("--initial-memory", actual["LDFLAGS"])
                builds.append(command)
                if "too-small.wasm" in " ".join(command):
                    self.assertEqual(actual["LDFLAGS"], "")
                    return subprocess.CompletedProcess(command, 1, "", "initial memory too small")
                return subprocess.CompletedProcess(command, 0, "", "")
            if "print" in command:
                initial, maximum, stack = ((256, 2048, 1 << 20) if len(builds) == 1
                                            else (128, 1024, 2 << 20))
                wat = f'''(module
(import "env" "memory" (memory (;0;) {initial} {maximum} shared))
(global (;0;) i32 (i32.const 0))
(global (;1;) i32 (i32.const {stack}))
(export "__stack_low" (global 0))
(export "__stack_high" (global 1)))'''
                return subprocess.CompletedProcess(command, 0, wat, "")
            return subprocess.CompletedProcess(command, 0, "wasi thread startup ok\n", "")

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "targets").mkdir()
            with patch.object(check, "ROOT", root), patch.object(check.subprocess, "run", side_effect=run), \
                 patch.object(check, "run_inherited_memory_overrides") as inherited, \
                 contextlib.redirect_stdout(io.StringIO()):
                check.run_memory_config_probe(env, root)
            self.assertEqual(len(builds), 3)
            self.assertEqual(inherited.call_args.args[0]["LDFLAGS"], "")
            self.assertEqual(env["CCFLAGS"], "@missing-compiler.rsp")
            self.assertFalse(list((root / "targets").iterdir()))


if __name__ == "__main__":
    unittest.main()

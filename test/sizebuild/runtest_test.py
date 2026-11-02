import os
import subprocess
import sys
import unittest

from runtest import ROOT, command, controlled_environment


class EnvironmentTest(unittest.TestCase):
    def test_inherited_flags_do_not_change_qualification(self):
        inherited = {"PATH": "/tools", "LLVM_CONFIG": "/tools/llvm-config",
                     "WASMOPT": "/tools/wasm-opt", "CCFLAGS": "-DLOCAL",
                     "CFLAGS": "-g", "LDFLAGS": "-Wl,-Map=local.map",
                     "LLGO_LTO_PLUGIN": "/local/plugin", "LLGO_PLAN9ASM_PKGS": "none",
                     "LLGO_ROOT": "/other", "LLGO_BUILD_CACHE": "0"}
        before = inherited.copy()
        actual = controlled_environment(inherited)
        self.assertEqual(actual, {"PATH": "/tools", "LLVM_CONFIG": "/tools/llvm-config",
                                  "WASMOPT": "/tools/wasm-opt", "LLGO_ROOT": str(ROOT),
                                  "LLGO_BUILD_CACHE": "1"})
        self.assertEqual(inherited, before)


class CommandTest(unittest.TestCase):
    def test_timeout_preserves_partial_output(self):
        args = [sys.executable, "-u", "-c",
                "import sys, time; "
                "print('stdout-before-timeout'); "
                "print('stderr-before-timeout', file=sys.stderr); "
                "time.sleep(30)"]
        with self.assertRaises(RuntimeError) as raised:
            command(args, os.environ.copy(), timeout=1)
        error = raised.exception
        self.assertIn(repr(args), str(error))
        self.assertIn("timed out after 1s", str(error))
        self.assertIn("\nstdout-before-timeout\nstderr-before-timeout\n", str(error))
        self.assertIsInstance(error.__cause__, subprocess.TimeoutExpired)


if __name__ == "__main__":
    unittest.main()

# Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

"""Exercise the subprocess bridge without requiring LLDB's Python package."""
import importlib.util
from pathlib import Path
import sys
import types
import unittest

sys.modules['lldb'] = types.SimpleNamespace(eStateStopped=5, LLDB_INVALID_ADDRESS=2**64 - 1)
spec = importlib.util.spec_from_file_location('llgo_wasi', sys.argv.pop(1))
adapter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(adapter)


class Process:
    stop, identity, base, walks = 1, 1, 0x10000, 0

    def IsValid(self): return True
    def GetState(self): return 5
    def GetStopID(self): return self.stop
    def GetUniqueID(self): return self.identity
    def GetSelectedThread(self): return self
    def GetSelectedFrame(self): return self
    def __iter__(self):
        self.walks += 1
        value = types.SimpleNamespace(
            IsValid=lambda: True, GetError=lambda: types.SimpleNamespace(Success=lambda: True),
            GetLoadAddress=lambda: self.base + 1)
        return iter([[types.SimpleNamespace(FindVariable=lambda _: value)]])


class AdapterTest(unittest.TestCase):
    def setUp(self):
        adapter._BASE_CACHE = None
        self.process = Process()
        self.debugger = types.SimpleNamespace(GetSelectedTarget=lambda: types.SimpleNamespace(
            GetProcess=lambda: self.process))
        # Run a real child process. The adapter's transport is independent of
        # the decoder's implementation language.
        adapter.shutil.which = lambda _: sys.executable

    def test_large_stderr_does_not_block_result(self):
        (adapter.DIRECTORY / 'decoder.js').write_text(
            "import sys\nsys.stdin.readline()\n"
            "sys.stderr.write('diagnostic' * 65536)\nsys.stderr.flush()\n"
            "print('{\"result\": {\"ok\": true}}', flush=True)\n", encoding='utf-8')
        self.assertEqual(adapter.inspect(self.debugger, '$goroutines'), {'ok': True})

    def test_failed_child_preserves_bounded_diagnostics(self):
        (adapter.DIRECTORY / 'decoder.js').write_text(
            "import sys\nsys.stdin.readline()\n"
            "sys.stderr.write('decoder failed: ' + 'x' * 131072)\nsys.exit(1)\n", encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'decoder failed:') as error:
            adapter.inspect(self.debugger, '$goroutines')
        self.assertLess(len(str(error.exception)), 66000)

    def test_memory_base_cache_is_scoped_to_process_and_stop(self):
        self.assertEqual(adapter._memory_base(self.process), 0x10000)
        walks = self.process.walks
        self.assertEqual(adapter._memory_base(self.process), 0x10000)
        self.assertEqual(self.process.walks, walks)
        self.process.stop += 1
        self.process.base = 0x20000
        self.assertEqual(adapter._memory_base(self.process), 0x20000)
        self.process.identity += 1
        self.process.base = 0x30000
        self.assertEqual(adapter._memory_base(self.process), 0x30000)


unittest.main()

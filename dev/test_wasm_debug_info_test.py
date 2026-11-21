import unittest
import tempfile
from pathlib import Path
from unittest.mock import patch

import test_wasm_debug_info as check


class LineTableTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.module = Path(directory.name) / 'module.wasm'
        # A custom section before CODE makes its payload start at offset 16.
        self.module.write_bytes(b'\0asm\x01\0\0\0\0\x04\x01x\0\0\x0a\x03\0\0\0')

    def test_uses_the_units_file_index(self):
        table = '''debug_line[0x0]
file_names[  1]:
           name: "other.go"
file_names[  7]:
           name: "/workspace/main.go"
0x0000000000000010     12     23      1   0  is_stmt
0x0000000000000020     12     23      7   0  is_stmt
'''
        with patch.object(check, 'SOURCE_LINES', {'main.go': 12}), \
             patch.object(check, 'run', return_value='main.goProbe\n/workspace/main.go:12\n') as run:
            check.check_source_lines(self.module, table, 'llvm-addr2line')
            self.assertEqual(run.call_count, 1)
            self.assertEqual(run.call_args.args[0][-1], '0x30')

    def test_missing_source_row_still_fails(self):
        table = '''debug_line[0x0]
file_names[  1]:
           name: "other.go"
file_names[  2]:
           name: "main.go"
0x0000000000000010     12     23      1   0  is_stmt
'''
        with patch.object(check, 'SOURCE_LINES', {'main.go': 12}), patch.object(check, 'run') as run:
            with self.assertRaisesRegex(RuntimeError, '0 line-table rows'):
                check.check_source_lines(self.module, table, 'llvm-addr2line')
            run.assert_not_called()


if __name__ == '__main__':
    unittest.main()

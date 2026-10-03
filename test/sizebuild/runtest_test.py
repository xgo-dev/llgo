import os
import subprocess
import sys
import unittest

from runtest import command


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

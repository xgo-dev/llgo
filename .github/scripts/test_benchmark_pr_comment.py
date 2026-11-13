"""Run publisher API behavior tests from the CI policy suite."""
from pathlib import Path
import shutil
import subprocess
import unittest


class BenchmarkCommentTests(unittest.TestCase):
    @unittest.skipUnless(shutil.which("node"), "Node is needed to exercise the publisher")
    def test_github_api_behavior(self):
        result = subprocess.run(
            ["node", "--test", str(Path(__file__).with_name("benchmark_pr_comment.test.js"))],
            capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

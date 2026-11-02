#!/usr/bin/env python3

import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("classify_benchmark_changes.py")
SPEC = importlib.util.spec_from_file_location("classify_benchmark_changes", SCRIPT)
assert SPEC and SPEC.loader
classifier = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = classifier
SPEC.loader.exec_module(classifier)


class ClassifyPathTests(unittest.TestCase):
    def test_representative_categories(self):
        cases = {
            "cl/compile.go": "compiler",
            "cmd/llgo/main.go": "compiler",
            "internal/build/build.go": "compiler",
            "targets/device/riscv64.json": "compiler",
            "go.mod": "compiler",
            "runtime/abi/abi.go": "runtime",
            "runtime/go.mod": "runtime",
            "runtime/_patch/runtime/runtime.go": "stdlib",
            "runtime/internal/lib/reflect/value.go": "stdlib",
            "test/std/fmt.go": "test",
            "test/debug/runtime/main.go": "test",
            "test/debug/runtime/runtest.sh": "test",
            "cl/compile_test.go": "test",
            "runtime/runtime_test.go": "test",
            "runtime/_patch/_test/skipall/main.go": "test",
            "cl/_testgo/foo.go": "test",
            "cl/_testdata/foo.go": "test",
            "benchmark/binary_size/README.txt": "benchmark",
            "_demo/go/hello.go": "example",
            "README.md": "docs",
            "internal/gohex/LICENSE": "docs",
            ".github/workflows/ci.yml": "ci",
            "chore/gentests/main.go": "tooling",
            "CODEOWNERS": "other",
        }
        for path, expected in cases.items():
            with self.subTest(path=path):
                self.assertEqual(classifier.classify_path(path), expected)

    def test_mixed_report_only_sets_present_categories(self):
        report = classifier.build_report(
            [
                classifier.Change("M", ("README.md",)),
                classifier.Change("M", ("cl/compile.go",)),
                classifier.Change("M", ("cl/compile_test.go",)),
            ]
        )
        self.assertTrue(report["categories"]["compiler"])
        self.assertTrue(report["categories"]["docs"])
        self.assertTrue(report["categories"]["test"])
        self.assertFalse(report["categories"]["runtime"])


class GitNameStatusTests(unittest.TestCase):
    def test_rename_checks_old_and_new_paths(self):
        changes = classifier.parse_name_status_z(
            b"R100\0cl/old.go\0doc/new.md\0D\0runtime/old.go\0"
        )
        report = classifier.build_report(changes)
        self.assertEqual(
            report["filesByCategory"]["compiler"], ["cl/old.go"]
        )
        self.assertEqual(report["filesByCategory"]["docs"], ["doc/new.md"])
        self.assertEqual(
            report["filesByCategory"]["runtime"], ["runtime/old.go"]
        )

    def test_incomplete_record_is_rejected(self):
        with self.assertRaises(ValueError):
            classifier.parse_name_status_z(b"R100\0only-one-path\0")


class OutputTests(unittest.TestCase):
    def test_github_output_is_reusable(self):
        report = classifier.build_report(
            [classifier.Change("M", ("cl/compile.go",))]
        )
        with tempfile.NamedTemporaryFile(mode="r+", encoding="utf-8") as output:
            classifier.write_github_output(output.name, report)
            output.seek(0)
            values = dict(line.rstrip().split("=", 1) for line in output)
        self.assertEqual(values["compiler"], "true")
        self.assertEqual(values["runtime"], "false")
        self.assertEqual(values["categories"], "compiler")
        self.assertEqual(values["changed_files"], "1")


if __name__ == "__main__":
    unittest.main()

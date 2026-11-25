import pathlib
import subprocess
import unittest


FILTER = pathlib.Path(__file__).with_name("filter_host_coverage.awk")


class FilterHostCoverageTests(unittest.TestCase):
    def filter_profile(self, profile):
        result = subprocess.run(
            ["awk", "-f", str(FILTER)],
            input=profile,
            text=True,
            capture_output=True,
            check=True,
        )
        return result.stdout

    def test_excludes_entire_test_tree_without_changing_other_records(self):
        keep = [
            "github.com/xgo-dev/llgo/cmd/cl/compile.go:10.2,14.3 3 2\n",
            "github.com/xgo-dev/llgo/ssa/test.go:1.1,3.2 1 0\n",
            "github.com/xgo-dev/llgo/testing/a.go:1.1,3.2 1 1\n",
        ]
        exclude = [
            "github.com/xgo-dev/llgo/test/main.go:1.1,3.2 1 1\n",
            "github.com/xgo-dev/llgo/test/go/pkg/a.go:1.1,3.2 1 2\n",
            "github.com/xgo-dev/llgo/test/std/io/a.go:1.1,3.2 1 0\n",
            "github.com/xgo-dev/llgo/test/goroot/report/main.go:1.1,3.2 1 1\n",
        ]
        for mode in ("set", "count", "atomic"):
            with self.subTest(mode=mode):
                header = f"mode: {mode}\n"
                profile = header + keep[0] + "".join(exclude) + "".join(keep[1:])
                self.assertEqual(self.filter_profile(profile), header + "".join(keep))

    def test_empty_and_header_only_profiles(self):
        self.assertEqual(self.filter_profile(""), "")
        self.assertEqual(self.filter_profile("mode: atomic\n"), "mode: atomic\n")

    def test_all_fixture_records_removed(self):
        profile = "mode: atomic\ngithub.com/xgo-dev/llgo/test/a.go:1.1,2.2 1 9\n"
        self.assertEqual(self.filter_profile(profile), "mode: atomic\n")


if __name__ == "__main__":
    unittest.main()

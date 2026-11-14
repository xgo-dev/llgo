import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from ci_changes import Change, parse_name_status_z
import ci_policy


class DecisionTests(unittest.TestCase):
    def decide(self, *paths):
        return ci_policy.classify([Change("M", (p,)) for p in paths])

    def test_only_prose_can_skip_code(self):
        report = self.decide("README.md", "doc/design/pclntab-linkphase.md",
                             "targets/device/nrf/README.markdown", ".github/windows/WINGET.md")
        self.assertFalse(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_unknown_paths_and_test_fixtures_require_code(self):
        for path in ("new-language/source.xyz", "go.mod", "LICENSE",
                     "assets/image.png", "test/data.txt", "internal/testdata/README.md"):
            with self.subTest(path=path):
                self.assertTrue(self.decide(path)["run_code_ci"])

    def test_readme_sources_and_ci_require_both_checks(self):
        for path in ("doc/_readme/llgo_simple/simple.go", "doc/_readme/go.mod",
                     "doc/_readme/scripts/check_std_cover.sh", ".github/workflows/go.yml",
                     ".github/scripts/ci_policy.py", ".github/actions/setup-go/action.yml"):
            with self.subTest(path=path):
                report = self.decide(path)
                self.assertTrue(report["run_code_ci"])
                self.assertTrue(report["run_doc_checks"])

    def test_ordinary_code_can_skip_document_checks(self):
        report = self.decide("cl/compile.go", "runtime/go.mod")
        self.assertTrue(report["run_code_ci"])
        self.assertFalse(report["run_doc_checks"])

    def test_winget_checks_follow_release_and_installer_changes(self):
        for path in ("install.ps1", "install.sh", ".goreleaser.yaml",
                     ".github/windows/test-release.ps1", "test/installer/install_ps1_test.ps1",
                     ".github/workflows/release-build.yml", ".github/workflows/winget.yml",
                     ".github/workflows/ci-prepare.yml", ".github/scripts/ci_policy.py"):
            with self.subTest(path=path):
                self.assertTrue(self.decide(path)["run_winget_ci"])
        for path in ("cl/compile.go", "runtime/go.mod", "README.md"):
            with self.subTest(path=path):
                self.assertFalse(self.decide(path)["run_winget_ci"])

    def test_mixed_changes_require_both_checks(self):
        report = self.decide("cl/compile.go", "README.md")
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_deletions_renames_and_type_changes_require_both_checks(self):
        cases = [Change("D", ("README.md",)), Change("D", ("_demo/c/hello/main.go",)),
                 Change("R100", ("README.md", "doc/README.md")),
                 Change("R100", ("old.go", "new.md")), Change("T", ("README.md",))]
        for change in cases:
            with self.subTest(change=change):
                report = ci_policy.classify([change])
                self.assertTrue(report["run_code_ci"])
                self.assertTrue(report["run_doc_checks"])
                self.assertEqual([f["path"] for f in report["files"]], list(change.paths))

    def test_empty_diff_is_conservative(self):
        report = ci_policy.classify([])
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])
        self.assertTrue(report["run_winget_ci"])

    def test_main_tags_manual_and_scheduled_runs_do_not_inspect_diff(self):
        with patch.object(ci_policy, "git", side_effect=AssertionError("must not inspect Git")):
            for event in ("push", "schedule", "workflow_dispatch", "release"):
                with self.subTest(event=event):
                    report = ci_policy.prepare(event, {}, "xgo-dev")
                    self.assertTrue(report["run_code_ci"])
                    self.assertTrue(report["run_doc_checks"])
                    self.assertTrue(report["run_winget_ci"])

    def test_runner_owner_is_the_workflow_repository_not_pr_author(self):
        self.assertEqual(json.loads(ci_policy.runners("xgo-dev")["linux_large_runner"]),
                         ["qiniu", "ubuntu-24.04-large"])
        for owner in ("cpunion", "another-fork"):
            with self.subTest(owner=owner):
                for labels in ci_policy.runners(owner).values():
                    self.assertEqual(json.loads(labels), ["ubuntu-24.04"])

    def test_missing_metadata_or_git_failure_cannot_skip_ci(self):
        valid_event = {"pull_request": {"base": {"sha": "a" * 40}, "head": {"sha": "b" * 40}}}
        with contextlib.redirect_stderr(io.StringIO()):
            for event in ({}, {"pull_request": None}, valid_event):
                with patch.object(ci_policy, "git", side_effect=subprocess.CalledProcessError(1, "git")):
                    report = ci_policy.prepare("pull_request", event, "cpunion")
                    self.assertTrue(report["run_code_ci"])
                    self.assertTrue(report["run_doc_checks"])
                    self.assertTrue(report["run_winget_ci"])

    def test_mode_check_failure_cannot_leave_a_skip_decision(self):
        event = {"pull_request": {"base": {"sha": "a" * 40}, "head": {"sha": "b" * 40}}}
        with contextlib.redirect_stderr(io.StringIO()), patch.object(ci_policy, "git", side_effect=[
            b"a" * 40 + b"\n", b"M\0README.md\0", subprocess.CalledProcessError(1, "git")
        ]):
            report = ci_policy.prepare("pull_request", event, "cpunion")
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_paths_with_spaces_newlines_and_rename_are_not_split(self):
        changes = parse_name_status_z(b"R100\0old name.md\0new\nname.md\0M\0a|b.md\0")
        self.assertEqual(changes[0].paths, ("old name.md", "new\nname.md"))
        self.assertEqual(changes[1].paths, ("a|b.md",))


class GitHistoryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.old_cwd = os.getcwd()
        self.addCleanup(os.chdir, self.old_cwd)
        os.chdir(self.temp.name)
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "CI Test")
        self.git("config", "user.email", "ci@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.git("config", "core.filemode", "true")
        self.write("README.md", "base\n")
        self.base = self.commit()
        self.git("checkout", "-q", "-b", "pr")

    def git(self, *args):
        return subprocess.check_output(["git", *args], stderr=subprocess.PIPE).decode().strip()

    def write(self, name, content="test\n"):
        path = Path(name)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)

    def commit(self):
        self.git("add", ".")
        self.git("commit", "-qm", "fixture")
        return self.git("rev-parse", "HEAD")

    def report(self, base=None, head=None):
        event = {"pull_request": {
            "base": {"sha": base or self.base},
            "head": {"sha": head or self.git("rev-parse", "HEAD")},
        }}
        return ci_policy.prepare("pull_request", event, "cpunion")

    def test_docs_only_pr_ignores_unrelated_main_changes(self):
        self.write("README.md", "updated prose\n")
        head = self.commit()
        self.git("checkout", "-q", "main")
        self.write("unrelated.go", "package unrelated\n")
        base = self.commit()
        report = self.report(base, head)
        self.assertEqual(report["pr_merge_base"], self.base)
        self.assertEqual(report["pr_head_sha"], head)
        self.assertFalse(report["run_code_ci"])
        self.assertEqual([f["path"] for f in report["files"]], ["README.md"])

    def test_earlier_code_commit_is_not_hidden_by_last_docs_commit(self):
        self.write("compiler.go", "package compiler\n")
        self.commit()
        self.write("README.md", "last commit only edits docs\n")
        self.commit()
        report = self.report()
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_actual_document_rename_keeps_both_paths(self):
        self.git("mv", "README.md", "renamed.md")
        self.commit()
        report = self.report()
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])
        self.assertEqual({f["path"] for f in report["files"]}, {"README.md", "renamed.md"})

    def test_executable_markdown_requires_code(self):
        Path("README.md").chmod(0o755)
        self.commit()
        self.assertTrue(self.report()["run_code_ci"])

    def test_new_symlink_named_markdown_requires_code(self):
        Path("linked.md").symlink_to("README.md")
        self.commit()
        self.assertTrue(self.report()["run_code_ci"])

    def test_missing_base_does_not_fall_back_to_last_commit(self):
        self.write("compiler.go")
        self.commit()
        self.write("README.md", "updated prose\n")
        self.commit()
        with contextlib.redirect_stderr(io.StringIO()):
            report = self.report("f" * 40)
        self.assertTrue(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_large_pr_is_not_limited_to_first_page_or_path_filter_limit(self):
        for n in range(3100):
            self.write(f"doc/{n:04}.md")
        self.write("zzz/compiler.go")
        self.commit()
        report = self.report()
        self.assertEqual(len(report["files"]), 3101)
        self.assertTrue(report["run_code_ci"])

    def test_large_docs_only_pr_does_not_expand_paths_into_arguments(self):
        for n in range(3100):
            self.write(f"doc/{'long-name-' * 10}{n:04}.md")
        head = self.commit()
        with patch.object(ci_policy, "git", wraps=ci_policy.git) as git:
            report = self.report(head=head)
        self.assertEqual(len(report["files"]), 3100)
        self.assertFalse(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])
        git.assert_any_call("ls-tree", "-r", "-z", "--full-tree", head)
        self.assertEqual(git.call_count, 3)

    def test_tree_scan_ignores_unchanged_executables_and_symlinks(self):
        Path("README.md").chmod(0o755)
        Path("linked.md").symlink_to("README.md")
        base = self.commit()
        self.write("doc/space tab\tnewline\n[literal].md")
        self.commit()
        report = self.report(base=base)
        self.assertFalse(report["run_code_ci"])
        self.assertTrue(report["run_doc_checks"])

    def test_cli_emits_string_outputs_and_a_readable_summary(self):
        self.write("README.md", "updated prose\n")
        head = self.commit()
        event_path = Path(self.temp.name) / "event.json"
        event_path.write_text(json.dumps({"pull_request": {
            "base": {"sha": self.base}, "head": {"sha": head}}}))
        output = Path(self.temp.name) / "outputs"
        summary = Path(self.temp.name) / "summary"
        subprocess.run([os.sys.executable, str(Path(ci_policy.__file__).resolve())], check=True,
                       stdout=subprocess.PIPE, env={**os.environ,
                           "GITHUB_EVENT_NAME": "pull_request", "GITHUB_EVENT_PATH": str(event_path),
                           "GITHUB_REPOSITORY_OWNER": "cpunion", "GITHUB_OUTPUT": str(output),
                           "GITHUB_STEP_SUMMARY": str(summary)})
        values = dict(line.split("=", 1) for line in output.read_text().splitlines())
        self.assertEqual(values["run_code_ci"], "false")
        self.assertEqual(values["run_doc_checks"], "true")
        self.assertEqual(values["pr_head_sha"], head)
        self.assertIn("README.md", summary.read_text())


if __name__ == "__main__":
    unittest.main()

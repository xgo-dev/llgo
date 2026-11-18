#!/usr/bin/env python3
"""Resolve shared CI decisions. Only a fully classified PR may skip checks."""

from __future__ import annotations

import html
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys

from ci_changes import Change, git, parse_name_status_z


DOC_SUFFIXES = {".md", ".markdown", ".rst"}


def is_documentation(path: str) -> bool:
    parts = PurePosixPath(path).parts
    # Markdown fixtures and executable README examples are test inputs.
    if path.startswith("doc/_readme/") or any("testdata" in p for p in parts):
        return False
    return PurePosixPath(path).suffix.lower() in DOC_SUFFIXES


def classify(changes: list[Change]) -> dict:
    files = [
        {"path": path, "status": change.status,
         "category": "documentation" if is_documentation(path) else "code/config"}
        for change in changes for path in change.paths
    ]
    # Deletions/renames may break README links or remove packaged documents.
    # Type changes (including symlinks) and unrecognized statuses are not prose edits.
    structural = any(change.status not in {"A", "M"} for change in changes)
    code = any(f["category"] != "documentation" for f in files)
    docs = any(f["category"] == "documentation" for f in files)
    doc_inputs = any(
        f["path"].startswith(("doc/_readme/", ".github/")) for f in files
    )
    winget_inputs = any(
        f["path"].startswith((".github/windows/", "test/installer/")) or
        f["path"] in {"install.ps1", "install.sh", ".goreleaser.yaml",
                      ".github/workflows/release-build.yml", ".github/workflows/winget.yml",
                      ".github/workflows/ci-prepare.yml", ".github/scripts/ci_policy.py",
                      ".github/scripts/ci_changes.py"}
        for f in files
    )
    return {
        "run_code_ci": not files or structural or code,
        "run_doc_checks": not files or structural or docs or doc_inputs,
        "run_winget_ci": not files or structural or winget_inputs,
        "files": files,
        "reason": "PR changes classified; unknown files require code CI",
    }


def runners(owner: str) -> dict[str, str]:
    return {
        "linux_runner": json.dumps(
            ["qiniu", "ubuntu-24.04"] if owner == "xgo-dev" else ["ubuntu-24.04"]
        ),
        "linux_large_runner": json.dumps(
            ["qiniu", "ubuntu-24.04-large"] if owner == "xgo-dev" else ["ubuntu-24.04"]
        ),
    }


def prepare(event_name: str, event: dict, owner: str) -> dict:
    report = {
        "run_code_ci": True,
        "run_doc_checks": True,
        "run_winget_ci": True,
        "pr_head_sha": "",
        "pr_merge_base": "",
        "files": [],
        "reason": "Non-PR event: all checks enabled",
        **runners(owner),
    }
    if event_name != "pull_request":
        return report
    try:
        pull = event["pull_request"]
        base, head = pull["base"]["sha"], pull["head"]["sha"]
        for sha in (base, head):
            if not isinstance(sha, str) or not re.fullmatch(r"[0-9a-f]{40}", sha):
                raise ValueError("missing or invalid PR commit SHA")
        report["pr_head_sha"] = head
        # Never fall back to HEAD^: an earlier PR commit may contain code.
        merge_base = git("merge-base", base, head).decode().strip()
        report["pr_merge_base"] = merge_base
        changes = parse_name_status_z(
            git("diff", "--name-status", "-z", "--find-renames", merge_base, head)
        )
        report.update(classify(changes))
        if not report["run_code_ci"]:
            # An executable or symlink named *.md is not a prose-only change.
            # Walk the tree once and filter locally: passing every changed path
            # as an argument can exceed ARG_MAX on large docs-only PRs.
            paths = {f["path"].encode("utf-8", errors="surrogateescape")
                     for f in report["files"]}
            entries = git("ls-tree", "-r", "-z", "--full-tree", head)
            if any(entry.split(b"\t", 1)[1] in paths and not entry.startswith(b"100644 ")
                   for entry in entries.split(b"\0") if entry):
                report["run_code_ci"] = True
                report["reason"] = "Documentation includes executable files or symlinks; all checks enabled"
    except (KeyError, TypeError, ValueError, OSError, subprocess.CalledProcessError) as error:
        report["run_code_ci"] = True
        report["run_doc_checks"] = True
        report["run_winget_ci"] = True
        report["reason"] = f"Cannot classify the complete PR; all checks enabled ({type(error).__name__})"
        print(f"::warning::{report['reason']}", file=sys.stderr)
    return report


def write_outputs(path: str, report: dict) -> None:
    with open(path, "a", encoding="utf-8") as output:
        for key in ("run_code_ci", "run_doc_checks", "run_winget_ci", "pr_head_sha", "pr_merge_base",
                    "linux_runner", "linux_large_runner"):
            value = report[key]
            if isinstance(value, bool):
                value = str(value).lower()
            output.write(f"{key}={value}\n")


def write_summary(path: str, report: dict) -> None:
    with open(path, "a", encoding="utf-8") as summary:
        summary.write("## CI policy\n\n")
        summary.write(report["reason"] + ".\n\n")
        for key in ("run_code_ci", "run_doc_checks", "run_winget_ci", "pr_head_sha", "pr_merge_base"):
            summary.write(f"- `{key}`: `{report[key]}`\n")
        if report["files"]:
            summary.write(f"\nChanged paths: {len(report['files'])}\n\n")
            summary.write("| Status | Category | Path |\n| --- | --- | --- |\n")
            for item in report["files"][:100]:
                path_text = html.escape(item["path"]).replace("|", "&#124;")
                path_text = path_text.replace("\n", "\\n").replace("\r", "\\r")
                summary.write(f"| {item['status']} | {item['category']} | <code>{path_text}</code> |\n")
            if len(report["files"]) > 100:
                summary.write("\nShowing the first 100 paths; all paths were classified.\n")


def main() -> None:
    event_name = os.environ["GITHUB_EVENT_NAME"]
    # Main, tags and manual/scheduled runs do not depend on Git diff or PR metadata.
    event = json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text()) if event_name == "pull_request" else {}
    report = prepare(event_name, event, os.environ["GITHUB_REPOSITORY_OWNER"])
    write_outputs(os.environ["GITHUB_OUTPUT"], report)
    write_summary(os.environ["GITHUB_STEP_SUMMARY"], report)
    print(json.dumps(report, indent=2, ensure_ascii=True))


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Classify LLGo repository changes for benchmark workflow decisions.

Each changed path has one primary category.  A commit can therefore contain
multiple categories, while workflows can make decisions from stable boolean
outputs such as ``compiler=true``.
"""

from __future__ import annotations

import argparse
import json
from pathlib import PurePosixPath
import subprocess
import sys
from typing import Iterable, Sequence

from ci_changes import Change, git as _git, parse_name_status_z


CATEGORIES = (
    "compiler",
    "runtime",
    "stdlib",
    "test",
    "benchmark",
    "example",
    "docs",
    "ci",
    "tooling",
    "other",
)

TEST_DIRS = {
    "_cmptest",
    "test",
    "cl/cltest",
    "internal/filecheck",
    "internal/littest",
    "internal/llgen",
    "runtime/_test",
    "runtime/_patch/_test",
    "runtime/internal/test",
    "ssa/ssatest",
}

COMPILER_DIRS = {
    "cl",
    "cmd/internal/base",
    "cmd/internal/build",
    "cmd/internal/clean",
    "cmd/internal/compile",
    "cmd/internal/compilerhash",
    "cmd/internal/flags",
    "cmd/internal/get",
    "cmd/internal/help",
    "cmd/internal/install",
    "cmd/internal/lldb",
    "cmd/internal/monitor",
    "cmd/internal/run",
    "cmd/internal/test",
    "cmd/internal/version",
    "internal",
    "ltoplugin",
    "ssa",
    "targets",
    "xtool",
}


def _under(path: str, directory: str) -> bool:
    return path == directory or path.startswith(directory + "/")


def _under_any(path: str, directories: Iterable[str]) -> bool:
    return any(_under(path, directory) for directory in directories)


def classify_path(raw_path: str) -> str:
    """Return the primary category for a repository-relative path."""
    path = PurePosixPath(raw_path.replace("\\", "/")).as_posix()
    if path.startswith("./"):
        path = path[2:]
    name = PurePosixPath(path).name

    # Put purpose-specific files before their containing source tree.  For
    # example, runtime/foo_test.go is a test rather than a runtime change.
    if _under(path, ".github") or name in {".goreleaser.yml", ".goreleaser.yaml"}:
        return "ci"
    if (
        _under_any(path, {"doc", "docs", "LICENSES"})
        or name.lower().endswith((".md", ".markdown", ".rst"))
        or name in {"LICENSE", "THIRD_PARTY_NOTICES.md"}
    ):
        return "docs"
    if (
        name.endswith("_test.go")
        or "testdata" in PurePosixPath(path).parts
        or _under_any(path, TEST_DIRS)
        or path.startswith("cl/_test")
    ):
        return "test"
    if _under(path, "benchmark"):
        return "benchmark"
    if _under_any(path, {"_demo", "examples"}):
        return "example"

    # runtime/_patch and runtime/internal/lib mirror or replace Go standard
    # library packages.  Keep them distinct from LLGo's runtime support.
    if _under_any(path, {"runtime/_patch", "runtime/internal/lib"}):
        return "stdlib"
    if _under(path, "runtime"):
        return "runtime"

    if path in {"go.mod", "go.sum"} or path.startswith("cmd/llgo/"):
        return "compiler"
    if _under_any(path, COMPILER_DIRS):
        return "compiler"

    if _under_any(path, {"chore", "dev"}) or path == "install.sh":
        return "tooling"
    return "other"


def _valid_commit(revision: str) -> bool:
    if not revision or set(revision) == {"0"}:
        return False
    return subprocess.run(
        ["git", "cat-file", "-e", f"{revision}^{{commit}}"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    ).returncode == 0


def changes_from_git(base: str, head: str) -> list[Change]:
    if not _valid_commit(head):
        raise ValueError(f"head is not a local Git commit: {head}")
    if not _valid_commit(base):
        fallback = f"{head}^"
        if not _valid_commit(fallback):
            raise ValueError(f"base is not a local Git commit: {base}")
        print(
            f"warning: base {base or '<empty>'} is unavailable; using {fallback}",
            file=sys.stderr,
        )
        base = fallback
    return parse_name_status_z(
        _git("diff", "--name-status", "-z", "--find-renames", base, head)
    )


def build_report(changes: Sequence[Change]) -> dict[str, object]:
    files: list[dict[str, str]] = []
    seen: set[str] = set()
    for change in changes:
        for path in change.paths:
            if path in seen:
                continue
            seen.add(path)
            files.append(
                {"path": path, "status": change.status, "category": classify_path(path)}
            )

    by_category = {
        category: [entry["path"] for entry in files if entry["category"] == category]
        for category in CATEGORIES
    }
    return {
        "schemaVersion": 1,
        "categories": {category: bool(by_category[category]) for category in CATEGORIES},
        "filesByCategory": by_category,
        "files": files,
    }


def write_github_output(path: str, report: dict[str, object]) -> None:
    categories = report["categories"]
    assert isinstance(categories, dict)
    selected = [name for name in CATEGORIES if categories[name]]
    files = report["files"]
    assert isinstance(files, list)
    with open(path, "a", encoding="utf-8") as output:
        for category in CATEGORIES:
            output.write(f"{category}={'true' if categories[category] else 'false'}\n")
        output.write(f"categories={','.join(selected)}\n")
        output.write(f"changed_files={len(files)}\n")


def _markdown_cell(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ")


def write_github_summary(path: str, report: dict[str, object]) -> None:
    files_by_category = report["filesByCategory"]
    assert isinstance(files_by_category, dict)
    with open(path, "a", encoding="utf-8") as summary:
        summary.write("## LLGo change classification\n\n")
        summary.write("| Category | Files | Paths |\n")
        summary.write("| --- | ---: | --- |\n")
        for category in CATEGORIES:
            paths = files_by_category[category]
            if paths:
                shown = ", ".join(f"`{_markdown_cell(item)}`" for item in paths[:12])
                if len(paths) > 12:
                    shown += f", and {len(paths) - 12} more"
                summary.write(f"| {category} | {len(paths)} | {shown} |\n")
        compiler = bool(report["categories"]["compiler"])
        summary.write(
            "\nBinary-size and compile-time benchmarks: "
            + ("**triggered**" if compiler else "**not triggered**")
            + ".\n"
        )


def print_text(report: dict[str, object]) -> None:
    files_by_category = report["filesByCategory"]
    assert isinstance(files_by_category, dict)
    for category in CATEGORIES:
        paths = files_by_category[category]
        if not paths:
            continue
        print(f"{category} ({len(paths)}):")
        for path in paths:
            print(f"  {path}")


def parse_args(argv: Sequence[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("paths", nargs="*", help="paths to classify instead of a Git diff")
    parser.add_argument("--base", help="base Git revision")
    parser.add_argument("--head", default="HEAD", help="head Git revision (default: HEAD)")
    parser.add_argument("--format", choices=("text", "json"), default="text")
    parser.add_argument("--github-output", help="append reusable outputs to this file")
    parser.add_argument("--github-summary", help="append a Markdown summary to this file")
    args = parser.parse_args(argv)
    if args.paths and args.base:
        parser.error("paths and --base cannot be used together")
    if not args.paths and not args.base:
        parser.error("provide paths or --base")
    return args


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    try:
        changes = (
            [Change(status="M", paths=(path,)) for path in args.paths]
            if args.paths
            else changes_from_git(args.base, args.head)
        )
        report = build_report(changes)
    except (OSError, subprocess.CalledProcessError, ValueError) as error:
        print(f"classification failed: {error}", file=sys.stderr)
        return 2

    if args.format == "json":
        json.dump(report, sys.stdout, indent=2, sort_keys=True)
        print()
    else:
        print_text(report)
    if args.github_output:
        write_github_output(args.github_output, report)
    if args.github_summary:
        write_github_summary(args.github_summary, report)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Publish build coverage only after the independent inventory and all parts agree."""

import argparse
import hashlib
import json
from pathlib import Path
import subprocess

from run_build_tests import (PACKAGE, arm64_atomic_result, merge_coverage,
                             outcome_errors, partition, source_identity,
                             root_event, event_root, shard_index)


def read_events(path, owned):
    events, starts, passes = [], 0, 0
    for line in path.read_text(encoding="utf-8").splitlines():
        try:
            event = json.loads(line)
        except ValueError:  # Preserve ordinary tool diagnostics in the artifact.
            continue
        if not isinstance(event, dict):
            continue
        if event.get("Action") in ("fail", "build-fail"):
            raise ValueError("raw Go JSON reports a failure")
        if event.get("Package") == PACKAGE and not event.get("Test"):
            starts += event.get("Action") == "start"
            passes += event.get("Action") == "pass"
        observed = root_event(event)
        if observed:
            events.append(observed)
    if starts != 4 or passes != 4:
        raise ValueError("raw Go JSON lacks four completed package commands")
    errors = outcome_errors(owned, events)
    if errors:
        raise ValueError("; ".join(errors))
    return events


def collect(root, inventory_path, parts_directory, destination, require_arm64_atomic=False):
    # This output belongs to the collector. A prior success must not survive a
    # later failed collection and accidentally be published by another step.
    destination.unlink(missing_ok=True)
    pending = destination.with_suffix(destination.suffix + ".pending")
    pending.unlink(missing_ok=True)
    expected = json.loads(inventory_path.read_text())
    files = subprocess.check_output(["git", "ls-files", "-z"], cwd=root).decode().split("\0")
    if expected["source"] != source_identity(root, [name for name in files if name]):
        raise ValueError("inventory does not match this checkout")
    names, groups = expected["names"], expected["shards"]
    if (expected["schema"] != 1 or expected["parts"] != 2 or expected["workers"] != 2
            or len(groups) != 8 or groups != partition(names, 8) or not names):
        raise ValueError("invalid complete inventory or partition mapping")
    profile = expected["profile"]
    if (profile["covermode"] != "set" or profile["coverpkg"] != "./..."
            or profile["timeout"] != "45m" or profile["benchtime"] != "1x"
            or profile["gomaxprocs"] != 2
            or profile["require_arm64_atomic"] != require_arm64_atomic):
        raise ValueError("unexpected execution profile")
    if require_arm64_atomic and (profile["go"]["GOOS"] != "darwin"
                                 or profile["go"]["GOARCH"] != "arm64"
                                 or profile.get("llvm_version", "").split(".")[0] != "22"):
        raise ValueError("required native profile is not Darwin ARM64 LLVM22")
    reports = sorted(parts_directory.rglob("build-shards.json"))
    if len(reports) != expected["parts"]:
        raise ValueError("missing or duplicate build parts")
    seen_parts, seen_shards, all_events, profiles = set(), set(), [], []
    for path in reports:
        report = json.loads(path.read_text())
        for key in ("schema", "source", "profile", "names", "shards", "parts", "workers"):
            if report[key] != expected[key]:
                raise ValueError(f"part {path}: mismatched {key}")
        part = report["part"]
        if type(part) is not int or part not in range(2) or part in seen_parts:
            raise ValueError("duplicate or invalid part index")
        seen_parts.add(part)
        if report.get("success") is not True:
            raise ValueError("failed or incomplete part")
        owned = list(range(part, 8, 2))
        raw = read_events(path.parent / "build.jsonl",
                          [name for shard in owned for name in groups[shard]])
        results = report["results"]
        if sorted(result["shard"] for result in results) != owned:
            raise ValueError("missing or duplicate logical shard")
        for result in results:
            shard = result["shard"]
            if shard in seen_shards or result["status"] != 0:
                raise ValueError("failed or duplicate logical shard")
            seen_shards.add(shard)
            selected = [name for name in groups[shard] if not name.startswith("Benchmark")]
            if result["selected_tests"] != len(selected) or result["reported_tests"] != len(selected):
                raise ValueError("incomplete selected root count")
            errors = outcome_errors(groups[shard], result["events"])
            if errors:
                raise ValueError("; ".join(errors))
            actual = [event for event in raw if shard_index(event_root(event), 8) == shard]
            if actual != result["events"]:
                raise ValueError("reported events do not match original Go JSON")
            all_events.extend(actual)
        coverage = path.parent / "coverage-build.txt"
        if hashlib.sha256(coverage.read_bytes()).hexdigest() != report["coverage_sha256"]:
            raise ValueError("coverage profile does not match its completed part")
        profiles.append(coverage)
    if seen_parts != {0, 1} or seen_shards != set(range(8)):
        raise ValueError("incomplete global partition")
    errors = outcome_errors(names, all_events)
    if errors:
        raise ValueError("; ".join(errors))
    if require_arm64_atomic:
        native = arm64_atomic_result(profile["machine"], names,
                                     [{"Package": PACKAGE, **event} for event in all_events])
        if not native["success"]:
            raise ValueError("; ".join(native["errors"]))
    try:
        # Reuse the same strict block/statement validation and set-hit OR as
        # the local runner, including zero-hit blocks and child-only coverage.
        merge_coverage(profiles, pending)
        pending.replace(destination)
    finally:
        pending.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--inventory", required=True, type=Path)
    parser.add_argument("--parts-directory", required=True, type=Path)
    parser.add_argument("--output", default="coverage-build.txt", type=Path)
    parser.add_argument("--require-arm64-atomic", action="store_true")
    args = parser.parse_args()
    collect(Path(__file__).resolve().parents[2], args.inventory, args.parts_directory,
            args.output, args.require_arm64_atomic)


if __name__ == "__main__":
    main()

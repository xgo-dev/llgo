#!/usr/bin/env python3
"""Run every internal/build test in isolated processes within one CI job."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
import tempfile
import threading
import time


PACKAGE = "github.com/xgo-dev/llgo/internal/build"
ARM64_ATOMIC_TESTS = (
    "TestRuntimeAtomicARM64ActualWrappersMatchOriginalGo",
    "TestRuntimeAtomicARM64OriginalSDKTests",
)


def arm64_atomic_result(machine, names, events):
    """Require native execution from this suite, not a portable owner or SKIP."""
    actions = {name: [] for name in ARM64_ATOMIC_TESTS}
    for event in events:
        if event.get("Package") == PACKAGE and event.get("Test") in actions:
            actions[event["Test"]].append(event.get("Action"))
    errors = []
    if machine.lower() not in {"arm64", "aarch64"}:
        errors.append(f"native atomic acceptance requires an ARM64 host, got {machine!r}")
    for name, observed in actions.items():
        if names.count(name) != 1 or observed != ["run", "pass"]:
            errors.append(f"{name} requires one listed root and RUN/PASS, got {observed!r}")
    return {"machine": machine, "actions": actions, "errors": errors,
            "success": not errors}


def shard_index(name, workers):
    return int.from_bytes(hashlib.sha256(name.encode()).digest()[:4], "big") % workers


def partition(names, workers):
    if workers < 1 or len(names) != len(set(names)):
        raise ValueError("invalid worker count or duplicate test names")
    groups = [[] for _ in range(workers)]
    for name in names:
        groups[shard_index(name, workers)].append(name)
    return groups


def selection(names):
    # Go test names are identifiers; no regex metacharacters need quoting.
    return "^(" + "|".join(names) + ")$" if names else "^$"


def merge_coverage(profiles, destination):
    """Union set-mode counters from the same instrumented test binary."""
    merged = {}
    for profile in profiles:
        with profile.open(encoding="utf-8") as source:
            if source.readline().strip() != "mode: set":
                raise ValueError(f"expected set coverage in {profile}")
            blocks = {}
            for line in source:
                block, statements, count = line.rstrip().rsplit(" ", 2)
                statements, count = int(statements), int(count)
                if statements < 0 or count not in (0, 1) or block in blocks:
                    raise ValueError(f"invalid coverage block in {profile}: {line}")
                blocks[block] = (statements, count)
        if merged and blocks.keys() != merged.keys():
            raise ValueError(f"coverage blocks differ in {profile}")
        for block, (statements, count) in blocks.items():
            previous, hits = merged.get(block, (statements, 0))
            if previous != statements:
                raise ValueError(f"coverage statement count differs for {block}")
            merged[block] = (statements, max(hits, count))
    if not merged:
        raise ValueError("no coverage blocks produced")
    with destination.open("w", encoding="utf-8") as output:
        output.write("mode: set\n")
        for block, (statements, count) in sorted(merged.items()):
            output.write(f"{block} {statements} {count}\n")


def copy_sources(root, destination, files):
    # Copy the checkout's files, including local modifications. Independent
    # roots also isolate fixtures that publish outputs beside their sources.
    for name in files:
        source, target = root / name, destination / name
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target, follow_symlinks=False)


def exec_argument(words):
    # cmd/go's -exec parser understands quoted words, without shell expansion
    # or backslash escaping. Use forward slashes for Windows file paths.
    if any('"' in word or "\n" in word or "\r" in word for word in words):
        raise ValueError("unsupported quote in test executor argument")
    return " ".join('"' + word + '"' for word in words)


def run(args):
    started = time.monotonic()
    root = Path(__file__).resolve().parents[2]
    output_lock = threading.Lock()

    def emit(line):
        with output_lock:
            sys.stdout.write(line)
            sys.stdout.flush()

    def progress(message):
        emit(json.dumps({"Action": "output", "Package": PACKAGE,
                         "Output": message + "\n"}) + "\n")

    with tempfile.TemporaryDirectory(prefix="llgo-build-tests-",
                                     dir=os.getenv("RUNNER_TEMP")) as temporary:
        temporary = Path(temporary).resolve()
        go_test = ["go", "test", "-count=1", "-timeout=45m",
                   "-coverpkg=./...", "-covermode=set", "-benchtime=1x"]
        if args.overlay:
            go_test.append("-overlay=" + args.overlay)
        if args.ldflags:
            go_test.append("-ldflags=" + args.ldflags)
        # Compile before starting workers so they can reuse the same Go build
        # cache. Keep cmd/go responsible for coverage metadata, helper-process
        # GOCOVERDIR propagation, overlays, JSON output, and test exit status.
        progress("Listing all internal/build tests and preparing their shared build cache")
        listing = subprocess.check_output(go_test + ["-list=.", "./internal/build"],
                                          cwd=root, text=True, encoding="utf-8")
        names = [line for line in listing.splitlines()
                 if line.startswith(("Test", "Example", "Fuzz", "BenchmarkGo126"))]
        if not names:
            raise ValueError("internal/build listed no tests")
        groups = partition(names, args.workers)
        timing_dir = root / "go-test-timings"
        timing_dir.mkdir(exist_ok=True)
        report = {"gomaxprocs_per_worker": args.gomaxprocs, "shards": groups}
        report_path = timing_dir / "build-shards.json"
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        files = subprocess.check_output(["git", "ls-files", "-z"], cwd=root).decode().split("\0")
        files = [name for name in files if name]
        roots = [temporary / f"source-{i}" for i in range(args.workers)]
        profiles = [temporary / f"coverage-{i}.txt" for i in range(args.workers)]
        for source in roots:
            copy_sources(root, source, files)

        def worker(index):
            started = time.monotonic()
            tests = [name for name in groups[index] if not name.startswith("Benchmark")]
            benchmarks = [name for name in groups[index] if name.startswith("Benchmark")]
            executor = exec_argument([Path(sys.executable).as_posix(),
                                      Path(__file__).resolve().as_posix(), "--run-binary",
                                      roots[index].as_posix(), str(args.gomaxprocs)])
            command = go_test + ["-json", "-run=" + selection(tests),
                                 "-bench=" + selection(benchmarks),
                                 "-coverprofile=" + str(profiles[index]),
                                 "-exec=" + executor, "./internal/build"]
            progress(f"Starting build shard {index + 1}/{args.workers}: "
                     f"{len(tests)} tests, {len(benchmarks)} benchmarks")
            environment = {**os.environ, "GOMAXPROCS": str(args.gomaxprocs)}
            outcomes = []
            native_events = []
            with subprocess.Popen(command, cwd=root,
                                  env=environment, stdout=subprocess.PIPE,
                                  stderr=subprocess.STDOUT, text=True,
                                  encoding="utf-8", errors="replace", bufsize=1) as process:
                for line in process.stdout:
                    emit(line)
                    try:
                        event = json.loads(line)
                    except ValueError:
                        continue
                    if (isinstance(event, dict) and event.get("Package") == PACKAGE
                            and event.get("Action") in ("pass", "skip", "fail")
                            and event.get("Test") and "/" not in event["Test"]):
                        outcomes.append(event["Test"])
                    if (args.require_arm64_atomic and isinstance(event, dict)
                            and event.get("Package") == PACKAGE
                            and event.get("Test") in ARM64_ATOMIC_TESTS
                            and event.get("Action") in ("run", "pass", "skip", "fail")):
                        native_events.append({key: event[key]
                                              for key in ("Package", "Test", "Action")})
                status = process.wait()
            if sorted(outcomes) != sorted(tests):
                progress(f"Build shard {index + 1} did not report every selected test exactly once")
                status = 1
            progress(f"Build shard {index + 1} exited with {status}")
            return {"shard": index, "status": status, "seconds": time.monotonic() - started,
                    "reported_tests": len(outcomes), "selected_tests": len(tests),
                    "native_arm64_events": native_events}

        # Drain every worker even on failure; their logs and coverage remain
        # useful, and no compiler may outlive cleanup of its source directory.
        with ThreadPoolExecutor(args.workers) as pool:
            results = list(pool.map(worker, range(args.workers)))
        report.update(results=results, wall_seconds=time.monotonic() - started)
        native_failed = False
        if args.require_arm64_atomic:
            native = arm64_atomic_result(platform.machine(), names,
                                         [event for result in results
                                          for event in result["native_arm64_events"]])
            report["native_arm64_atomic"] = native
            native_failed = not native["success"]
            for error in native["errors"]:
                progress(error)
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        available = [profile for profile in profiles if profile.is_file()]
        if available:
            merge_coverage(available, root / "coverage-build.txt")
        if (native_failed or len(available) != args.workers
                or any(result["status"] for result in results)):
            return 1
    return 0


def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--run-binary":
        root, gomaxprocs, *command = sys.argv[2:]
        directory = Path(root) / "internal/build"
        return subprocess.call(command, cwd=directory,
                               env={**os.environ, "LLGO_ROOT": root,
                                    "PWD": str(directory),
                                    "GOMAXPROCS": gomaxprocs})
    # The parent timing reporter reads UTF-8, including on native Windows
    # where redirected Python stdout can otherwise use the ANSI code page.
    sys.stdout.reconfigure(encoding="utf-8")
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", type=int, choices=(1, 2), default=2)
    parser.add_argument("--gomaxprocs", type=int, default=2)
    parser.add_argument("--overlay", default="")
    parser.add_argument("--ldflags", default="")
    parser.add_argument("--require-arm64-atomic", action="store_true",
                        help="require both native ARM64 atomic tests to RUN and PASS")
    args = parser.parse_args()
    if args.gomaxprocs < 1:
        parser.error("--gomaxprocs must be positive")
    return run(args)


if __name__ == "__main__":
    sys.exit(main())

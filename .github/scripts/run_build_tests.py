#!/usr/bin/env python3
"""Run every internal/build test in isolated processes within one CI job."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import platform
import re
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


def arm64_atomic_result(machine, names, events, required=ARM64_ATOMIC_TESTS):
    """Require native execution from this suite, not a portable owner or SKIP."""
    actions = {name: [] for name in required}
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


def source_identity(root, files):
    digest = hashlib.sha256()
    for name in sorted(files):
        path = root / name
        data = os.readlink(path).encode() if path.is_symlink() else path.read_bytes()
        digest.update(name.encode() + b"\0" + hashlib.sha256(data).digest())
    head = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root,
                                   stderr=subprocess.DEVNULL).decode().strip()
    if not re.fullmatch(r"[0-9a-f]{40}", head):
        raise ValueError("a complete committed checkout identity is required")
    return {"head": head, "tracked_sha256": digest.hexdigest()}


def inventory(root, files, names, args):
    go = json.loads(subprocess.check_output(
        ["go", "env", "-json", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS",
         "CGO_ENABLED", "GOEXPERIMENT", "GOARM64"], cwd=root, text=True))
    profile = {"go": go, "machine": platform.machine(), "gomaxprocs": args.gomaxprocs,
               "overlay_sha256": hashlib.sha256(Path(args.overlay).read_bytes()).hexdigest()
               if args.overlay else None, "ldflags": args.ldflags,
               "coverpkg": "./...", "covermode": "set", "timeout": "45m",
               "benchtime": "1x", "require_arm64_atomic": args.require_arm64_atomic,
               "environment": {name: os.getenv(name, "") for name in (
                   "CC", "CXX", "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS",
                   "SDKROOT", "DEVELOPER_DIR", "MACOSX_DEPLOYMENT_TARGET", "PKG_CONFIG",
                   "PKG_CONFIG_PATH", "PKG_CONFIG_LIBDIR", "ImageOS", "ImageVersion",
                   "LLGO_TEST_EXTENDED_ABI", "LLGO_REQUIRE_MSVC", "LLGO_STDIO_NOBUF")}}
    if args.require_arm64_atomic:
        profile["llvm_version"] = subprocess.check_output(
            [os.getenv("LLVM_CONFIG", "llvm-config"), "--version"], text=True).strip()
    return {"schema": 1, "source": source_identity(root, files), "profile": profile,
            "names": names, "shards": partition(names, args.shards),
            "parts": args.parts, "workers": args.workers}


def event_root(event):
    name = event["Test"]
    return re.sub(r"-\d+$", "", name.split("/", 1)[0]) if event["Action"] == "bench" else name


def root_event(event):
    if not isinstance(event, dict) or event.get("Package") != PACKAGE:
        return None
    # Go's bench action is conditional on a BENCH frame (for example B.Log).
    # The measured result line is emitted with and without that frame. Use it
    # once, rather than double-counting a result plus its optional bench action.
    if event.get("Action") == "output":
        result = re.match(r"^(Benchmark\S+)\s+[0-9]+\s+\S", event.get("Output", ""))
        if result:
            return {"Test": result[1], "Action": "bench"}
    if (event.get("Test") and "/" not in event["Test"]
            and event.get("Action") in ("run", "pass", "skip", "fail")):
        return {key: event[key] for key in ("Test", "Action")}


def outcome_errors(names, events):
    """Test roots need RUN/terminal once; benchmarks use Go's bench events."""
    actions = {name: [] for name in names if not name.startswith("Benchmark")}
    benches = {name: [] for name in names if name.startswith("Benchmark")}
    bench_runs = {name: 0 for name in benches}
    errors = []
    for event in events:
        name, action = event["Test"], event["Action"]
        if action == "bench":
            root = event_root(event)
            if root not in benches or name in benches[root]:
                errors.append(f"unexpected or duplicate benchmark {name}")
            else:
                benches[root].append(name)
        elif name in actions:
            actions[name].append(action)
        elif name in benches and action == "run":
            bench_runs[name] += 1
        else:
            errors.append(f"unexpected root {name}")
    errors.extend(f"incomplete or failed root {name}: {observed}"
                  for name, observed in actions.items()
                  if observed not in (["run", "pass"], ["run", "skip"]))
    errors.extend(f"incomplete benchmark {name}" for name, observed in benches.items()
                  if not observed or bench_runs[name] != 1)
    return errors


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
        args.shards = getattr(args, "shards", None)
        if args.shards is None:
            args.shards = args.workers
        args.parts = getattr(args, "parts", 1)
        args.part = getattr(args, "part", 0)
        groups = partition(names, args.shards)
        indices = list(range(args.part, args.shards, args.parts))
        timing_dir = root / "go-test-timings"
        timing_dir.mkdir(exist_ok=True)
        report = {"gomaxprocs_per_worker": args.gomaxprocs, "shards": groups}
        report_path = timing_dir / "build-shards.json"
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        files = subprocess.check_output(["git", "ls-files", "-z"], cwd=root).decode().split("\0")
        files = [name for name in files if name]
        if args.parts > 1 or getattr(args, "inventory_only", False):
            report.update(inventory(root, files, names, args), part=args.part)
            report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        if getattr(args, "inventory_only", False):
            (timing_dir / "build-inventory.json").write_text(
                json.dumps(report, indent=2) + "\n", encoding="utf-8")
            return 0
        roots = {i: temporary / f"source-{i}" for i in indices}
        profiles = {i: temporary / f"coverage-{i}.txt" for i in indices}
        for source in roots.values():
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
            progress(f"Starting build shard {index + 1}/{args.shards}: "
                     f"{len(tests)} tests, {len(benchmarks)} benchmarks")
            environment = {**os.environ, "GOMAXPROCS": str(args.gomaxprocs)}
            outcomes = []
            root_events = []
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
                    observed = root_event(event)
                    if observed:
                        root_events.append(observed)
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
            errors = outcome_errors(groups[index], root_events)
            if errors:
                progress("; ".join(errors))
                status = 1
            progress(f"Build shard {index + 1} exited with {status}")
            return {"shard": index, "status": status, "seconds": time.monotonic() - started,
                    "reported_tests": len(outcomes), "selected_tests": len(tests),
                    "native_arm64_events": native_events, "events": root_events}

        # Drain every worker even on failure; their logs and coverage remain
        # useful, and no compiler may outlive cleanup of its source directory.
        with ThreadPoolExecutor(args.workers) as pool:
            order = (sorted(indices, key=lambda index: (-len(groups[index]), index))
                     if args.shards > args.workers else indices)
            results = list(pool.map(worker, order))
        report.update(results=results, wall_seconds=time.monotonic() - started)
        native_failed = False
        if args.require_arm64_atomic:
            selected = [name for index in indices for name in groups[index]]
            required = tuple(name for name in ARM64_ATOMIC_TESTS if name in selected)
            native = arm64_atomic_result(platform.machine(), selected,
                                         [event for result in results
                                          for event in result["native_arm64_events"]], required)
            report["native_arm64_atomic"] = native
            native_failed = not native["success"]
            for error in native["errors"]:
                progress(error)
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        available = [profile for profile in profiles.values() if profile.is_file()]
        failed = (native_failed or len(available) != len(indices)
                  or any(result["status"] for result in results))
        if available:
            # Failed partial profiles are evidence, never the published profile.
            destination = timing_dir / ("coverage-build.txt" if not failed else "coverage-incomplete.txt")
            merge_coverage(available, destination)
            report["coverage_sha256"] = hashlib.sha256(destination.read_bytes()).hexdigest()
            if not failed and args.parts == 1:
                shutil.copyfile(destination, root / "coverage-build.txt")
        report["success"] = not failed
        report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
        if failed:
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
    parser.add_argument("--shards", type=int, default=None)
    parser.add_argument("--parts", type=int, default=1)
    parser.add_argument("--part", type=int, default=0)
    parser.add_argument("--inventory-only", action="store_true")
    parser.add_argument("--gomaxprocs", type=int, default=2)
    parser.add_argument("--overlay", default="")
    parser.add_argument("--ldflags", default="")
    parser.add_argument("--require-arm64-atomic", action="store_true",
                        help="require both native ARM64 atomic tests to RUN and PASS")
    args = parser.parse_args()
    if args.gomaxprocs < 1:
        parser.error("--gomaxprocs must be positive")
    if args.shards is None:
        args.shards = args.workers
    if not (1 <= args.parts <= args.shards and 0 <= args.part < args.parts
            and args.shards % args.parts == 0):
        parser.error("require positive, evenly divided shards/parts and a valid part index")
    if args.parts > 1 and args.overlay:
        parser.error("multi-part overlays are not supported; use the unchanged single-job runner")
    return run(args)


if __name__ == "__main__":
    sys.exit(main())

"""Contracts for bounded, complete cross-job internal/build coverage."""

import argparse
import copy
import contextlib
import io
import json
import os
import re
from pathlib import Path
import tempfile
import threading
import time
import unittest
from unittest import mock

import yaml

import run_build_tests
import collect_build_test_parts


class BuildPartsTests(unittest.TestCase):
    def test_invalid_partition_arguments_fail_before_starting_commands(self):
        variants = (["--shards", "0"], ["--shards", "-1"], ["--parts", "0"],
                    ["--shards", "8", "--parts", "3"], ["--part", "1"],
                    ["--shards", "8", "--parts", "2", "--overlay", "replacement.json"])
        for arguments in variants:
            with self.subTest(arguments=arguments), \
                    mock.patch.object(run_build_tests.sys, "argv", ["runner", *arguments]), \
                    mock.patch.object(run_build_tests, "run", side_effect=AssertionError("must not execute")), \
                    contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit) as error:
                run_build_tests.main()
            self.assertEqual(error.exception.code, 2)

    def write_report(self, path, report):
        path.write_text(json.dumps(report))
        raw = []
        for result in report["results"]:
            raw.append({"Package": run_build_tests.PACKAGE, "Action": "start"})
            for event in result["events"]:
                if event["Action"] == "bench":
                    raw.append({"Package": run_build_tests.PACKAGE, "Action": "output",
                                "Output": event["Test"] + "\t1\t1.0 ns/op\n"})
                else:
                    raw.append({"Package": run_build_tests.PACKAGE, **event})
            raw.append({"Package": run_build_tests.PACKAGE, "Action": "pass"})
        path.with_name("build.jsonl").write_text("".join(json.dumps(event) + "\n" for event in raw))

    def parts_fixture(self, root):
        names = ["TestOrdinaryHelper", *run_build_tests.ARM64_ATOMIC_TESTS,
                 "BenchmarkGo126Fixture"] + [f"TestCase{i}" for i in range(12)]
        groups = run_build_tests.partition(names, 8)
        header = {"schema": 1, "source": {"head": "a" * 40, "tracked_sha256": "source"},
                  "profile": {"go": {"GOVERSION": "go1.27.0", "GOOS": "darwin", "GOARCH": "arm64",
                                      "GOFLAGS": "", "CGO_ENABLED": "1", "GOEXPERIMENT": "", "GOARM64": "v8.0"},
                              "machine": "arm64", "llvm_version": "22.1.8",
                              "gomaxprocs": 2, "covermode": "set", "coverpkg": "./...",
                              "timeout": "45m", "benchtime": "1x", "require_arm64_atomic": True,
                              "environment": {"ImageOS": "macos26", "ImageVersion": "20261013.0424.1"}},
                  "names": names, "shards": groups, "parts": 2, "workers": 2}
        inventory = root / "build-inventory.json"
        inventory.write_text(json.dumps(header))
        parts = root / "parts"
        for part in range(2):
            directory = parts / str(part)
            directory.mkdir(parents=True)
            profile = directory / "coverage-build.txt"
            profile.write_text(f"mode: set\np/a.go:1.1,2.2 1 {part}\np/a.go:3.1,4.2 2 {1-part}\np/a.go:5.1,6.2 1 0\n")
            report = {**copy.deepcopy(header), "part": part, "success": True,
                      "coverage_sha256": run_build_tests.hashlib.sha256(profile.read_bytes()).hexdigest(),
                      "results": []}
            for shard in range(part, 8, 2):
                events = []
                tests = [name for name in groups[shard] if not name.startswith("Benchmark")]
                for name in groups[shard]:
                    if name.startswith("Benchmark"):
                        events.append({"Test": name, "Action": "run"})
                        events.append({"Test": name + "-2", "Action": "bench"})
                    else:
                        events += [{"Test": name, "Action": action} for action in
                                   ("run", "skip" if name == "TestOrdinaryHelper" else "pass")]
                report["results"].append({"shard": shard, "status": 0, "selected_tests": len(tests),
                                          "reported_tests": len(tests), "events": events})
            self.write_report(directory / "build-shards.json", report)
        return inventory, parts, header

    def collect(self, root, inventory, parts):
        with mock.patch.object(collect_build_test_parts.subprocess, "check_output", return_value=b""), \
                mock.patch.object(collect_build_test_parts, "source_identity",
                                  return_value={"head": "a" * 40, "tracked_sha256": "source"}):
            collect_build_test_parts.collect(root, inventory, parts, root / "coverage-build.txt", True)

    def test_complete_parts_keep_helper_skips_benchmarks_and_union_zero_blocks(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            inventory, parts, _ = self.parts_fixture(root)
            self.collect(root, inventory, parts)
            self.assertEqual((root / "coverage-build.txt").read_text(),
                             "mode: set\np/a.go:1.1,2.2 1 1\np/a.go:3.1,4.2 2 1\np/a.go:5.1,6.2 1 0\n")

    def test_rolling_image_revision_is_retained_without_changing_profile_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            inventory, parts, _ = self.parts_fixture(root)
            path = parts / "1/build-shards.json"
            report = json.loads(path.read_text())
            report["profile"]["environment"]["ImageVersion"] = "20261021.0445.8"
            self.write_report(path, report)
            self.collect(root, inventory, parts)
            self.assertTrue((root / "coverage-build.txt").is_file())
            self.assertEqual(json.loads(path.read_text())["profile"]["environment"]["ImageVersion"],
                             "20261021.0445.8")
            self.assertEqual(json.loads(inventory.read_text())["profile"]["environment"]["ImageVersion"],
                             "20261013.0424.1")

    def test_image_revision_exception_keeps_semantic_profile_fields_strict(self):
        variants = [("go", "GOVERSION", "go1.27.1"), ("go", "GOOS", "linux"),
                    ("go", "GOARCH", "amd64"), ("go", "GOFLAGS", "-tags=changed"),
                    ("go", "CGO_ENABLED", "0"), ("go", "GOEXPERIMENT", "changed"),
                    ("go", "GOARM64", "v9.0"), ("profile", "llvm_version", "23.0.0"),
                    ("profile", "ldflags", "-X changed"), ("profile", "overlay_sha256", "changed")]
        variants += [("environment", key, "changed") for key in (
            "ImageOS", "CC", "CXX", "CGO_CFLAGS", "CGO_CPPFLAGS", "CGO_CXXFLAGS",
            "CGO_LDFLAGS", "SDKROOT", "DEVELOPER_DIR", "MACOSX_DEPLOYMENT_TARGET",
            "PKG_CONFIG", "PKG_CONFIG_PATH", "PKG_CONFIG_LIBDIR", "LLGO_TEST_EXTENDED_ABI")]
        for group, key, value in variants:
            with self.subTest(group=group, key=key), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                inventory, parts, _ = self.parts_fixture(root)
                path = parts / "1/build-shards.json"
                report = json.loads(path.read_text())
                report["profile"]["environment"]["ImageVersion"] = "20261021.0445.8"
                changed = report["profile"] if group == "profile" else report["profile"][group]
                changed[key] = value
                self.write_report(path, report)
                with self.assertRaisesRegex(ValueError, "mismatched profile"):
                    self.collect(root, inventory, parts)
                self.assertFalse((root / "coverage-build.txt").exists())

    def test_incomplete_duplicate_failed_stale_or_trimmed_parts_never_publish(self):
        mutations = ("missing-part", "duplicate-part", "failed-part", "duplicate-shard",
                     "missing-shard", "wrong-part", "missing-terminal", "duplicate-run",
                     "native-skip", "failed-root", "missing-benchmark", "duplicate-benchmark",
                     "changed-profile", "stale-parts", "trimmed-parts", "stale-inventory",
                     "missing-block", "changed-statements", "wrong-covermode", "profile-hash",
                     "raw-missing-terminal", "raw-package-fail", "raw-empty", "raw-duplicate-root")
        for mutation in mutations:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                inventory, parts, header = self.parts_fixture(root)
                paths = sorted(parts.glob("*/build-shards.json"))
                reports = [json.loads(path.read_text()) for path in paths]
                if mutation == "missing-part":
                    paths[1].unlink()
                elif mutation == "duplicate-part":
                    extra = parts / "extra"; extra.mkdir()
                    (extra / "build-shards.json").write_text(paths[0].read_text())
                elif mutation == "failed-part":
                    reports[0]["success"] = False
                elif mutation == "duplicate-shard":
                    reports[0]["results"].append(reports[0]["results"][0])
                elif mutation == "missing-shard":
                    reports[0]["results"].pop()
                elif mutation == "wrong-part":
                    reports[1]["part"] = 0
                elif mutation in ("missing-terminal", "duplicate-run", "native-skip", "failed-root",
                                  "missing-benchmark", "duplicate-benchmark"):
                    target = ("BenchmarkGo126Fixture-2" if "benchmark" in mutation
                              else run_build_tests.ARM64_ATOMIC_TESTS[0])
                    result = next(result for report in reports for result in report["results"]
                                  if any(event["Test"] == target for event in result["events"]))
                    index = next(i for i, event in enumerate(result["events"])
                                 if event["Test"] == target)
                    if mutation in ("missing-terminal", "missing-benchmark"):
                        result["events"].pop(index + (mutation == "missing-terminal"))
                    elif mutation in ("duplicate-run", "duplicate-benchmark"):
                        result["events"].insert(index, result["events"][index].copy())
                    else:
                        result["events"][index+1]["Action"] = "skip" if mutation == "native-skip" else "fail"
                elif mutation == "changed-profile":
                    reports[1]["profile"]["go"]["GOFLAGS"] = "-tags=changed"
                elif mutation in ("stale-parts", "stale-inventory"):
                    for report in reports:
                        report["source"]["head"] = "b" * 40
                    if mutation == "stale-inventory":
                        header["source"]["head"] = "b" * 40
                        inventory.write_text(json.dumps(header))
                elif mutation == "trimmed-parts":
                    for report in reports:
                        report["names"] = report["names"][1:]
                        report["shards"] = run_build_tests.partition(report["names"], 8)
                elif mutation.startswith("raw-"):
                    pass
                else:
                    profile = paths[1].parent / "coverage-build.txt"
                    data = profile.read_text()
                    if mutation == "missing-block":
                        data = "\n".join(data.splitlines()[:-1]) + "\n"
                    elif mutation == "changed-statements":
                        data = data.replace("4.2 2", "4.2 3")
                    elif mutation == "wrong-covermode":
                        data = data.replace("mode: set", "mode: count")
                    profile.write_text(data)
                    reports[1]["coverage_sha256"] = ("wrong" if mutation == "profile-hash" else
                        run_build_tests.hashlib.sha256(profile.read_bytes()).hexdigest())
                for path, report in zip(paths, reports):
                    if path.exists():
                        self.write_report(path, report)
                if mutation.startswith("raw-"):
                    path = paths[0].with_name("build.jsonl")
                    raw = [json.loads(line) for line in path.read_text().splitlines()]
                    if mutation == "raw-empty":
                        raw = []
                    elif mutation == "raw-package-fail":
                        next(event for event in raw if event["Action"] == "pass"
                             and not event.get("Test"))["Action"] = "fail"
                    else:
                        index = next(i for i, event in enumerate(raw)
                                     if event.get("Test") and event["Action"] ==
                                     ("pass" if mutation == "raw-missing-terminal" else "run"))
                        if mutation == "raw-missing-terminal":
                            raw.pop(index)
                        else:
                            raw.insert(index, raw[index].copy())
                    path.write_text("".join(json.dumps(event) + "\n" for event in raw))
                # Even a stale previous success cannot be reused after failure.
                (root / "coverage-build.txt").write_text("stale output")
                with self.assertRaises((ValueError, OSError)):
                    self.collect(root, inventory, parts)
                self.assertFalse((root / "coverage-build.txt").exists())
                self.assertFalse((root / "coverage-build.txt.pending").exists())

    def test_logical_shards_prevent_cumulative_command_timeout(self):
        # Exercise the real listing/partition/worker/coverage boundary. The
        # external process models a cumulative budget, not an individual hang:
        # one root fits; batching several roots exceeds the original deadline.
        names = [next(f"TestCase{i}" for i in range(1000)
                      if run_build_tests.shard_index(f"TestCase{i}", 8) == shard)
                 for shard in range(8)]
        for part in range(2):
            with self.subTest(part=part), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                args = argparse.Namespace(workers=2, shards=8, parts=2, part=part,
                                          gomaxprocs=2, overlay="", ldflags="",
                                          require_arm64_atomic=False)
                commands = []
                active = 0
                peak = 0
                lock = threading.Lock()

                def output(command, **unused):
                    if "-list=." in command:
                        return "\n".join(names)
                    if command[:2] == ["git", "ls-files"]:
                        return b""
                    if command[:2] == ["git", "rev-parse"]:
                        return b"a" * 40 + b"\n"
                    if command[:2] == ["go", "env"]:
                        return json.dumps({"GOVERSION": "go1.27.0", "GOOS": "darwin",
                                           "GOARCH": "arm64", "GOFLAGS": "",
                                           "CGO_ENABLED": "1"})
                    raise AssertionError(command)

                def start(command, **unused):
                    nonlocal active, peak
                    with lock:
                        active += 1
                        peak = max(peak, active)
                    commands.append(command)
                    selector = next(word for word in command if word.startswith("-run="))
                    selected = [name for name in names if re.fullmatch(selector[5:], name)]
                    profile = next(word.split("=", 1)[1] for word in command
                                   if word.startswith("-coverprofile="))
                    Path(profile).write_text("mode: set\np/a.go:1.1,2.2 1 1\n")
                    fit = len(selected) <= 1
                    events = [{"Package": run_build_tests.PACKAGE, "Test": name,
                               "Action": action}
                              for name in selected for action in
                              (("run", "pass") if fit else ("run",))]
                    process = mock.MagicMock()
                    process.__enter__.return_value = process
                    def lines():
                        time.sleep(0.01)  # Give the other allowed worker a chance to start.
                        yield from (json.dumps(event) + "\n" for event in events)

                    def wait():
                        nonlocal active
                        with lock:
                            active -= 1
                        return 0 if fit else 124

                    process.stdout = lines()
                    process.wait.side_effect = wait
                    return process

                with mock.patch.object(run_build_tests, "__file__", str(root / ".github/scripts/runner.py")), \
                        mock.patch.object(run_build_tests.subprocess, "check_output", side_effect=output), \
                        mock.patch.object(run_build_tests.subprocess, "Popen", side_effect=start), \
                        mock.patch.dict(os.environ, {"RUNNER_TEMP": directory}), \
                        contextlib.redirect_stdout(io.StringIO()):
                    status = run_build_tests.run(args)
                self.assertEqual(status, 0, "logical shards must bound each cumulative command")
                self.assertEqual(len(commands), 4)
                self.assertEqual(peak, 2)
                self.assertEqual(active, 0)
                self.assertTrue(all("-timeout=45m" in command for command in commands))
                selected = [next(word for word in command if word.startswith("-run="))
                            for command in commands]
                owned = [name for name in names
                         if run_build_tests.shard_index(name, 8) % 2 == part]
                self.assertCountEqual([name for name in names
                                       for selector in selected if re.fullmatch(selector[5:], name)], owned)

    def test_macos_build_work_is_isolated_without_reducing_inventory(self):
        path = Path(__file__).resolve().parents[1] / "workflows/go.yml"
        workflow = yaml.load(path.read_text(), Loader=yaml.BaseLoader)
        jobs = workflow["jobs"]
        self.assertIn("macos-build", jobs)
        self.assertIn("macos-build-coverage", jobs)
        build = jobs["macos-build"]
        self.assertEqual(build["timeout-minutes"], "90")
        self.assertEqual(build["strategy"]["matrix"]["part"], ["0", "1"])
        self.assertNotIn("max-parallel", build["strategy"])
        self.assertEqual(build["runs-on"], "macos-latest")
        body = "\n".join(step.get("run", "") for step in build["steps"])
        self.assertIn("--shards 8 --workers 2 --parts 2", body)
        self.assertIn("--require-arm64-atomic", body)
        inventory = next(step for step in jobs["test"]["steps"]
                         if step.get("name") == "List the complete macOS build inventory")
        self.assertIn("--inventory-only --shards 8 --workers 2 --parts 2", inventory["run"])
        self.assertNotIn("-run=", inventory["run"])
        self.assertNotIn("-skip", inventory["run"])
        upload = next(step for step in build["steps"] if step.get("uses", "").startswith("actions/upload-artifact@"))
        self.assertEqual(upload["with"]["name"], "go-build-macos-part-${{ matrix.part }}")
        collector = jobs["macos-build-coverage"]
        self.assertEqual(collector["needs"], ["prepare", "test", "macos-build"])
        commands = [step.get("run", "") for step in collector["steps"]]
        self.assertIn('test "${{ needs.macos-build.result }}" = success', "\n".join(commands))
        self.assertTrue(any("collect_build_test_parts.py" in command for command in commands))
        publish = next(step for step in collector["steps"] if step.get("uses", "").startswith("codecov/"))
        self.assertNotIn("if", publish)
        self.assertNotIn("continue-on-error", collector)
        self.assertNotIn("continue-on-error", publish)
        core = next(step["run"] for step in jobs["test"]["steps"]
                    if step.get("id") == "test_coverage")
        self.assertIn('if [[ "$RUNNER_OS" != macOS ]]; then', core)
        for owner in ("run_main_suite", "run_compiler_suites"):
            self.assertIn(owner, core)
        later = [step.get("name") for step in jobs["test"]["steps"]]
        self.assertIn("Test traceback formatter with coverage", later)
        self.assertIn("Test with embedded emulator env", later)


if __name__ == "__main__":
    unittest.main()

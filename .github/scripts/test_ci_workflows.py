"""Guard the integration contract so workflow edits cannot silently bypass policy."""

import json
import os
from pathlib import Path
import runpy
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import yaml

import ci_policy


WORKFLOWS = Path(__file__).resolve().parents[1] / "workflows"
PREPARE = "./.github/workflows/ci-prepare.yml"
CODE_WORKFLOWS = {
    "llgo.yml": 40, "go.yml": 6, "targets.yml": 4, "build-cache.yml": 4,
    "benchmark.yml": 9, "release-build.yml": 24, "doc.yml": 6, "fmt.yml": 1,
    "mise.yml": 5, "pixi.yml": 5,
}
BOOTSTRAP_RUNNER_JOBS = {
    ("ci-prepare.yml", "prepare"), ("ci-policy.yml", "policy"),
    ("pr-closed.yml", "cancel-pr-ci"),
}


def load(name):
    # BaseLoader preserves the GitHub Actions 'on' key instead of YAML 1.1's True.
    return yaml.load((WORKFLOWS / name).read_text(), Loader=yaml.BaseLoader)


def workflow_paths():
    return sorted(path for path in WORKFLOWS.iterdir()
                  if path.suffix in {".yml", ".yaml"})


def pr_workflows():
    return {path.name: workflow for path in workflow_paths()
            if "pull_request" in (workflow := load(path.name)).get("on", {})}


def needs(job):
    value = job.get("needs", [])
    return [value] if isinstance(value, str) else value


def matrix_size(job):
    matrix = job.get("strategy", {}).get("matrix", {})
    axes = [v for k, v in matrix.items() if k not in {"include", "exclude"}]
    if not axes:
        return len(matrix.get("include", [None]))
    count = 1
    for values in axes:
        count *= len(values)
    return count - len(matrix.get("exclude", []))


class WorkflowContractTests(unittest.TestCase):
    def test_macos_benchmark_installs_only_legacy_base_dependency(self):
        steps = load("benchmark.yml")["jobs"]["benchmark"]["steps"]
        step = next(s for s in steps if s.get("name") == "Install legacy macOS linker for the benchmark base")
        self.assertIn("github.event_name == 'pull_request'", step["if"])
        for legacy in (False, True):
            with self.subTest(legacy=legacy), tempfile.TemporaryDirectory() as directory:
                root = Path(directory) / "workspace with spaces"
                base = root / ".benchmark/base-source"
                (base / "internal/build").mkdir(parents=True)
                marker = root / "installed"
                (base / "install.sh").write_text(
                    'test "$LLGO_INSTALLER_LIBRARY_ONLY" = 1 || exit 9\n'
                    'install_macos_lld() { touch "$TEST_MARKER"; PATH=/invalid; }\n')
                if legacy:
                    (base / "internal/build/darwin_linker.go").touch()
                result = subprocess.run(["bash", "-c", step["run"] + '\ncommand -v bash\n'],
                                        capture_output=True, text=True,
                                        env={**os.environ, "GITHUB_WORKSPACE": str(root), "TEST_MARKER": str(marker)})
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(marker.exists(), legacy)

    def test_pr_close_covers_every_pr_workflow(self):
        workflow = load("pr-closed.yml")
        self.assertEqual(workflow["on"], {"pull_request_target": {"types": ["closed"]}})
        self.assertEqual(workflow["permissions"], {})
        self.assertNotIn("concurrency", workflow)
        job = workflow["jobs"]["cancel-pr-ci"]
        self.assertNotIn("if", job)
        self.assertNotIn("concurrency", job)
        self.assertEqual(job["permissions"], {
            "actions": "write", "contents": "read", "pull-requests": "read"})
        checkout, cancel = job["steps"]
        self.assertEqual(checkout["uses"], "actions/checkout@v7")
        self.assertEqual(checkout["with"], {
            "ref": "${{ github.sha }}", "persist-credentials": "false",
            "sparse-checkout": ".github/scripts", "sparse-checkout-cone-mode": "false"})
        self.assertEqual(cancel["env"], {"GH_TOKEN": "${{ github.token }}"})
        self.assertEqual(cancel["run"], "python3 .github/scripts/pr_ci.py")
        self.assertEqual(job["timeout-minutes"], "5")
        self.assertFalse((WORKFLOWS / "pr-ci-cancel.yml").exists())

    def test_pr_jobs_and_local_reusable_jobs_remain_cancelable(self):
        # Discover every job rather than keeping a job-name allowlist: new jobs
        # and matrices are covered even when CODE_WORKFLOWS does not list them.
        pending = list(pr_workflows())
        checked = set()
        while pending:
            filename = pending.pop()
            if filename in checked:
                continue
            checked.add(filename)
            for name, job in load(filename)["jobs"].items():
                with self.subTest(workflow=filename, job=name):
                    condition = " ".join(job.get("if", "").split())
                    condition = condition.removeprefix("${{").removesuffix("}}").strip()
                    # Running job conditions are re-evaluated on cancellation.
                    # Unconditional always() would keep even a gate running.
                    if "always()" in condition:
                        self.assertEqual(condition, "always() && !cancelled()")
                    reusable = job.get("uses", "")
                    if reusable.startswith("./.github/workflows/"):
                        pending.append(Path(reusable).name)

    def test_extended_cross_target_abi_matrix_runs_on_linux(self):
        step = next(step for step in load("go.yml")["jobs"]["test"]["steps"]
                    if step.get("id") == "test_coverage")
        self.assertEqual(step["env"]["LLGO_TEST_EXTENDED_ABI"],
                         "${{ runner.os == 'Linux' && '1' || '0' }}")

    def test_native_workloads_have_independent_bounded_jobs(self):
        jobs = load("llgo.yml")["jobs"]
        self.assertIn("native-simd", list(jobs))
        self.assertIn("native-integration", list(jobs))
        core = jobs["llgo"]
        integration = jobs["native-integration"]
        simd = jobs["native-simd"]
        entries = core["strategy"]["matrix"]["include"]
        self.assertEqual(matrix_size(core), 8)
        self.assertEqual(integration["strategy"], core["strategy"])
        self.assertEqual(simd["strategy"]["matrix"]["include"],
                         [entry for entry in entries if entry.get("windows_arch") != "386"])
        self.assertEqual(matrix_size(simd), 6)
        for job in (core, integration, simd):
            self.assertEqual(needs(job), ["prepare"])
            self.assertEqual(job["if"], "needs.prepare.outputs.run_code_ci == 'true'")
            self.assertEqual(job["timeout-minutes"], "60")
            self.assertEqual(job["strategy"]["fail-fast"], "false")
            self.assertEqual(job["runs-on"], core["runs-on"])
            self.assertNotIn("continue-on-error", job)
            self.assertTrue(all("continue-on-error" not in step for step in job["steps"]))

        common = ["Install dependencies", "Install embedded dependencies",
                  "Setup demo dependencies", "Set up Go for build", "Restore LLGo cache",
                  "Build LLGo and test tools", "Activate native Windows target"]
        core_steps = {step.get("name"): step for step in core["steps"]}
        integration_steps = {step.get("name"): step for step in integration["steps"]}
        self.assertEqual([step["name"] for step in integration["steps"]
                          if step.get("name") in common], common)
        for name in common:
            self.assertEqual(integration_steps[name], core_steps[name])
        for name in ("Execute Windows 386 atomic contracts", "Check Windows native runtime and FFI",
                     "Run Go 1.27 LLGo core tests", "Upload LLGo test coverage to Codecov"):
            self.assertIn(name, core_steps)
            self.assertNotIn(name, integration_steps)
        self.assertNotIn("Test SIMD behavior with Go and LLGo", core_steps)
        self.assertNotIn("Test demos", core_steps)

    def test_native_simd_keeps_the_complete_original_harness(self):
        jobs = load("llgo.yml")["jobs"]
        self.assertIn("native-simd", list(jobs))
        job = jobs["native-simd"]
        steps = {step.get("name"): step for step in job["steps"]}
        deps = steps["Install dependencies"]
        self.assertEqual(deps["uses"], "./.github/actions/setup-deps")
        self.assertEqual(deps["with"]["windows-msvc-dependencies"], "runtime")
        self.assertNotIn("Install embedded dependencies", steps)
        self.assertNotIn("Setup demo dependencies", steps)
        self.assertEqual(steps["Activate native Windows target"],
                         next(step for step in jobs["llgo"]["steps"]
                              if step.get("name") == "Activate native Windows target"))
        test = steps["Test SIMD behavior with Go and LLGo"]
        self.assertNotIn("if", test)
        self.assertEqual(test["id"], "native_simd")
        self.assertEqual(test["shell"], "bash")
        self.assertEqual(test["env"], {
            "GOEXPERIMENT": "simd",
            "LLGO_SIMD_ARTIFACT_DIR": "${{ runner.temp }}/llgo-simd-matrix"})
        self.assertEqual(test["run"].splitlines(), [
            "(cd runtime && go test -count=1 -timeout=2m ./internal/runtime/math)",
            "go test -count=1 -timeout=2m ./test/simd/...",
            "bash dev/test_native_simd.sh"])
        artifact = steps["Upload failed native SIMD executables"]
        self.assertEqual(artifact["if"], "failure() && steps.native_simd.outcome == 'failure'")
        self.assertEqual(artifact["with"]["path"], test["env"]["LLGO_SIMD_ARTIFACT_DIR"])
        build = steps["Build SIMD compiler"]
        self.assertIn('go build -o "$RUNNER_TEMP/llgo-bin/llgo${tool_suffix}" ./cmd/llgo', build["run"])
        self.assertIn('echo "LLGO_ROOT=$GITHUB_WORKSPACE" >> "$GITHUB_ENV"', build["run"])

    def test_native_integration_preserves_ordered_post_core_gates(self):
        jobs = load("llgo.yml")["jobs"]
        self.assertIn("native-integration", list(jobs))
        steps = jobs["native-integration"]["steps"]
        names = [step.get("name") for step in steps]
        start = names.index("Test Hello World with Go 1.20")
        self.assertEqual(names[start:], [
            "Test Hello World with Go 1.20",
            "Test Hello World with go.mod 1.21 through 1.26",
            "Test Hello World with go.mod 1.27",
            "Test runtime module compatibility",
            "Test zero-sized reflection argument ABI",
            "Run caller regressions in C shared libraries and archives",
            "Validate the native Windows toolchain profile",
            "Test Windows CPU profiler context",
            "Test demo without RPATH (expect failure)",
            "Test static cargs without a runtime search path",
            "Test demos",
            "Test demos (embedded target)",
            "Test demos (embedded target, build only)",
            "Test C header generation",
            "Test export with different symbol names on embedded targets",
            "Test ESP serial smoke",
            "Test ESP32-C3 startup regression",
            "_xtool build tests",
            "Show test result",
            "Install LLDB for integration tests",
            "GDB values, registry and complete worker unwind (Linux)",
            "Install MinGW LLDB for integration tests",
            "LLDB integration tests",
            "LLDB integration tests (MSVC)",
            "LLDB integration tests (MinGW)",
            "LLDB integration tests (MinGW cross-architecture)",
            "Native source-debug acceptance tests",
            "Native debug metadata tests",
            "Native debug metadata tests (Windows)",
            "Set up the MSYS2 host shell for MSVC",
            "Test from the MSYS2 host shell",
            "Install Cygwin for host-shell qualification",
            "Test from the Cygwin host shell",
            "Install Windows GDB",
            "GDB native values and registry (Windows)",
            "GDB complete worker unwind (Windows amd64 and 386)"])
        self.assertTrue(all(step.get("run") or step.get("uses") for step in steps[start:]))

    def test_source_abi_keeps_five_executed_llvm22_regressions(self):
        job = load("llgo.yml")["jobs"]["source-abi"]
        cases = ["sqlite", "purego", "turso", "libccallback", "crc32"]
        self.assertEqual(job["strategy"], {
            "fail-fast": "false", "matrix": {"case": cases}})
        self.assertEqual(matrix_size(job), 5)
        self.assertEqual(needs(job), ["prepare"])
        self.assertEqual(job["if"], "needs.prepare.outputs.run_code_ci == 'true'")
        self.assertEqual(job["runs-on"],
                         "${{ fromJSON(needs.prepare.outputs.linux_large_runner) }}")
        self.assertEqual(job["timeout-minutes"], "45")
        self.assertNotIn("continue-on-error", job)
        for step in job["steps"]:
            self.assertNotIn("continue-on-error", step)
            self.assertNotIn("|| true", step.get("run", ""))
        deps = next(step for step in job["steps"]
                    if step.get("uses") == "./.github/actions/setup-deps")
        self.assertEqual(deps["with"]["llvm-version"], "22")
        self.assertNotIn("if", deps)
        steps = {step.get("name"): step for step in job["steps"]}
        build = steps["Build source ABI compiler"]
        self.assertNotIn("if", build)
        self.assertEqual(build["shell"], "bash")
        self.assertIn('go build -o "$RUNNER_TEMP/llgo-bin/llgo" ./cmd/llgo', build["run"])
        self.assertIn('echo "$RUNNER_TEMP/llgo-bin" >> "$GITHUB_PATH"', build["run"])
        self.assertIn('echo "LLGO_ROOT=$GITHUB_WORKSPACE" >> "$GITHUB_ENV"', build["run"])
        test = steps["Test exact-version source ABI regression"]
        self.assertNotIn("if", test)
        self.assertEqual(test["id"], "source_abi")
        self.assertEqual(test["shell"], "bash")
        self.assertEqual(test["run"], "bash dev/test_source_abi.sh '${{ matrix.case }}'")
        callback = steps["Test callback ABI and foreign-thread lifetime"]
        self.assertEqual(callback["if"], "matrix.case == 'purego'")
        self.assertEqual(callback["shell"], "bash")
        self.assertEqual(callback["working-directory"], "runtime/_test/puregobridge")
        self.assertEqual(callback["run"], "timeout -k 10s 10m llgo run .")
        receipts = steps["Upload source ABI regression receipts"]
        # !cancelled() also runs after a failed build/run or callback step.
        self.assertEqual(receipts["if"],
                         "${{ !cancelled() && steps.source_abi.outcome != 'skipped' }}")
        self.assertEqual(receipts["uses"], "actions/upload-artifact@v7")
        self.assertEqual(receipts["with"], {
            "name": "source-abi-${{ matrix.case }}", "path": "_out/source-abi.*/",
            "if-no-files-found": "error", "retention-days": "7"})
        self.assertEqual([step["name"] for step in job["steps"][-4:]],
                         [build["name"], test["name"], callback["name"], receipts["name"]])

    def test_source_abi_script_retains_build_run_and_failure_receipts(self):
        script = (WORKFLOWS.parent.parent / "dev/test_source_abi.sh").read_text()
        self.assertIn("set -euo pipefail", script)
        self.assertIn("cases=(sqlite purego turso libccallback crc32)", script)
        self.assertIn('[[ $("$llgo" env LLGO_LLVM_VERSION) == 22.* ]]', script)
        self.assertIn("grep -Eq 'clang version 22\\.'", script)
        self.assertIn("compilers=(go llgo)", script)
        self.assertIn("compilers=(go go-adapted llgo)", script)
        self.assertIn('flags+=(-modfile="$stage/build.mod")', script)
        self.assertIn('tool="$llgo"', script)
        self.assertIn('timeout -k 10s 20m "$tool" build "${flags[@]}" -o "$binary" .', script)
        self.assertIn('if ! timeout -k 10s 2m "$binary" > "$report/$name-$compiler.run.log" 2>&1; then\n'
                      '        status=failed', script)
        self.assertIn('else\n      status=failed\n    fi', script)
        self.assertIn('"$name" "$compiler" "$phase" "$status" >> "$report/results.jsonl"', script)
        self.assertIn('if [[ $status == failed ]]; then\n      failed=1', script)
        self.assertTrue(script.rstrip().endswith('exit "$failed"'))
        self.assertNotIn("|| true", script)

    def test_zero_sized_reflection_gate_keeps_native_abi_boundary(self):
        steps = load("llgo.yml")["jobs"]["native-integration"]["steps"]
        test = next(step for step in steps
                    if step.get("name") == "Test zero-sized reflection argument ABI")
        # Windows AMD64 and 386 do not omit empty parameters; their existing
        # ABI rules are not qualified by the newly added omission fix.
        self.assertEqual(test["if"],
                         "runner.os != 'Windows' || matrix.windows_arch == 'arm64'")
        self.assertEqual(test["shell"], "bash")
        self.assertEqual(test["working-directory"], "runtime/_test/reflectzeroabi")
        self.assertEqual(test["run"].splitlines(), ["go run .", "llgo run ."])
        self.assertNotIn("continue-on-error", test)

    def test_gdb_windows_acceptance_covers_both_abis_and_all_native_architectures(self):
        job = load("llgo.yml")["jobs"]["native-integration"]
        entries = job["strategy"]["matrix"]["include"]
        windows = {(entry["windows_abi"], entry["windows_arch"])
                   for entry in entries if entry["os"].startswith("windows")}
        self.assertEqual(windows, {(abi, arch) for abi in ("msvc", "mingw")
                                   for arch in ("amd64", "386", "arm64")})
        steps = job["steps"]
        install = next(step for step in steps
                       if step.get("name") == "Install Windows GDB")
        test = next(step for step in steps
                    if step.get("name") == "GDB native values and registry (Windows)")
        self.assertEqual(install["if"], "runner.os == 'Windows'")
        self.assertEqual(test["if"], "runner.os == 'Windows'")
        self.assertNotIn("continue-on-error", test)
        self.assertEqual(test["env"]["LLGO_GDB_INTEGRATION"], "1")
        # Its isolated MSYS2 setup changes the shell helper: it belongs after
        # all existing LLDB/host-shell gates, not before target qualification.
        shell_tests = [index for index, step in enumerate(steps)
                       if step.get("name", "").startswith(("LLDB integration", "Test from the"))]
        self.assertGreater(steps.index(install), max(shell_tests))
        complete = next(step for step in steps
                        if step.get("name") == "GDB complete worker unwind (Windows amd64 and 386)")
        self.assertEqual(complete["if"], "runner.os == 'Windows' && matrix.windows_arch != 'arm64'")
        self.assertIn("^TestGDBCompleteWorkerUnwind$", complete["run"])
        self.assertNotIn("continue-on-error", complete)
        linux = next(step for step in steps
                     if step.get("name") == "GDB values, registry and complete worker unwind (Linux)")
        self.assertIn("^TestGDB(Integration|CompleteWorkerUnwind)$", linux["run"])

    def test_intel_macos_runs_both_native_debuggers(self):
        job = load("llgo.yml")["jobs"]["native-debuggers-intel"]
        self.assertEqual(job["runs-on"], "macos-15-intel")
        commands = "\n".join(step.get("run", "") for step in job["steps"])
        self.assertIn("test/debug/runtime/runtest.sh -v", commands)
        self.assertIn("^TestGDBIntegration$", commands)
        self.assertNotIn("^TestGDBCompleteWorkerUnwind$", commands)
        self.assertIn("test/debug/runtime/gdb_darwin.py", commands)
        self.assertNotIn("sudo", commands)
        helper = runpy.run_path(str(WORKFLOWS.parents[1] / "test/debug/runtime/gdb_darwin.py"))
        with mock.patch("os.execvp", side_effect=SystemExit) as execute, self.assertRaises(SystemExit):
            helper["main"]("/native/gdb", ["--version"])
        execute.assert_called_once_with("sudo", ["sudo", "-n", "/native/gdb", "--version"])
        self.assertNotIn("continue-on-error", job)
        self.assertTrue(all("continue-on-error" not in step for step in job["steps"]))

    def test_windows_386_atomic_contracts_build_on_host_and_execute_on_target(self):
        job = load("llgo.yml")["jobs"]["llgo"]
        lanes = [entry for entry in job["strategy"]["matrix"]["include"]
                 if entry.get("windows_arch") == "386"]
        self.assertEqual({entry["windows_abi"] for entry in lanes}, {"msvc", "mingw"})
        self.assertEqual(len(lanes), 2)
        self.assertTrue(all(entry["os"] == "windows-2022" and
                            entry["host_go_arch"] == "x64" for entry in lanes))
        steps = job["steps"]
        build = next((step for step in steps if step.get("name") ==
                      "Build host runner for Windows 386 atomic contracts"), None)
        execute = next((step for step in steps if step.get("name") ==
                        "Execute Windows 386 atomic contracts"), None)
        self.assertIsNotNone(build, "the real 386 atomic roots have no CI host runner")
        self.assertIsNotNone(execute, "the real 386 atomic roots have no native execution step")
        activate = next(step for step in steps
                        if step.get("name") == "Activate native Windows target")
        self.assertLess(steps.index(build), steps.index(activate))
        self.assertLess(steps.index(activate), steps.index(execute))
        condition = "runner.os == 'Windows' && matrix.windows_arch == '386'"
        for step in (build, execute):
            self.assertEqual(step["if"], condition)
            self.assertEqual(step["shell"], "pwsh")
            self.assertNotIn("continue-on-error", step)
            self.assertNotIn("GITHUB_ENV", step["run"])
            self.assertIn('$ErrorActionPreference = "Stop"', step["run"])
            self.assertIn("$LASTEXITCODE -ne 0", step["run"])

        compile_script = build["run"]
        self.assertIn('$env:GOARCH = "amd64"', compile_script)
        self.assertIn("$env:CC = $env:LLGO_WINDOWS_HOST_CC", compile_script)
        self.assertIn("$env:CXX = $env:LLGO_WINDOWS_HOST_CXX", compile_script)
        self.assertIn('$hostBin = Join-Path $env:LLGO_MINGW_HOST_ROOT "bin"', compile_script)
        self.assertIn('$env:CC = Join-Path $hostBin "clang.exe"', compile_script)
        self.assertIn('$env:CXX = Join-Path $hostBin "clang++.exe"', compile_script)
        self.assertIn("go test -c -tags=llgo_windows_386_cross", compile_script)
        self.assertIn("./internal/build", compile_script)
        self.assertIn("release-lib.ps1", compile_script)
        self.assertIn("Copy-ReleaseDLLs", compile_script)
        self.assertIn('-GoArch "amd64"', compile_script)
        self.assertIn("-CheckOnly", compile_script)
        self.assertNotIn("LLGO_WINDOWS_CROSS_CC", compile_script)

        run_script = execute["run"]
        self.assertIn("$env:LLGO_WINDOWS_CROSS_CC = $env:CC", run_script)
        self.assertIn("$env:LLGO_WINDOWS_CROSS_CXX = $env:CXX", run_script)
        self.assertNotIn("$env:CC =", run_script)
        self.assertNotIn("$env:CXX =", run_script)
        self.assertIn("^(TestRuntimeAtomic386(CompletePackage|ActualWrappersMatchOriginalGo)|TestMapsRuntime386CompletePackage)$",
                      run_script)
        self.assertIn('"-test.list=$selector"', run_script)
        for argument in ("-test.run=$selector", "-test.v", "-test.count=1", "-test.timeout=30m"):
            self.assertIn(f'"{argument}"', run_script)
        self.assertIn('$output = @(& $runner @arguments 2>&1)', run_script)
        self.assertIn("TestRuntimeAtomic386CompletePackage", run_script)
        self.assertIn("TestRuntimeAtomic386ActualWrappersMatchOriginalGo", run_script)
        self.assertIn("'TestMapsRuntime386CompletePackage'", run_script)
        self.assertIn("$listed.Count -ne 3", run_script)
        self.assertIn("Compare-Object", run_script)
        self.assertIn("--- PASS:", run_script)
        self.assertIn("--- (FAIL|SKIP):", run_script)

    def test_traceback_coverage_uses_bash_on_every_host(self):
        steps = load("go.yml")["jobs"]["test"]["steps"]
        step = next(step for step in steps
                    if step.get("name") == "Test traceback formatter with coverage")
        # PowerShell splits the unquoted -coverprofile=../... argument and Go
        # treats its value as a package. Preserve the shell when moving steps.
        self.assertEqual(step["shell"], "bash")
        self.assertEqual(step["working-directory"], "runtime")
        self.assertIn("-coverprofile=../coverage-traceback.txt", step["run"])

    def test_go_suite_scheduling_preserves_overlays_and_propagates_failures(self):
        step = next(step for step in load("go.yml")["jobs"]["test"]["steps"]
                    if step.get("id") == "test_coverage")
        script = step["run"].replace("${{ matrix.windows_abi }}", "msvc")
        scenarios = [("Linux", "", ""), ("macOS", "", ""), ("Windows", "", ""),
                     ("Windows", "C:/runner temp/runtime-overlay.json", "")]
        scenarios += [(runner, "", suite) for runner in ("Linux", "macOS", "Windows")
                      for suite in ("main", "build", "compiler", "language")]
        for runner, overlay, failure in scenarios:
            with self.subTest(runner=runner, overlay=overlay, failure=failure), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                log = root / "commands.jsonl"
                summary = root / "summary.md"
                fake_go = root / "go"
                fake_go.write_text(f"#!{sys.executable}\n" + r"""
import json, os, sys, time
from pathlib import Path
if sys.argv[1] == 'list':
    print('github.com/xgo-dev/llgo/cmd/internal/version')
    print('github.com/xgo-dev/llgo/internal/build')
else:
    args = sys.argv[1:]
    suite = next((name for package, name in (
        ('./internal/build', 'build'), ('./cl', 'compiler'), ('./test/go', 'language'))
        if package in args), 'main')
    root = Path(os.environ['TEST_GO_LOG']).parent
    with (root / 'commands.jsonl').open('a') as stream:
        stream.write(json.dumps(args) + '\n')
    (root / (suite + '.started')).touch()
    if os.environ['RUNNER_OS'] == 'Linux' and suite in {'main', 'build'}:
        peer = root / (('main' if suite == 'build' else 'build') + '.started')
        deadline = time.monotonic() + 5
        while not peer.exists():
            if time.monotonic() > deadline:
                sys.exit('independent suites did not overlap')
            time.sleep(0.01)
    elif os.environ['RUNNER_OS'] != 'Linux':
        order = ['main', 'build', 'compiler', 'language']
        index = order.index(suite)
        if index and not (root / (order[index - 1] + '.finished')).exists():
            sys.exit('host suites did not retain serial order')
    (root / (suite + '.finished')).touch()
    sys.exit(7 if suite == os.environ['TEST_GO_FAILURE'] else 0)
""")
                fake_go.chmod(0o755)
                (root / "dev").mkdir()
                (root / "dev/go_test_windows.sh").write_text('exec "$@"\n')
                helper = root / ".github/scripts/run_go_tests.py"
                helper.parent.mkdir(parents=True)
                shutil.copyfile(WORKFLOWS.parent / "scripts/run_go_tests.py", helper)
                # This contract checks suite ordering and Windows options.
                # The real shard executor has separate end-to-end Go tests.
                (helper.parent / "run_build_tests.py").write_text(r"""
import argparse, os, subprocess, sys
parser = argparse.ArgumentParser()
parser.add_argument('--overlay')
parser.add_argument('--ldflags')
parser.add_argument('--require-arm64-atomic', action='store_true')
args = parser.parse_args()
if args.require_arm64_atomic != (os.environ['RUNNER_OS'] == 'macOS'):
    sys.exit('native ARM64 atomic gate was omitted or applied to another host')
command = ['go', 'test', '-json', '-coverpkg=./...', '-coverprofile=coverage-build.txt']
if args.overlay:
    command.append('-overlay=' + args.overlay)
if args.ldflags:
    command.append('-ldflags=' + args.ldflags)
sys.exit(subprocess.call(command + ['./internal/build']))
""")
                result = subprocess.run(["bash", "-c", script], cwd=root,
                                        capture_output=True, text=True, timeout=20,
                                        env={**os.environ,
                                             "PATH": directory + os.pathsep + os.environ["PATH"],
                                             "RUNNER_OS": runner, "LLGO_GO_TEST_OVERLAY": overlay,
                                             "RUNNER_ARCH": "ARM64" if runner == "macOS" else "X64",
                                             "TEST_GO_LOG": str(log), "TEST_GO_FAILURE": failure,
                                             "GITHUB_STEP_SUMMARY": str(summary)})
                if failure == "main":
                    # GNU and BSD xargs map a child failure to different codes.
                    self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                else:
                    self.assertEqual(result.returncode, 7 if failure else 0,
                                     result.stdout + result.stderr)
                calls = [json.loads(line) for line in log.read_text().splitlines()]
                expected_suites = {"main", "build"}
                if failure != "main":
                    expected_suites.add("compiler")
                if failure not in {"main", "compiler"}:
                    expected_suites.add("language")
                if runner != "Linux":
                    order = ["main", "build", "compiler", "language"]
                    expected_suites = set(order[:order.index(failure) + 1] if failure else order)
                self.assertEqual(len(calls), len(expected_suites))
                report = summary.read_text()
                for suite in expected_suites:
                    self.assertTrue((root / (suite + ".finished")).exists())
                    self.assertIn(f"### Go tests: {suite} (exit {7 if suite == failure else 0})", report)
                for package in ("./cl", "./internal/build", "./test/go"):
                    for args in (args for args in calls if package in args):
                        self.assertEqual([arg for arg in args if arg.startswith("-overlay=")],
                                         [f"-overlay={overlay}"] if overlay else [])
                main = next(args for args in calls if "github.com/xgo-dev/llgo/cmd/internal/version" in args)
                self.assertEqual("-p=2" in main, runner == "Linux")
                self.assertFalse(any(arg.startswith("-overlay=") for arg in main))
                # Every build case still runs exactly once with compiler coverage.
                if "build" in expected_suites:
                    build = next(args for args in calls if "./internal/build" in args)
                    self.assertNotIn("-run", build)
                    self.assertNotIn("-skip", build)
                    self.assertIn("-coverprofile=coverage-build.txt", build)
                    self.assertIn("-coverpkg=./...", build)
                    self.assertEqual(sum("./internal/build" in args for args in calls), 1)
                    self.assertEqual(any(arg.startswith("-ldflags=-linkmode=external") for arg in build),
                                     runner == "Windows")
                self.assertFalse(any("github.com/xgo-dev/llgo/internal/build" in args
                                     for args in calls))
                self.assertTrue(all("-json" in args for args in calls))

    def test_macos_build_suite_requires_arm64_native_atomic_acceptance(self):
        job = load("go.yml")["jobs"]["test"]
        entries = job["strategy"]["matrix"]["include"]
        self.assertIn("macos-latest", [entry["os"] for entry in entries])
        step = next(step for step in job["steps"] if step.get("id") == "test_coverage")
        self.assertNotIn("continue-on-error", job)
        self.assertNotIn("continue-on-error", step)
        # Exercise the existing function without invoking a Go or LLVM process.
        body = step["run"].split("run_build_suite() {", 1)[1].split("\n}", 1)[0]
        function = "run_build_suite() {" + body + "\n}\n"
        script = function + 'python() { printf "%s\\n" "$@"; }\nrun_build_suite\n'
        for arch in ("ARM64", "X64", ""):
            with self.subTest(arch=arch):
                result = subprocess.run(["bash", "-c", script], capture_output=True, text=True,
                                        env={**os.environ, "RUNNER_OS": "macOS", "RUNNER_ARCH": arch,
                                             "LLGO_GO_TEST_OVERLAY": ""}, timeout=10)
                if arch == "ARM64":
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout.splitlines().count("--require-arm64-atomic"), 1)
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(result.stdout, "")
                    self.assertIn("native ARM64 atomic owners", result.stderr)

    def test_cross_host_windows_checks_keep_every_host_target_pair(self):
        job = load("llgo.yml")["jobs"]["windows-gnu-cross"]
        entries = job["strategy"]["matrix"]["include"]
        checks = [step for step in job["steps"]
                  if step.get("uses") == "./.github/actions/test-windows-cross"]
        actual = set()
        for entry in entries:
            for step in checks:
                arch = step["with"]["windows-arch"]
                if entry["windows_arch"] in {"all", arch}:
                    actual.add((entry["os"], arch))
                self.assertIn("!cancelled()", step["if"])
                self.assertIn("steps.host_compiler.outcome == 'success'", step["if"])
                self.assertIn(f"matrix.windows_arch == '{arch}'", step["if"])
                self.assertNotIn("continue-on-error", step)
        self.assertEqual(actual, {(host, arch) for host in ("ubuntu-24.04", "macos-latest")
                                  for arch in ("amd64", "arm64", "386")})
        self.assertEqual(sum(entry["os"] == "macos-latest" for entry in entries), 1)

    def test_embedded_debuggers_share_setup_without_losing_backend_failures(self):
        job = load("targets.yml")["jobs"]["embedded-debug"]
        self.assertCountEqual(job["strategy"]["matrix"]["os"],
                              ["ubuntu-24.04", "macos-15", "windows-2022"])
        self.assertNotIn("backend", job["strategy"]["matrix"])
        for backend in ("gdb", "lldb"):
            step = next(step for step in job["steps"]
                        if f"runtest.py --backend {backend}" in step.get("run", ""))
            self.assertIn("!cancelled()", step["if"])
            self.assertIn("steps.compiler.outcome == 'success'", step["if"])
            self.assertNotIn("continue-on-error", step)

    def test_all_board_fixtures_share_one_compiler_and_cache(self):
        job = load("targets.yml")["jobs"]["llgo"]
        self.assertEqual(matrix_size(job), 1)
        steps = job["steps"]
        compiler = next(step for step in steps if step.get("id") == "compiler")
        self.assertIn("go build -tags=dev", compiler["run"])
        self.assertIn("LLGO_TEST_COMPILER=", compiler["run"])
        cache = next(step for step in steps
                     if step.get("uses") == "./.github/actions/setup-llgo-cache")
        self.assertEqual(cache["with"]["crosscompile-cache-mode"], "readwrite")
        self.assertEqual(cache["with"]["build-cache-scope"], "embedded-boards")
        for fixture in ("empty", "defer"):
            step = next(step for step in steps
                        if step.get("run") == f"bash build.sh {fixture}")
            self.assertEqual(step["working-directory"], "_demo/embed/targetsbuild")
            self.assertIn("!cancelled()", step["if"])
            self.assertIn("steps.compiler.outcome == 'success'", step["if"])
            self.assertNotIn("continue-on-error", step)

    def test_wasm_caches_use_pinned_go_and_include_workers(self):
        steps = load("llgo.yml")["jobs"]["wasm-runtime"]["steps"]
        go = next(i for i, step in enumerate(steps)
                  if step.get("uses") == "./.github/actions/setup-go")
        cache = next(i for i, step in enumerate(steps)
                     if step.get("uses") == "./.github/actions/setup-llgo-cache")
        self.assertLess(go, cache)
        self.assertNotIn("if", steps[cache])
        self.assertEqual(steps[cache]["with"]["build-cache-mode"], "readwrite")
        self.assertEqual(steps[cache]["with"]["build-cache-scope"], "wasm-${{ matrix.suite }}")
        self.assertEqual(steps[cache]["with"]["crosscompile-cache-mode"], "readwrite")
        self.assertIn("matrix.suite != 'workers'", steps[cache]["with"]["wasmer-cache-mode"])

    def test_benchmark_runtime_manifest_keeps_versions_and_full_demo_dependencies(self):
        root = WORKFLOWS.parents[1]
        full = json.loads((root / ".github/windows/vcpkg/vcpkg.json").read_text())
        runtime = json.loads((root / ".github/windows/vcpkg-runtime/vcpkg.json").read_text())
        self.assertEqual(full["builtin-baseline"], runtime["builtin-baseline"])
        self.assertEqual(full["overrides"], runtime["overrides"])
        runtime_names = {dep if isinstance(dep, str) else dep["name"]
                         for dep in runtime["dependencies"]}
        full_names = {dep if isinstance(dep, str) else dep["name"]
                      for dep in full["dependencies"]}
        self.assertEqual(runtime_names, {"bdwgc", "libffi", "libuv", "pkgconf", "zlib"})
        self.assertEqual(full_names - runtime_names, {"openssl", "cjson", "sqlite3"})
        self.assertIn({"name": "pkgconf", "host": True}, runtime["dependencies"])
        action = yaml.load((root / ".github/actions/setup-deps/action.yml").read_text(),
                           Loader=yaml.BaseLoader)
        self.assertEqual(action["inputs"]["windows-msvc-dependencies"]["default"], "full")
        cache = next(step for step in action["runs"]["steps"]
                     if step.get("id") == "windows_vcpkg_cache")
        self.assertIn("steps.windows_profile.outputs.manifest-hash", cache["with"]["key"])
        self.assertIn("steps.windows_profile.outputs.manifest-hash", cache["with"]["path"])
        steps = load("benchmark.yml")["jobs"]["benchmark"]["steps"]
        deps = next(step for step in steps
                    if step.get("uses") == "./.github/actions/setup-deps")
        self.assertEqual(deps["with"]["windows-msvc-dependencies"], "runtime")

    def test_cache_scopes_preserve_legacy_keys_and_isolate_package_workloads(self):
        root = WORKFLOWS.parents[1]
        action = yaml.load((root / ".github/actions/setup-llgo-cache/action.yml").read_text(),
                           Loader=yaml.BaseLoader)
        resolve = action["runs"]["steps"][0]["run"]
        with tempfile.TemporaryDirectory() as directory:
            directory = Path(directory)
            for tool, output in (("go", "go1.27.0"), ("clang", "clang version 22.1.8")):
                path = directory / tool
                path.write_text(f"#!/bin/sh\necho '{output}'\n")
                path.chmod(0o755)
            profiles = []
            for scope in ("", "embedded-boards", "wasm-runtime", "wasm-workers", "bad/scope"):
                output = directory / "outputs"
                output.write_text("")
                result = subprocess.run(["bash", "-c", resolve], cwd=root,
                    env={**os.environ, "PATH": f"{directory}{os.pathsep}{os.environ['PATH']}",
                         "BUILD_CACHE_MODE": "readwrite", "CROSSCOMPILE_CACHE_MODE": "readwrite",
                         "WASMER_CACHE_MODE": "off", "BUILD_CACHE_SCOPE": scope,
                         "LLGO_WINDOWS_ABI": "native", "LLGO_WINDOWS_ARCH": "native",
                         "GITHUB_OUTPUT": str(output)}, capture_output=True, text=True)
                if scope == "bad/scope":
                    self.assertNotEqual(result.returncode, 0)
                    continue
                self.assertEqual(result.returncode, 0, result.stderr)
                values = dict(line.split("=", 1) for line in output.read_text().splitlines())
                self.assertEqual(values["profile"], "native-native")
                self.assertEqual(values["build-profile"], "native-native" + (f"-{scope}" if scope else ""))
                profiles.append(values["build-profile"])
            self.assertEqual(len(profiles), len(set(profiles)))

    def test_code_and_document_jobs_consume_the_shared_decision(self):
        for filename in [*CODE_WORKFLOWS, "doc-link-checker.yml", "model-demo.yml"]:
            workflow = load(filename)
            flag = "run_doc_checks" if filename == "doc-link-checker.yml" else "run_code_ci"
            self.assertEqual(workflow["jobs"]["prepare"]["uses"], PREPARE)
            for name, job in workflow["jobs"].items():
                if name in {"prepare", "ci-gate"}:
                    continue
                with self.subTest(workflow=filename, job=name):
                    self.assertIn("prepare", needs(job))
                    self.assertIn(f"needs.prepare.outputs.{flag} == 'true'", job["if"])

    def test_pr_and_main_concurrency_groups_are_isolated(self):
        for path in workflow_paths():
            workflow = load(path.name)
            events = workflow.get("on", {})
            if "push" not in events and "pull_request" not in events:
                continue
            with self.subTest(workflow=path.name):
                if "push" in events:
                    push = events["push"] or {}
                    self.assertIn("main", push.get("branches", []))
                    self.assertNotIn("paths", push)
                    self.assertNotIn("paths-ignore", push)
                concurrency = workflow["concurrency"]
                # Use the filename as a stable key, independent of display names.
                # Retain main supersession and unique tag/manual run groups.
                self.assertEqual(concurrency["cancel-in-progress"],
                                 "${{ github.event_name == 'pull_request' || (github.event_name == 'push' && github.ref == 'refs/heads/main') }}")
                self.assertEqual(concurrency["group"],
                                 path.name + "-${{ github.event_name }}-${{ github.event.pull_request.number || (github.event_name == 'push' && github.ref == 'refs/heads/main' && 'main') || github.run_id }}")

    def test_gated_pr_workflows_always_start_the_prepare_job(self):
        for filename in [*CODE_WORKFLOWS, "doc-link-checker.yml"]:
            workflow = load(filename)
            with self.subTest(workflow=filename):
                trigger = workflow["on"]["pull_request"]
                self.assertNotIn("paths", trigger)
                self.assertNotIn("paths-ignore", trigger)
                self.assertNotIn("if", workflow["jobs"]["prepare"])
                self.assertNotIn("needs", workflow["jobs"]["prepare"])

    def test_platform_coverage_is_preserved(self):
        for filename, expected in CODE_WORKFLOWS.items():
            workflow = load(filename)
            count = sum(matrix_size(job) for name, job in workflow["jobs"].items()
                        if name not in {"prepare", "ci-gate", "release", "submit-winget"})
            with self.subTest(workflow=filename):
                self.assertEqual(count, expected)

    def test_release_builds_bound_nested_concurrency_without_dropping_targets(self):
        jobs = load("release-build.yml")["jobs"]
        for job, name in (("build", "Run GoReleaser (Build & Test)"),
                          ("release", "Run GoReleaser (Release)")):
            with self.subTest(job=job):
                step = next(step for step in jobs[job]["steps"] if step.get("name") == name)
                command = step["run"].replace("\\\n", " ")
                build = next(line for line in command.splitlines()
                             if "docker run" in line and "release --" in line)
                self.assertIn("-e GOMAXPROCS=2", build)
                self.assertIn("-e GOFLAGS=-p=2", build)
                self.assertIn("--parallelism=1", build)
                self.assertNotIn("--single-target", build)
                self.assertNotIn("continue-on-error", step)
        config = yaml.safe_load((WORKFLOWS.parent.parent / ".goreleaser.yaml").read_text())
        self.assertCountEqual([target for build in config["builds"] for target in build["targets"]],
                              ["darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64"])

    def test_self_host_bounds_builds_without_changing_runtime_acceptance(self):
        job = load("llgo.yml")["jobs"]["self-host"]
        steps = job["steps"]
        named = {step.get("name"): step for step in steps}
        first = named["Build stage 1 with Go"]
        second = named["Build stage 2 with LLGo"]
        self.assertIn('GOMAXPROCS=2 go build -p=2 -o "$stage1_dir/llgo" ./cmd/llgo',
                      first["run"])
        self.assertIn('build=("$stage1" build -p=1 -v -o "$stage2_dir/llgo" ./cmd/llgo)',
                      second["run"])
        self.assertIn('GOMAXPROCS=2 /usr/bin/time -v "${build[@]}"', second["run"])
        self.assertIn('GOMAXPROCS=2 "${build[@]}"', second["run"])
        self.assertIn('[[ "$RUNNER_OS" == Linux && -x /usr/bin/time ]]', second["run"])
        self.assertIn('::warning::GNU time unavailable; running the unchanged bootstrap command',
                      second["run"])
        self.assertIn('"$stage2_dir/llgo" version', second["run"])
        self.assertIn('file "$stage2_dir/llgo"', second["run"])
        for scope in [job, *steps]:
            self.assertFalse({"GOMAXPROCS", "GOFLAGS", "GOMEMLIMIT"} &
                             set(scope.get("env", {})))
        for step in (first, second):
            self.assertNotIn("continue-on-error", step)
            self.assertNotIn("|| true", step["run"])
        smoke = named["Test with stage 2"]["run"]
        demos = named["Run demos with stage 2"]["run"]
        self.assertIn('LLGO_BUILD_CACHE=off llgo test -timeout=10m ./test/std/io/ioutil', smoke)
        self.assertIn("-run '^TestRuntimeGMPLinks$' ./test/llgoext", smoke)
        self.assertIn('LLGO_BUILD_CACHE=off llgo run ./stdlib', demos)
        self.assertIn('LLGO_BUILD_CACHE=off llgo run ./hello', demos)
        for command in (smoke, demos):
            self.assertNotIn("-p=", command)
            self.assertNotIn("GOMAXPROCS", command)

    def test_self_host_retains_before_and_after_resource_evidence(self):
        steps = load("llgo.yml")["jobs"]["self-host"]["steps"]
        named = {step.get("name"): step for step in steps}
        before = named.get("Record self-host resources before build")
        after = named.get("Record self-host resources after build")
        self.assertIsNotNone(before, "self-host lacks actual pre-build resource evidence")
        self.assertIsNotNone(after, "self-host lacks post-failure resource evidence")
        self.assertEqual(before["if"], "runner.os == 'Linux'")
        self.assertEqual(after["if"], "always() && runner.os == 'Linux'")
        self.assertLess(steps.index(before), steps.index(named["Build stage 1 with Go"]))
        self.assertGreater(steps.index(after), steps.index(named["Run demos with stage 2"]))
        self.assertEqual(before["run"], after["run"])
        for source in ("/proc/self/cgroup", "/proc/self/mountinfo", "/proc/meminfo",
                       "memory.events", "memory.max", "memory.current"):
            self.assertIn(source, before["run"])
        self.assertIn("timeout 10s", before["run"])

    def test_windows_release_covers_all_architectures_with_native_x86_toolchains(self):
        jobs = load("release-build.yml")["jobs"]
        for name in ("build-windows", "test-windows-artifacts"):
            matrix = jobs[name]["strategy"]["matrix"]
            self.assertCountEqual(matrix["abi"], ["msvc", "mingw"])
            self.assertCountEqual(matrix["goarch"], ["amd64", "arm64"])
            x86 = jobs[name + "-x86"]["strategy"]["matrix"]
            self.assertCountEqual(x86["abi"], ["msvc", "mingw"])
            self.assertEqual(x86["goarch"], ["386"])
            self.assertEqual(x86["os"], ["windows-2022"])
            self.assertEqual(jobs[name]["steps"], jobs[name + "-x86"]["steps"])
        self.assertIn("build-windows-compiler", needs(jobs["build-windows"]))
        self.assertIn("build-windows-compiler-x86", needs(jobs["build-windows-x86"]))
        compiler = jobs["build-windows-compiler"]
        self.assertCountEqual(compiler["strategy"]["matrix"]["goarch"], ["amd64", "arm64"])
        self.assertEqual(jobs["build-windows-compiler-x86"]["strategy"]["matrix"]["goarch"], ["386"])
        self.assertEqual(compiler["steps"], jobs["build-windows-compiler-x86"]["steps"])
        self.assertNotIn("abi", compiler["strategy"]["matrix"])
        host = next(step for step in compiler["steps"]
                    if step.get("uses") == "./.github/actions/setup-deps")
        self.assertEqual(host["with"]["windows-abi"], "msvc")
        upload = next(step for step in compiler["steps"]
                      if step.get("name") == "Upload shared host compiler")
        download = next(step for step in jobs["build-windows"]["steps"]
                        if step.get("name") == "Download shared host compiler")
        self.assertEqual(upload["with"]["name"], download["with"]["name"])
    def test_windows_release_uses_checksum_pinned_official_toolchains(self):
        jobs = load("release-build.yml")["jobs"]
        self.assertNotIn("build-windows-x86-toolchains", jobs)
        self.assertFalse((WORKFLOWS.parent / "windows/build-x86-toolchains.sh").exists())
        pins = json.loads((WORKFLOWS.parent / "windows/espressif-toolchain.json").read_text())
        self.assertRegex(pins["release"], r"^22\.1\.4_\d{8}$")
        self.assertRegex(pins["producer_commit"], r"^[0-9a-f]{40}$")
        self.assertEqual(pins["sdk"]["version"], "22.1.8")
        self.assertEqual(pins["sdk"]["asset"], "llvm-sdk-22.1.8-i686-w64-mingw32.tar.xz")
        self.assertRegex(pins["sdk"]["sha256"], r"^[0-9a-f]{64}$")
        self.assertEqual(set(pins["payloads"]), {"386", "amd64", "arm64"})
        for arch, target in (("386", "i686"), ("amd64", "x86_64"), ("arm64", "aarch64")):
            pin = pins["payloads"][arch]
            self.assertEqual(pin["asset"], f"clang-esp-{pins['release']}-{target}-w64-mingw32.tar.xz")
            self.assertRegex(pin["sha256"], r"^[0-9a-f]{64}$")
        for name in ("build-windows-compiler", "build-windows-compiler-x86"):
            step = next(step for step in jobs[name]["steps"]
                        if step.get("name") == "Cache Windows x86 LLVM SDK archive")
            self.assertEqual(step["if"], "matrix.goarch == '386'")
            self.assertIn("espressif-toolchain.json", step["with"]["key"])
            self.assertEqual(step["with"]["path"], "${{ runner.tool_cache }}/llgo-release-downloads/llvm-x86")
        for name in ("build-windows", "build-windows-x86"):
            step = next(step for step in jobs[name]["steps"]
                        if step.get("name") == "Cache ESP release archive")
            self.assertNotIn("if", step)
            self.assertIn("espressif-toolchain.json", step["with"]["key"])
        for filename in ("build-compiler.ps1", "build-release.ps1"):
            script = (WORKFLOWS.parent / "windows" / filename).read_text()
            self.assertIn("espressif-toolchain.json", script)
            self.assertIn("https://github.com/xgo-dev/espressif-llvm-project-prebuilt/releases/download/", script)
            self.assertNotIn("Expand-ReleaseLocalXz", script)
            self.assertNotIn("cpunion/espressif", script)

    def test_release_artifact_dependencies_and_tag_guard_are_preserved(self):
        jobs = load("release-build.yml")["jobs"]
        for consumer, producer in {
            "build": "populate-linux-sysroot", "build-windows": "build",
            "test-windows-artifacts": "build-windows", "prepare-winget": "test-windows-artifacts",
            "test-windows-artifacts-x86": "build-windows-x86",
            "test-artifacts": "build", "test-linux-distributions": "build",
        }.items():
            self.assertIn(producer, needs(jobs[consumer]))
        self.assertEqual(set(needs(jobs["release"])), {
            "prepare", "prepare-winget", "test-artifacts", "test-windows-artifacts", "test-windows-artifacts-x86", "test-linux-distributions", "populate-linux-sysroot"})
        self.assertIn("test-windows-artifacts-x86", needs(jobs["prepare-winget"]))
        self.assertIn("startsWith(github.ref, 'refs/tags/')", jobs["release"]["if"])

    def test_linux_distribution_ci_builds_packages_without_publishing(self):
        jobs = load("release-build.yml")["jobs"]
        build = next(step for step in jobs["build"]["steps"]
                     if step.get("name") == "Run GoReleaser (Build & Test)")["run"]
        self.assertIn("--snapshot", build)
        skipped = build.split("--skip=", 1)[1].split()[0].split(",")
        self.assertIn("publish", skipped)
        self.assertNotIn("nfpm", skipped)
        checks = jobs["test-linux-distributions"]
        self.assertEqual({entry["goarch"] for entry in checks["strategy"]["matrix"]["include"]},
                         {"amd64", "arm64"})
        commands = "\n".join(step.get("run", "") for step in checks["steps"])
        self.assertIn("test-packages.sh", commands)
        self.assertIn("docker build", commands)
        self.assertIn("test-installed.sh", commands)
        self.assertNotIn("FURY_TOKEN", str(jobs["build"]))
        self.assertNotIn("FURY_TOKEN", str(checks))

    def test_fury_upload_is_only_enabled_for_stable_tags(self):
        steps = load("release-build.yml")["jobs"]["release"]["steps"]
        release_type = next(step for step in steps if step.get("id") == "release-type")
        upload = next(step for step in steps if step.get("name") == "Upload Linux packages to Fury")
        self.assertEqual(upload["if"], "steps.release-type.outputs.stable == 'true'")
        self.assertEqual(upload["env"]["FURY_TOKEN"], "${{ secrets.FURY_TOKEN }}")
        for tag in ("v1.2.3", "v0.0.0", "v1.2.3-rc.1", "v1.2.3+build", "1.2.3", "v1.2", "main"):
            with self.subTest(tag=tag), tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / "outputs"
                subprocess.run(["bash", "-e", "-c", release_type["run"]], check=True,
                               env={**os.environ, "GITHUB_REF_NAME": tag,
                                    "GITHUB_OUTPUT": str(output)})
                expected = "true" if tag in {"v1.2.3", "v0.0.0"} else "false"
                self.assertEqual(output.read_text(), f"stable={expected}\n")

    def test_winget_submission_follows_stable_release_and_has_a_read_only_preview(self):
        jobs = load("release-build.yml")["jobs"]
        call = jobs["submit-winget"]
        self.assertEqual(set(needs(call)), {"prepare", "release"})
        self.assertIn("github.repository == 'xgo-dev/llgo'", call["if"])
        self.assertIn("needs.release.outputs.stable == 'true'", call["if"])
        self.assertEqual(jobs["release"]["outputs"]["stable"], "${{ steps.release-type.outputs.stable }}")
        self.assertEqual(call["uses"], "./.github/workflows/winget.yml")
        workflow = load("winget.yml")
        self.assertEqual(set(workflow["on"]), {"workflow_call", "workflow_dispatch"})
        self.assertEqual(workflow["permissions"], {"contents": "read"})
        self.assertEqual(workflow["on"]["workflow_dispatch"]["inputs"]["dry_run"]["default"], "true")
        self.assertEqual(workflow["on"]["workflow_call"]["inputs"]["dry_run"]["default"], "false")
        job = workflow["jobs"]["submit"]
        self.assertEqual(job["if"], "github.repository == 'xgo-dev/llgo' || inputs.dry_run")
        step = next(step for step in job["steps"] if step.get("name") == "Submit release manifests")
        self.assertEqual(step["env"]["WINGET_PR_TOKEN"], "${{ !inputs.dry_run && secrets.WINGET_PR_TOKEN || '' }}")
        self.assertNotIn("${{", step["run"])

    def test_fury_upload_validates_packages_and_stops_on_failure(self):
        steps = load("release-build.yml")["jobs"]["release"]["steps"]
        upload = next(step for step in steps if step.get("name") == "Upload Linux packages to Fury")
        packages = [f".dist/llgo1.2.3.linux-{arch}.{fmt}"
                    for arch in ("amd64", "arm64") for fmt in ("deb", "rpm")]
        cases = [(case, "xgo-dev", "bad/account" if case == "invalid-account" else "goplus", "goplus")
                 for case in ("success", "created", "missing-token", "missing-package", "invalid-account",
                              "http-failure", "transport-failure")]
        cases += [("upstream-default", "xgo-dev", "", "goplus"),
                  ("fork-default", "cpunion", "", "cpunion"),
                  ("upstream-override", "xgo-dev", "custom-account", "custom-account"),
                  ("fork-override", "cpunion", "custom-account", "custom-account")]
        success_cases = {"success", "created", "upstream-default", "fork-default",
                         "upstream-override", "fork-override"}
        for case, owner, account, expected_account in cases:
            with self.subTest(case=case), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                (root / ".dist").mkdir()
                for name in packages:
                    if case != "missing-package" or name != packages[-1]:
                        (root / name).write_bytes(b"test package")
                bin_dir = root / "bin"
                bin_dir.mkdir()
                curl = bin_dir / "curl"
                curl.write_text(f"#!{sys.executable}\n" +
                                "import json, os, sys\n"
                                "with open(os.environ['UPLOAD_LOG'], 'a') as log:\n"
                                "    log.write(json.dumps(sys.argv[1:]) + '\\n')\n"
                                "print(os.environ['CURL_STATUS'], end='')\n"
                                "sys.exit(int(os.environ['CURL_EXIT']))\n")
                curl.chmod(0o755)
                log = root / "uploads"
                result = subprocess.run(["bash", "-c", upload["run"]], cwd=root,
                                        capture_output=True, text=True, env={**os.environ,
                                            "PATH": str(bin_dir) + os.pathsep + os.environ["PATH"],
                                            "FURY_TOKEN": "" if case == "missing-token" else "test-only-token",
                                            "FURY_ACCOUNT": account,
                                            "GITHUB_REPOSITORY_OWNER": owner,
                                            "GITHUB_REF_NAME": "v1.2.3",
                                            "UPLOAD_LOG": str(log),
                                            "CURL_STATUS": "503" if case == "http-failure" else "201" if case == "created" else "200",
                                            "CURL_EXIT": "7" if case == "transport-failure" else "0"})
                calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
                self.assertEqual(result.returncode == 0, case in success_cases, result.stderr)
                self.assertNotIn("test-only-token", result.stdout + result.stderr)
                expected_calls = 4 if case in success_cases else 1 if case.endswith("failure") else 0
                self.assertEqual(len(calls), expected_calls)
                for package, args in zip(packages, calls):
                    self.assertIn(f"file=@{package}", args)
                    self.assertIn("public=true", args)
                    self.assertIn("Authorization: test-only-token", args)
                    self.assertEqual(args[-1], f"https://push.fury.io/uploads?as={expected_account}")

    def test_runner_policy_is_not_duplicated_in_workflows(self):
        for path in workflow_paths():
            workflow = load(path.name)
            for name, job in workflow["jobs"].items():
                with self.subTest(workflow=path.name, job=name):
                    runner = str(job.get("runs-on", ""))
                    if (path.name, name) not in BOOTSTRAP_RUNNER_JOBS:
                        self.assertNotIn("qiniu", runner)
                        self.assertNotIn("github.repository_owner", runner)
                    if "needs.prepare" in runner:
                        self.assertIn("prepare", needs(job))

    def test_bootstrap_runners_match_the_policy_without_waiting_for_outputs(self):
        upstream = ci_policy.runners("xgo-dev")["linux_runner"]
        fork = ci_policy.runners("cpunion")["linux_runner"]
        expected = ("${{ fromJSON(github.repository_owner == 'xgo-dev' && '" +
                    upstream + "' || '" + fork + "') }}")
        for filename, name in BOOTSTRAP_RUNNER_JOBS:
            with self.subTest(workflow=filename, job=name):
                job = load(filename)["jobs"][name]
                self.assertEqual(job["runs-on"], expected)
                self.assertNotIn("needs", job)

    def test_linux_environment_checks_keep_native_arm64_and_fork_fallbacks(self):
        for filename in ("mise.yml", "pixi.yml"):
            workflow = load(filename)
            job = workflow["jobs"]["devenv"]
            with self.subTest(workflow=filename):
                self.assertEqual(workflow["jobs"]["prepare"]["uses"], PREPARE)
                self.assertIn("prepare", needs(job))
                self.assertEqual(job["if"], "needs.prepare.outputs.run_code_ci == 'true'")
                self.assertEqual(job["runs-on"],
                                 "${{ matrix.os == 'ubuntu-24.04' && fromJSON(needs.prepare.outputs.linux_large_runner) || matrix.os }}")
                self.assertCountEqual(job["strategy"]["matrix"]["os"],
                                      ["ubuntu-24.04", "ubuntu-24.04-arm", "macos-15", "macos-15-intel", "windows-2025"])

    def test_release_artifact_runner_migration_keeps_native_architectures(self):
        jobs = load("release-build.yml")["jobs"]
        self.assertEqual(jobs["prepare-winget"]["runs-on"],
                         "${{ fromJSON(needs.prepare.outputs.linux_runner) }}")
        job = jobs["test-artifacts"]
        self.assertEqual(job["runs-on"],
                         "${{ matrix.os == 'ubuntu-latest' && fromJSON(needs.prepare.outputs.linux_large_runner) || matrix.os }}")
        linux = {(entry["os"], entry["goarch"]) for entry in job["strategy"]["matrix"]["include"]
                 if entry["goos"] == "linux"}
        self.assertEqual(linux, {("ubuntu-latest", "amd64"), ("ubuntu-24.04-arm", "arm64")})

    def test_linux_container_jobs_use_large_runners_and_check_privileges(self):
        jobs = load("release-build.yml")["jobs"]
        for name in ("populate-linux-sysroot", "build"):
            self.assertEqual(jobs[name]["runs-on"],
                             "${{ fromJSON(needs.prepare.outputs.linux_large_runner) }}")
            self.assertIn("prepare", needs(jobs[name]))
        probe = jobs["populate-linux-sysroot"]["steps"][0]
        self.assertNotIn("if", probe)  # A sysroot cache hit must still check the new runner.
        self.assertIn("docker run --rm --privileged", probe["run"])
        self.assertIn("mount -t tmpfs", probe["run"])
        self.assertNotIn("|| true", probe["run"])
        self.assertEqual(jobs["test-linux-distributions"]["runs-on"],
                         "${{ matrix.goarch == 'amd64' && fromJSON(needs.prepare.outputs.linux_large_runner) || matrix.os }}")
        self.assertEqual(jobs["test-linux-distributions"]["strategy"]["matrix"]["include"],
                         [{"os": "ubuntu-24.04", "goarch": "amd64"}, {"os": "ubuntu-24.04-arm", "goarch": "arm64"}])
        self.assertEqual(jobs["release"]["runs-on"], "ubuntu-latest")

    def test_fedora_shell_exit_uses_standard_host_without_reducing_coverage(self):
        workflow = load("installer-shells.yml")
        self.assertEqual(workflow["jobs"]["prepare"]["uses"], PREPARE)
        installer = workflow["jobs"]["installer-shells"]
        self.assertIn("prepare", needs(installer))
        self.assertNotIn("if", installer)
        self.assertNotIn("continue-on-error", installer)
        self.assertEqual(installer["timeout-minutes"], "20")
        self.assertEqual(installer["runs-on"],
                         "${{ matrix.runner || fromJSON(needs.prepare.outputs.linux_large_runner) }}")
        strategy = installer["strategy"]
        self.assertEqual(strategy["fail-fast"], "false")
        matrix = strategy["matrix"]
        self.assertNotIn("exclude", matrix)
        self.assertCountEqual(matrix["image"],
                              ["ubuntu:24.04", "debian:12", "fedora:latest", "archlinux:latest", "alpine:latest", "opensuse/tumbleweed:latest"])
        self.assertEqual(matrix["include"],
                         [{"image": "fedora:latest", "runner": "ubuntu-24.04"}])
        self.assertEqual(matrix_size(installer), 6)
        steps = installer["steps"]
        self.assertEqual(len(steps), 2)
        startup = steps[1]
        self.assertNotIn("if", startup)
        self.assertNotIn("continue-on-error", startup)
        self.assertEqual(startup["env"], {"INSTALLER_TEST_IMAGE": "${{ matrix.image }}"})
        self.assertEqual(startup["run"],
                         'bash test/installer/install_shell_distro_test.sh "$INSTALLER_TEST_IMAGE"')

    def test_prepare_exports_each_decision_and_has_no_caller_concurrency(self):
        workflow = load("ci-prepare.yml")
        self.assertNotIn("concurrency", workflow)
        job = workflow["jobs"]["prepare"]
        outputs = workflow["on"]["workflow_call"]["outputs"]
        self.assertEqual(set(outputs), {"run_code_ci", "run_doc_checks", "pr_head_sha",
                                       "pr_merge_base", "linux_runner", "linux_large_runner"})
        for key in outputs:
            self.assertEqual(outputs[key]["value"], f"${{{{ jobs.prepare.outputs.{key} }}}}")
            self.assertEqual(job["outputs"][key], f"${{{{ steps.policy.outputs.{key} }}}}")
        checkout = job["steps"][0]
        self.assertEqual(checkout["with"]["fetch-depth"],
                         "${{ github.event_name == 'pull_request' && '0' || '1' }}")
        self.assertGreater(int(checkout["env"]["GIT_HTTP_LOW_SPEED_LIMIT"]), 0)
        self.assertGreater(int(checkout["env"]["GIT_HTTP_LOW_SPEED_TIME"]), 0)
        retry_budget = 3 * int(checkout["env"]["GIT_HTTP_LOW_SPEED_TIME"]) + 2 * 20
        self.assertLess(retry_budget, int(job["timeout-minutes"]) * 60)
        self.assertEqual(workflow["permissions"], {"contents": "read"})

    def test_pr_policy_comes_from_base_and_missing_base_enables_all_checks(self):
        step = load("ci-prepare.yml")["jobs"]["prepare"]["steps"][1]
        self.assertEqual(step["env"]["PR_BASE_SHA"], "${{ github.event.pull_request.base.sha }}")
        self.assertEqual(step["env"]["PR_HEAD_SHA"], "${{ github.event.pull_request.head.sha }}")
        for case in ("base-policy", "missing-base-policy", "docs-only"):
            with self.subTest(case=case), tempfile.TemporaryDirectory() as directory:
                repo = Path(directory)

                def git(*args):
                    return subprocess.check_output(["git", *args], cwd=repo,
                                                   text=True, stderr=subprocess.PIPE).strip()

                git("init", "-q")
                git("config", "user.name", "CI Test")
                git("config", "user.email", "ci@example.invalid")
                (repo / "README.md").write_text("base\n")
                scripts = repo / ".github" / "scripts"
                if case != "missing-base-policy":
                    scripts.mkdir(parents=True)
                    for name in ("ci_policy.py", "ci_changes.py"):
                        (scripts / name).write_text((Path(__file__).parent / name).read_text())
                git("add", ".")
                git("commit", "-qm", "base")
                base = git("rev-parse", "HEAD")
                historical_blob = git("rev-parse", "HEAD:README.md")

                if case == "docs-only":
                    (repo / "README.md").write_text("updated prose\n")
                else:
                    scripts.mkdir(parents=True, exist_ok=True)
                    # A PR-controlled policy would suppress code checks here.
                    (scripts / "ci_policy.py").write_text(
                        "import os\nopen(os.environ['GITHUB_OUTPUT'], 'a').write('run_code_ci=false\\n')\n")
                    (repo / "compiler.go").write_text("package compiler\n")
                git("add", ".")
                git("commit", "-qm", "pr")
                if case == "base-policy":
                    (repo / "README.md").write_text("final documentation edit\n")
                    git("add", ".")
                    git("commit", "-qm", "docs after code")
                head = git("rev-parse", "HEAD")

                # Exercise checkout's transport, including lazy retrieval of
                # the trusted policy when the PR replaces its current blob.
                git("config", "uploadpack.allowFilter", "true")
                git("config", "uploadpack.allowAnySHA1InWant", "true")
                remote = repo.as_uri()
                repo = repo / "checkout"
                repo.mkdir()
                git("init", "-q")
                git("config", "protocol.file.allow", "always")
                git("remote", "add", "origin", remote)
                fetch = ["fetch", "--quiet", "--no-tags"]
                configured_filter = load("ci-prepare.yml")["jobs"]["prepare"]["steps"][0]["with"].get("filter")
                if configured_filter:
                    fetch.append("--filter=" + configured_filter)
                git(*fetch, "origin", "+refs/heads/*:refs/remotes/origin/*")
                git("checkout", "-q", "--detach", head)
                self.assertEqual(git("rev-parse", "--is-shallow-repository"), "false")
                if case != "missing-base-policy":
                    self.assertIn("?" + historical_blob,
                                  git("rev-list", "--objects", "--all", "--missing=print"))

                event = repo / "event.json"
                event.write_text(json.dumps({"pull_request": {
                    "base": {"sha": base}, "head": {"sha": head}}}))
                output = repo / "outputs"
                summary = repo / "summary"
                subprocess.run(["bash", "-e", "-o", "pipefail", "-c", step["run"]],
                               cwd=repo, check=True, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE, text=True, env={**os.environ,
                                   "GITHUB_EVENT_NAME": "pull_request",
                                   "GITHUB_EVENT_PATH": str(event),
                                   "GITHUB_REPOSITORY_OWNER": "cpunion",
                                   "GITHUB_WORKSPACE": str(repo),
                                   "GITHUB_OUTPUT": str(output),
                                   "GITHUB_STEP_SUMMARY": str(summary),
                                   "RUNNER_TEMP": directory,
                                   "PR_BASE_SHA": base, "PR_HEAD_SHA": head})
                values = dict(line.split("=", 1) for line in output.read_text().splitlines())
                self.assertEqual(values["run_code_ci"],
                                 "false" if case == "docs-only" else "true")
                self.assertEqual(values["run_doc_checks"], "true")
                self.assertEqual(values["pr_head_sha"], head)
                self.assertEqual(values["pr_merge_base"], base)

    def test_preparation_failures_are_reported_unless_workflow_is_cancelled(self):
        names = set()
        for filename in [*CODE_WORKFLOWS, "doc-link-checker.yml", "model-demo.yml"]:
            gate = load(filename)["jobs"]["ci-gate"]
            with self.subTest(workflow=filename):
                self.assertNotIn(gate["name"], names)
                names.add(gate["name"])
                self.assertEqual(needs(gate), ["prepare"])
                self.assertEqual(gate["if"], "always() && !cancelled()")
                # On success the gate shares the qiniu pool. A failed prepare
                # has no runner output, so keep its diagnostic job schedulable.
                self.assertEqual(gate["runs-on"],
                                 "${{ fromJSON(needs.prepare.outputs.linux_runner || '[\"ubuntu-24.04\"]') }}")
                self.assertIn("needs.prepare.result", gate["steps"][0]["run"])
                self.assertIn('= "success"', gate["steps"][0]["run"])

    def test_policy_tests_cannot_skip_themselves(self):
        workflow = load("ci-policy.yml")
        self.assertNotIn("paths", workflow["on"]["pull_request"])
        job = workflow["jobs"]["policy"]
        self.assertNotIn("if", job)
        self.assertNotIn("needs", job)
        self.assertTrue(any("unittest discover" in step.get("run", "") for step in job["steps"]))

    def test_benchmarks_use_the_prepared_revision_pair(self):
        for name in ("benchmark", "wasm-benchmark"):
            steps = load("benchmark.yml")["jobs"][name]["steps"]
            for step in steps:
                self.assertNotIn("git merge-base", step.get("run", ""))
                options = step.get("with", {})
                path = options.get("path", "")
                if path.endswith("base-source"):
                    self.assertEqual(options["ref"], "${{ needs.prepare.outputs.pr_merge_base }}")
                if path.endswith("head-source"):
                    self.assertEqual(options["ref"], "${{ needs.prepare.outputs.pr_head_sha }}")

    def test_main_external_benchmark_dispatch_has_no_change_filter(self):
        job = load("notify-benchmarks.yml")["jobs"]["dispatch"]
        request = next(step for step in job["steps"] if step.get("name", "").startswith("Request benchmarks"))
        self.assertNotIn("if", request)
        self.assertEqual(job["if"], "github.repository == 'xgo-dev/llgo' && github.event_name != 'pull_request'")

    def test_wasm_baseline_metadata_requires_a_successful_measurement(self):
        steps = load("benchmark.yml")["jobs"]["wasm-benchmark"]["steps"]
        record = next(step for step in steps
                      if step.get("name") == "Record WebAssembly benchmark result")
        for key in ("baseline-benchmark-file", "baseline-repository", "baseline-sha", "baseline-ref"):
            with self.subTest(input=key):
                expression = " ".join(record["with"][key].split())
                self.assertTrue(expression.startswith(
                    "${{ github.event_name == 'pull_request' && "
                    "steps.measure-wasm-base.outcome == 'success' && "))
                self.assertTrue(expression.endswith(" || '' }}"))

    def test_publish_jobs_require_their_own_artifacts(self):
        jobs = load("benchmark-publish.yml")["jobs"]
        self.assertEqual(jobs["artifacts"]["if"], "github.event.workflow_run.conclusion == 'success'")
        for name, suite in (("publish", "baseline"), ("publish-wasm", "wasm")):
            self.assertIn("artifacts", needs(jobs[name]))
            self.assertEqual(jobs[name]["if"], f"needs.artifacts.outputs.{suite} == 'true'")

    @unittest.skipUnless(shutil.which("node"), "Node is needed to exercise github-script")
    def test_artifact_probe_handles_empty_partial_and_expired_results(self):
        script = load("benchmark-publish.yml")["jobs"]["artifacts"]["steps"][0]["with"]["script"]
        cases = [[], [{"name": "unrelated", "expired": False}],
                 [{"name": "go-benchmark-llgo-baseline-linux", "expired": True}],
                 [{"name": "go-benchmark-llgo-baseline-linux", "expired": False}],
                 [{"name": "go-benchmark-llgo-wasm-linux", "expired": False}],
                 [{"name": "go-benchmark-llgo-baseline-linux", "expired": False},
                  {"name": "go-benchmark-llgo-wasm-linux", "expired": False}]]
        harness = """
          const { script, cases } = JSON.parse(require('fs').readFileSync(0, 'utf8'));
          const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
          (async () => {
            const results = [];
            for (const artifacts of cases) {
              const output = {};
              const github = {
                rest: { actions: { listWorkflowRunArtifacts: 'list' } },
                paginate: async (method, args) => {
                  if (method !== 'list' || args.run_id !== 123 || args.per_page !== 100)
                    throw new Error('must paginate artifacts from the triggering run');
                  return artifacts;
                }
              };
              await new AsyncFunction('github', 'context', 'core', script)(github,
                { repo: { owner: 'xgo-dev', repo: 'llgo' }, payload: { workflow_run: { id: 123 } } },
                { setOutput: (key, value) => { output[key] = value; }, info: () => {} });
              results.push(output);
            }
            console.log(JSON.stringify(results));
          })().catch(error => { console.error(error); process.exitCode = 1; });
        """
        result = subprocess.run(["node", "-e", harness],
                                input=json.dumps({"script": script, "cases": cases}),
                                text=True, capture_output=True, check=True)
        self.assertEqual(json.loads(result.stdout), [
            {"baseline": "false", "wasm": "false"}, {"baseline": "false", "wasm": "false"},
            {"baseline": "false", "wasm": "false"}, {"baseline": "true", "wasm": "false"},
            {"baseline": "false", "wasm": "true"}, {"baseline": "true", "wasm": "true"},
        ])


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3

"""Exercise the default WASI pthread backend under Wasmer."""

import json
import os
import pathlib
import re
import shutil
import subprocess
import tempfile
import time


ROOT = pathlib.Path(__file__).resolve().parent.parent
LLGO = os.environ.get("LLGO", "llgo")
WASMER = os.environ.get("WASMER", "wasmer")


def wasmer_command(module, *args):
    return [WASMER, "run", "--enable-exceptions", "--enable-simd",
            "--stack-size=1048576", "--volume=" + str(ROOT), "--volume=/tmp",
            str(module), "--", *args]


def run_output_cache_probe(env, directory):
    module = pathlib.Path(directory) / "runner-output.wasm"
    cache = pathlib.Path(directory) / "runner-cache"
    subprocess.run([os.environ.get("WASM_TOOLS", "wasm-tools"), "parse",
                    str(ROOT / "internal/build/testdata/wasm-wasi-runner-output/main.wat"),
                    "-o", str(module)], check=True, timeout=30)
    command = wasmer_command(module)
    command[2:2] = ["--cache-dir", str(cache)]
    previous_cache = None
    for state in ("cold", "warm"):
        result = subprocess.run(command, env=env, capture_output=True, timeout=30)
        expected_stderr = b'{"level":"WARN","target":"wasmer","fields":{"message":"guest stderr"}}\n'
        if (result.returncode != 7 or result.stdout != b"guest stdout\n"
                or result.stderr != expected_stderr):
            raise SystemExit(f"Wasmer {state} cache changed guest output/status: {result}")
        artifacts = {str(p.relative_to(cache)): (p.stat().st_size, p.stat().st_mtime_ns)
                     for p in cache.rglob("*.bin")}
        if not artifacts or (previous_cache is not None and artifacts != previous_cache):
            raise SystemExit(f"Wasmer {state} module cache was not reused: {artifacts}")
        previous_cache = artifacts
    print("wasi cold/warm cache and guest stdout/stderr/exit status ok", flush=True)


def run_probe(env, directory, name, fixture, tags, marker, timeout,
              expected_exit=0, args=(), runs=1):
    module = pathlib.Path(directory) / f"{name}.wasm"
    command = [LLGO, "build", "-target", "wasi"]
    if tags:
        command += ["-tags", tags]
    command += ["-o", str(module), str(ROOT / "internal/build/testdata" / fixture)]
    subprocess.run(
        command,
        check=True,
        env=env,
        timeout=180,
    )
    allowed_exits = (expected_exit,) if isinstance(expected_exit, int) else expected_exit
    for attempt in range(runs):
        result = subprocess.run(
            wasmer_command(module, *args),
            env=env, capture_output=True,
            text=True,
            timeout=timeout,
        )
        print(result.stdout, end="")
        print(result.stderr, end="")
        markers = (marker,) if isinstance(marker, str) else marker
        lines = result.stdout.splitlines() + result.stderr.splitlines()
        if result.returncode not in allowed_exits or any(m not in lines for m in markers):
            raise SystemExit(
                f"Wasmer {name} probe {attempt + 1}/{runs} failed "
                f"with exit code {result.returncode}")


def run_llgo(env, args, marker, timeout=180):
    result = subprocess.run([LLGO, *args], capture_output=True, text=True,
                            env=env, timeout=timeout)
    print(result.stdout, end="")
    print(result.stderr, end="")
    if result.returncode != 0 or (
        marker not in result.stdout.splitlines()
        and marker not in result.stderr.splitlines()
    ):
        raise SystemExit(f"LLGo {' '.join(args)} failed with exit code {result.returncode}")


def check_size_probe(env, module):
    if b"unusedFunctionMetadata" in module.read_bytes():
        raise SystemExit(f"{module.name}: dead function metadata survived")
    source = ROOT / "internal/build/testdata/wasm-size/main.go"
    panic_line = next(i for i, line in enumerate(source.read_text().splitlines(), 1)
                      if line.startswith("func panicSite()"))
    for args, expected_exit, marker in (
        ((), 0, 'effects=2 map=5 value=0000002a float=1.25 text="中"'),
        (("panic",), 2, "panic: size-probe-panic"),
    ):
        result = subprocess.run(wasmer_command(module, *args), env=env,
                                capture_output=True, text=True, timeout=60)
        output = result.stdout + result.stderr
        print(output, end="", flush=True)
        if result.returncode != expected_exit or marker not in output.splitlines():
            raise SystemExit(f"{module.name} {args}: exit {result.returncode}")
        if args and ("main.panicSite" not in output
                     or f"/wasm-size/main.go:{panic_line}" not in output):
            raise SystemExit(f"{module.name}: panic source location lost")


def run_arena_boundaries(env, directory):
    module = pathlib.Path(directory) / "gc-arena.wasm"
    subprocess.run(
        [LLGO, "build", "-target", "wasi", "-o", str(module),
         str(ROOT / "internal/build/testdata/wasm-wasi-gc-arena")],
        check=True, env=env, timeout=180,
    )
    # Each size needs a fresh heap: an earlier oversized arena could mask
    # the bug by satisfying a later allocation from its unused capacity.
    for size in ((32 << 20) - (128 << 10), (32 << 20) - 1,
                 32 << 20, (32 << 20) + 1, 33 << 20):
        result = subprocess.run(
            wasmer_command(module, str(size)), env=env, capture_output=True, text=True, timeout=180,
        )
        print(f"Wasmer arena boundary: {size} bytes")
        print(result.stdout, end="")
        print(result.stderr, end="")
        lines = result.stdout.splitlines() + result.stderr.splitlines()
        if result.returncode != 0 or "wasi gc arena boundary ok" not in lines:
            raise SystemExit(f"Wasmer arena boundary {size} failed: {result.returncode}")


def run_memory_config_probe(env, directory):
    # Budget assertions require a known linker policy, including the negative
    # case. Inherited flags or @response files must not override these targets.
    probe_env = dict(env, CCFLAGS="", CFLAGS="", LDFLAGS="")
    # Reuse one target name to exercise a changed link policy with warm package
    # caches. The final module, including imported shared memory, is authoritative.
    with tempfile.NamedTemporaryFile(dir=ROOT / "targets", prefix="wasi-memory-",
                                     suffix=".json", delete=False) as file:
        target = pathlib.Path(file.name)
    try:
        for initial, maximum, stack in ((16 << 20, 128 << 20, 1 << 20),
                                         (8 << 20, 64 << 20, 2 << 20)):
            target.write_text(json.dumps({
                "inherits": ["wasip1"], "wasm-initial-memory": initial,
                "wasm-max-memory": maximum, "wasm-stack-size": stack,
            }), encoding="utf-8")
            module = pathlib.Path(directory) / "memory-config.wasm"
            build_env = dict(probe_env, LDFLAGS="-Wl,--export=__stack_low,--export=__stack_high")
            subprocess.run([
                LLGO, "build", "-target", target.stem,
                "-o", str(module), str(ROOT / "internal/build/testdata/wasm-wasi-thread-startup"),
            ], check=True, env=build_env, timeout=180)
            wat = subprocess.run([os.environ.get("WASM_TOOLS", "wasm-tools"), "print", str(module)],
                                 check=True, capture_output=True, text=True, timeout=30).stdout
            memory = re.search(r'\(import "env" "memory" \(memory (?:\$\S+ )?\(;\d+;\) (\d+) (\d+) shared\)\)', wat)
            if not memory or tuple(map(int, memory.groups())) != (initial // 65536, maximum // 65536):
                raise SystemExit("WASI target memory limits did not reach the final module")
            globals_by_id = {}
            for match in re.finditer(r'\(global (?:(\$\S+) )?\(;(\d+);\) i32 \(?i32.const (\d+)\)?\)', wat):
                name, index, value = match.groups()
                globals_by_id[index] = globals_by_id[name] = int(value)
            bounds = []
            for name in ("__stack_low", "__stack_high"):
                match = re.search(r'\(export "' + name + r'" \(global ([^)]+)\)\)', wat)
                if not match or match.group(1) not in globals_by_id:
                    raise SystemExit(f"missing WASI process-stack bound {name}")
                bounds.append(globals_by_id[match.group(1)])
            if bounds[1] - bounds[0] != stack:
                raise SystemExit(f"WASI process stack is {bounds}, expected {stack} bytes")
            result = subprocess.run(wasmer_command(module), env=env, capture_output=True,
                                    text=True, timeout=30)
            if result.returncode != 0 or "wasi thread startup ok" not in (result.stdout + result.stderr).splitlines():
                raise SystemExit(f"configured WASI pthread startup failed: {result}")
            print(f"wasi target memory: initial={initial}, max={maximum}, stack={stack} ok", flush=True)
        target.write_text(json.dumps({
            "inherits": ["wasi"], "wasm-initial-memory": 65536,
            "wasm-stack-size": 1048576,
        }), encoding="utf-8")
        result = subprocess.run([
            LLGO, "build", "-target", target.stem,
            "-o", str(pathlib.Path(directory) / "too-small.wasm"),
            str(ROOT / "internal/build/testdata/wasm-wasi-thread-startup"),
        ], env=probe_env, capture_output=True, text=True, timeout=180)
        if result.returncode == 0 or "initial memory too small" not in result.stderr:
            raise SystemExit(f"WASI accepted an insufficient initial-memory budget: {result}")
        print("wasi target memory rejects an insufficient initial budget", flush=True)
        run_inherited_memory_overrides(probe_env, directory, target)
    finally:
        target.unlink()


def run_inherited_memory_overrides(env, directory, parent):
    parent.write_text(json.dumps({"inherits": ["wasi"], "wasm-initial-memory": 16 << 20,
                                  "wasm-max-memory": 128 << 20, "wasm-stack-size": 1 << 20}), encoding="utf-8")
    with tempfile.NamedTemporaryFile(dir=ROOT / "targets", prefix="wasi-memory-child-",
                                     suffix=".json", delete=False) as file:
        child = pathlib.Path(file.name)
    try:
        child.write_text(json.dumps({"inherits": [parent.stem], "wasm-max-memory": 8 << 20}), encoding="utf-8")
        response = pathlib.Path(directory) / "memory.rsp"
        response.write_text("--initial-memory=8388608", encoding="utf-8")
        response_flag = "-Wl,@" + response.as_posix().replace(" ", "\\ ")
        for name, flags, success in (
            ("conflict", "", False),
            ("initial", "-Wl,--initial-memory=8388608", True),
            ("heap", "-Wl,--initial-heap=1048576", True),
            ("response", response_flag, True),
        ):
            module = pathlib.Path(directory) / f"memory-override-{name}.wasm"
            result = subprocess.run([LLGO, "build", "-target", child.stem,
                                     "-o", str(module), str(ROOT / "internal/build/testdata/wasm-wasi-thread-startup")],
                                    env=dict(env, LDFLAGS=flags), capture_output=True, text=True, timeout=180)
            if success:
                if result.returncode != 0:
                    raise SystemExit(f"inherited WASI budget override {name} rejected: {result}")
                result = subprocess.run(wasmer_command(module), env=env, capture_output=True, text=True, timeout=30)
                if result.returncode != 0 or "wasi thread startup ok" not in (result.stdout + result.stderr).splitlines():
                    raise SystemExit(f"inherited WASI budget override {name} failed: {result}")
            elif result.returncode == 0 or "maximum memory too small" not in result.stderr:
                raise SystemExit(f"inherited WASI budget conflict did not reach wasm-ld: {result}")
            print(f"wasi inherited memory override {name} ok", flush=True)
    finally:
        child.unlink()


def main():
    wasmer = shutil.which(WASMER)
    if wasmer is None:
        raise SystemExit(f"Wasmer runner not found: {WASMER}")

    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    # Keep inherited Rust diagnostics out of guest-output assertions.
    env["RUST_LOG"] = "off"
    env.pop("LLGO_WASI_THREADS", None)
    env["PATH"] = str(pathlib.Path(wasmer).resolve().parent) + os.pathsep + env["PATH"]
    with tempfile.TemporaryDirectory(prefix="llgo-wasi-threads-") as directory:
        module = pathlib.Path(directory) / "size.wasm"
        subprocess.run([LLGO, "build", "-target", "wasi", "-o", str(module),
                        str(ROOT / "internal/build/testdata/wasm-size")],
                       check=True, env=env, timeout=300)
        check_size_probe(env, module)
        run_memory_config_probe(env, directory)
        run_output_cache_probe(env, directory)
        simd = pathlib.Path(directory) / "simd-threads-eh.wasm"
        subprocess.run([os.environ.get("WASM_TOOLS", "wasm-tools"), "parse",
                        str(ROOT / "internal/build/testdata/wasm-wasi-simd/threads.wat"),
                        "-o", str(simd)], check=True, timeout=30)
        subprocess.run(wasmer_command(simd), env=env, check=True, timeout=30)
        print("wasi SIMD/thread/standard-EH boundary ok", flush=True)
        for tags in ("nogc", ""):
            run_probe(env, directory, f"startup-{tags or 'gc'}", "wasm-wasi-thread-startup", tags,
                      "wasi thread startup ok", 30)
        # Accept a host error (1) or the guest fatal status (2), never success.
        deadlock_exits = (1, 2)
        # LLVM lowers both Go defer/Goexit and C setjmp/longjmp through standard
        # Wasm EH. A caught exception must not terminate unrelated pthreads.
        for tags in ("nogc", ""):
            suffix = tags or "gc"
            for mode in ("return", "goexit"):
                run_probe(env, directory, f"thread-exit-{mode}-{suffix}",
                          ROOT / "test/wasm/wasi-thread-exit", tags,
                          "wasi thread exit resources ok", 60, args=(mode,), runs=3)
            run_probe(env, directory, f"deferred-goexit-{suffix}",
                      "wasm-wasi-goexit-defer", tags,
                      "wasi worker defer ok", 30, runs=10)
            for mode in ("init", "main"):
                run_probe(env, directory, f"{mode}-defer-{suffix}",
                          "wasm-wasi-goexit-defer", tags,
                          (f"wasi {mode} defer ok",
                           "fatal error: no goroutines (main called runtime.Goexit) - deadlock!"),
                          30, expected_exit=deadlock_exits, args=(mode,))
            run_probe(env, directory, f"uncaught-{suffix}",
                      "wasm-wasi-goexit-defer", tags,
                      "panic: wasi uncaught sentinel", 30,
                      expected_exit=deadlock_exits, args=("uncaught",))
        # A raw exception escaping the Wasm entry point must still fail. Go's
        # unrecovered panic exits explicitly, so it cannot test this boundary.
        uncaught = pathlib.Path(directory) / "uncaught-eh.wasm"
        subprocess.run([os.environ.get("WASM_TOOLS", "wasm-tools"), "parse",
                        str(ROOT / "internal/build/testdata/wasm-wasi-goexit-defer/uncaught.wat"),
                        "-o", str(uncaught)], check=True, timeout=30)
        result = subprocess.run(wasmer_command(uncaught), env=env, capture_output=True,
                                text=True, timeout=30)
        if result.returncode == 0 or "Uncaught exception with payload: [I32(42)]" not in result.stdout + result.stderr:
            raise SystemExit(f"Wasmer swallowed an escaping exception: {result}")
        run_probe(env, directory, "main-goexit", "wasm-wasi-main-goexit", "nogc",
                  "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=deadlock_exits)
        run_probe(env, directory, "main-goexit-gc", "wasm-wasi-main-goexit", "",
                  "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=deadlock_exits)
        run_probe(env, directory, "main-goexit-collect",
                  "wasm-wasi-main-goexit-gc", "", "wasi goexit gc collected",
                  30, expected_exit=deadlock_exits)
        run_probe(env, directory, "main-goexit-timer", "wasm-wasi-main-goexit-timer",
                  "nogc", "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=deadlock_exits)
        run_probe(env, directory, "main-goexit-timer-gc", "wasm-wasi-main-goexit-timer",
                  "", "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=deadlock_exits)
        run_probe(env, directory, "threads", "wasm-wasi-threads", "nogc",
                  "wasi threads ok", 30)
        run_probe(env, directory, "threaded-gc", "wasm-wasi-threaded-gc",
                  "", "wasi threaded gc ok", 180)
        run_llgo(env, ["test", "-v", "-count=1", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/runtime"), str(ROOT / "test/std/sync")],
                 "PASS", timeout=300)
        run_arena_boundaries(env, directory)
        for _ in range(2):
            # The second build exercises the bridge-enabled package cache.
            run_llgo(env, ["run", "-target", "wasi", "-emulator",
                           str(ROOT / "test/wasm/reflect-method-value")],
                     "wasm reflect method values: 42 42 42 42 42")
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threads")],
                 "wasi threads ok")
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threaded-fs")],
                 "wasi threaded filesystem ok")
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/errors")], "PASS")
        # Compiling test/go on a cold CI runner and running the GC race are
        # separate budgets. Verbose, inherited output identifies a slow test
        # immediately instead of discarding it when subprocess.run times out.
        module = pathlib.Path(directory) / "runtime-gc-tests.wasm"
        started = time.monotonic()
        subprocess.run([LLGO, "test", "-c", "-target", "wasi", "-o", str(module),
                        str(ROOT / "test/go")], env=env, check=True, timeout=300)
        print(f"Wasmer GC test compilation: {time.monotonic() - started:.2f}s", flush=True)
        # The two startup shapes each race 20 goroutines against continuous
        # full GC. Give each an independent runtime/deadline, and keep finalizer,
        # callback GC and first-use symbol lookup in a third invocation. All
        # cases and repetition counts remain enabled; a stalled case still
        # fails within 300 seconds with its last active test visible.
        runtime_cases = (
            ("finalizers/callback/symbols",
             "^(TestRuntimeSetFinalizer.*|TestReflectMakeFuncGoroutineGC|TestRuntimeFuncInfoConcurrentFirstUse)$"),
            ("startup-pointer", "^TestReflectMakeFuncGoroutineStartup$/^pointer_argument$"),
            ("startup-zero", "^TestReflectMakeFuncGoroutineStartup$/^zero_arguments$"),
        )
        for name, pattern in runtime_cases:
            started = time.monotonic()
            subprocess.run(wasmer_command(module, "-test.v", "-test.run=" + pattern),
                           env=env, check=True, timeout=300)
            print(f"Wasmer GC test {name}: {time.monotonic() - started:.2f}s", flush=True)
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator", "-run",
                       "^TestPoolAfterGC$", str(ROOT / "test/std/sync")], "PASS",
                 timeout=300)
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/go/importer")], "PASS")
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator", "-run",
                       "^(TestTBasicMethods|FuzzExample)$",
                       str(ROOT / "test/std/testing")], "PASS")
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/weak")], "PASS")
        run_llgo(env, ["test", "-v", "-target", "wasi", "-emulator",
                       "-run", "^TestConcurrentSelectProposeReplyStress$",
                       str(ROOT / "test")], "PASS")
        subprocess.run(
            ["go", "run", "./dev/wasmstdlib", "-profile", "W32-WASI",
             "-llgo", LLGO, "-report", str(pathlib.Path(directory) / "w32-wasmer.json")],
            check=True, cwd=ROOT, env=env, timeout=600,
        )
        goroot = subprocess.check_output(["go", "env", "GOROOT"], env=env,
                                         text=True).strip()
        subprocess.run(
            ["go", "test", "./test/goroot", "-run", "^TestGoRootRunCases$",
             "-count=1", "-args", "-goroot", goroot, "-llgo", LLGO,
             "-wasm-profile", "W32-WASI", "-directive-mode", "ci",
             "-case", r"^(helloworld|env)\.go$", "-min-swap-free-mib=0"],
            check=True, cwd=ROOT, env=env, timeout=180,
        )


if __name__ == "__main__":
    main()

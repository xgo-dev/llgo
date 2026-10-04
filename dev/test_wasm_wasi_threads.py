#!/usr/bin/env python3

"""Exercise the default WASI pthread backend under Wasmer."""

import os
import pathlib
import shutil
import subprocess
import tempfile
import time


ROOT = pathlib.Path(__file__).resolve().parent.parent
LLGO = os.environ.get("LLGO", "llgo")
WASMER = os.environ.get("WASMER", "wasmer")


def wasmer_command(module, *args):
    return [WASMER, "run", "--v8" if os.name == "nt" else "--cranelift", "--enable-exceptions", "--enable-simd",
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
        run_output_cache_probe(env, directory)
        simd = pathlib.Path(directory) / "simd-threads-eh.wasm"
        subprocess.run([os.environ.get("WASM_TOOLS", "wasm-tools"), "parse",
                        str(ROOT / "internal/build/testdata/wasm-wasi-simd/threads.wat"),
                        "-o", str(simd)], check=True, timeout=30)
        subprocess.run(wasmer_command(simd), env=env, check=True, timeout=30)
        print("wasi SIMD/thread/standard-EH boundary ok", flush=True)
        run_probe(env, directory, "startup", "wasm-wasi-thread-startup", "nogc",
                  "wasi thread startup ok", 30)
        # Accept a host error (1) or the guest fatal status (2), never success.
        deadlock_exits = (1, 2)
        # LLVM lowers both Go defer/Goexit and C setjmp/longjmp through standard
        # Wasm EH. A caught exception must not terminate unrelated pthreads.
        for tags in ("nogc", ""):
            suffix = tags or "gc"
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
        run_arena_boundaries(env, directory)
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threads")],
                 "wasi threads ok")
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threaded-fs")],
                 "wasi threaded filesystem ok")
        run_llgo(env, ["test", "-target", "wasi", "-emulator",
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
        run_llgo(env, ["test", "-target", "wasi", "-emulator", "-run",
                       "^TestPoolAfterGC$", str(ROOT / "test/std/sync")], "PASS",
                 timeout=300)
        run_llgo(env, ["test", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/go/importer")], "PASS")
        run_llgo(env, ["test", "-target", "wasi", "-emulator", "-run",
                       "^(TestTBasicMethods|FuzzExample)$",
                       str(ROOT / "test/std/testing")], "PASS")
        run_llgo(env, ["test", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/weak")], "PASS")
        run_llgo(env, ["test", "-target", "wasi", "-emulator",
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
             "-case", r"^helloworld\.go$", "-min-swap-free-mib=0"],
            check=True, cwd=ROOT, env=env, timeout=180,
        )


if __name__ == "__main__":
    main()

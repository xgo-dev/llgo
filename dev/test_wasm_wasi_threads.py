#!/usr/bin/env python3

"""Exercise the experimental WASI pthread backend under WAMR."""

import os
import pathlib
import shutil
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parent.parent
LLGO = os.environ.get("LLGO", "llgo")
IWASM = os.environ.get("IWASM", "iwasm")


def run_probe(env, directory, name, fixture, tags, marker, timeout,
              max_threads=8, expected_exit=0):
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
    result = subprocess.run(
        [IWASM, f"--max-threads={max_threads}", "--stack-size=1048576",
         "--heap-size=0", "--dir=" + str(ROOT), "--dir=/tmp", str(module)],
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    print(result.stdout, end="")
    print(result.stderr, end="")
    if result.returncode != expected_exit or (
        marker not in result.stdout.splitlines()
        and marker not in result.stderr.splitlines()
    ):
        raise SystemExit(f"WAMR {name} probe failed with exit code {result.returncode}")


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


def main():
    iwasm = shutil.which(IWASM)
    if iwasm is None:
        raise SystemExit(f"WAMR runner not found: {IWASM}")

    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    env["LLGO_WASI_THREADS"] = "1"
    env["PATH"] = str(pathlib.Path(iwasm).resolve().parent) + os.pathsep + env["PATH"]
    with tempfile.TemporaryDirectory(prefix="llgo-wasi-threads-") as directory:
        run_probe(env, directory, "startup", "wasm-wasi-thread-startup", "nogc",
                  "wasi thread startup ok", 30, max_threads=32)
        run_probe(env, directory, "main-goexit", "wasm-wasi-main-goexit", "nogc",
                  "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=2)
        run_probe(env, directory, "main-goexit-gc", "wasm-wasi-main-goexit", "",
                  "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=2)
        run_probe(env, directory, "main-goexit-timer", "wasm-wasi-main-goexit-timer",
                  "nogc", "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=2)
        run_probe(env, directory, "main-goexit-timer-gc", "wasm-wasi-main-goexit-timer",
                  "", "fatal error: no goroutines (main called runtime.Goexit) - deadlock!",
                  30, expected_exit=2)
        run_probe(env, directory, "threads", "wasm-wasi-threads", "nogc",
                  "wasi threads ok", 30)
        run_probe(env, directory, "threaded-gc", "wasm-wasi-threaded-gc",
                  "", "wasi threaded gc ok", 180)
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threads")],
                 "wasi threads ok")
        run_llgo(env, ["run", "-target", "wasi", "-emulator",
                       str(ROOT / "internal/build/testdata/wasm-wasi-threaded-fs")],
                 "wasi threaded filesystem ok")
        run_llgo(env, ["test", "-target", "wasi", "-emulator",
                       str(ROOT / "test/std/errors")], "PASS")
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
             "-llgo", LLGO, "-report", str(pathlib.Path(directory) / "w32-wamr.json")],
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

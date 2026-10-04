#!/usr/bin/env python3

"""Verify real WASI LTO, SIMD and SjLj under Wasmer with a dev compiler."""

import os
import pathlib
import subprocess
import tempfile

from test_wasm_wasi_threads import wasmer_command


ROOT = pathlib.Path(__file__).resolve().parent.parent
LLGO = os.environ.get("LLGO", "llgo")


def build(env, directory, mode, fixture, tags="", test=False):
    module = directory / f"{fixture.name}-{mode}-{tags or 'gc'}.wasm"
    link_object = module.with_suffix(".lto.o")
    build_env = env.copy()
    # An actual LTO backend output catches flags that are accepted but ignored.
    flags = f"-Wl,--lto-obj-path={link_object}"
    if mode == "full":
        flags += " -Wl,--save-temps"
    build_env["LDFLAGS"] = (env.get("LDFLAGS", "") + " " + flags).strip()
    command = [LLGO, "test", "-c"] if test else [LLGO, "build"]
    command += ["-target", "wasi", "-O2", f"-lto={mode}"]
    if tags:
        command += ["-tags", tags]
    subprocess.run(command + ["-o", str(module), str(fixture)],
                   cwd=ROOT, env=build_env, check=True, timeout=300)
    if not link_object.is_file() or link_object.stat().st_size == 0:
        raise SystemExit(f"{module.name}: missing LTO backend object")
    if mode == "full":
        bitcode = pathlib.Path(str(module) + ".0.0.preopt.bc")
        if not bitcode.is_file() or bitcode.read_bytes()[:4] != b"BC\xc0\xde":
            raise SystemExit(f"{module.name}: missing merged LTO bitcode")
        if fixture.name == "globaldce_interface_matrix":
            ir = subprocess.check_output(
                [os.environ.get("LLVM_DIS", "llvm-dis"), str(bitcode), "-o", "-"],
                text=True, timeout=30)
            if "llvm.type.checked.load" not in ir:
                raise SystemExit("Full LTO did not exercise dev GlobalDCE metadata")
    return module


def run(env, module, markers, expected_exits=(0,), args=(), timeout=30):
    result = subprocess.run(
        wasmer_command(module, *args), env=env,
        capture_output=True, text=True, timeout=timeout)
    print(result.stdout, end="", flush=True)
    print(result.stderr, end="", flush=True)
    lines = result.stdout.splitlines() + result.stderr.splitlines()
    if result.returncode not in expected_exits or any(m not in lines for m in markers):
        raise SystemExit(f"{module.name} {args}: failed with exit {result.returncode}")


def main():
    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    env["RUST_LOG"] = "off"
    with tempfile.TemporaryDirectory(prefix="llgo-wasi-lto-") as temporary:
        directory = pathlib.Path(temporary)
        for mode in ("thin", "full"):
            print(f"Testing WASI {mode} LTO", flush=True)
            fixture = ROOT / "cl/_testlto/globaldce_interface_matrix"
            module = build(env, directory, mode, fixture)
            run(env, module, (fixture / "expect.txt").read_text().splitlines())
            for tags in ("nogc", ""):
                fixture = ROOT / "internal/build/testdata/wasm-wasi-goexit-defer"
                module = build(env, directory, mode, fixture, tags)
                # Exercise caught Go/C exceptions repeatedly across pthreads.
                for _ in range(3):
                    run(env, module, ["wasi worker defer ok"])
                for entry in ("init", "main"):
                    run(env, module, [f"wasi {entry} defer ok",
                                 "fatal error: no goroutines (main called runtime.Goexit) - deadlock!"],
                        expected_exits=(1, 2), args=(entry,))
                run(env, module, ["panic: wasi uncaught sentinel"],
                    expected_exits=(1, 2), args=("uncaught",))
            # Qualify the SIMD path introduced by the Wasmer base with actual
            # link-time optimization, including calls and exception fallbacks.
            simd_env = dict(env, GOEXPERIMENT="simd")
            module = build(simd_env, directory, mode, ROOT / "test/simd", test=True)
            run(env, module, ["PASS"], args=("-test.v", "-test.count=1", "-test.timeout=2m"),
                timeout=180)


if __name__ == "__main__":
    main()

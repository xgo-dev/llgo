#!/usr/bin/env python3

"""Verify real WASI LTO, SIMD and SjLj under Wasmer with a dev compiler."""

import os
import pathlib
import shlex
import subprocess
import tempfile

from test_wasm_wasi_threads import check_size_probe, wasmer_command


ROOT = pathlib.Path(__file__).resolve().parent.parent
LLGO = os.environ.get("LLGO", "llgo")


def build(env, directory, mode, fixture, tags="", test=False, extra_flags=(), opt="O2"):
    module = directory / f"{fixture.name}-{mode}-{tags or 'gc'}.wasm"
    link_object = module.with_suffix(".lto.o")
    build_env = env.copy()
    # An actual LTO backend output catches flags that are accepted but ignored.
    if mode == "thin":
        # --lto-obj-path forces an empty regular-LTO object even for ThinLTO.
        # LLVM 22 can mark that synthetic object memory64. A fresh cache proves
        # actual ThinLTO codegen without adding an object to the link.
        cache = pathlib.Path(tempfile.mkdtemp(prefix="thinlto-", dir=directory))
        flags = [f"-Wl,--thinlto-cache-dir={cache}"]
    else:
        flags = [f"-Wl,--lto-obj-path={link_object}", "-Wl,--save-temps"]
    build_env["LDFLAGS"] = (env.get("LDFLAGS", "") + " " + shlex.join(flags)).strip()
    command = [LLGO, "test", "-c"] if test else [LLGO, "build"]
    command += ["-target", "wasi", f"-{opt}", f"-lto={mode}"]
    command += extra_flags
    if tags:
        command += ["-tags", tags]
    subprocess.run(command + ["-o", str(module), str(fixture)],
                   cwd=ROOT, env=build_env, check=True, timeout=300)
    objects = list(cache.glob("llvmcache-*")) if mode == "thin" else [link_object]
    if not objects or any(not obj.is_file() or obj.read_bytes()[:8] != b"\0asm\1\0\0\0"
                          for obj in objects):
        raise SystemExit(f"{module.name}: missing or invalid LTO backend object")
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
        diagnostic = directory / "bootstrap-diagnostic.wasm"
        subprocess.run([LLGO, "build", "-target=wasi", "-o", str(diagnostic),
                        "./internal/clite/diagnostic/_test"],
                       cwd=ROOT / "runtime", env=env, check=True, timeout=300)
        run(env, diagnostic, ["0", "ffffffffffffffff", "(error=42)", "zero: (error=0)", "negative: (error=-1)",
                              "minimum: (error=-2147483648)", "maximum: (error=2147483647)"])
        for mode in ("thin", "full"):
            print(f"Testing WASI {mode} LTO", flush=True)
            fixture = ROOT / "cl/_testlto/globaldce_interface_matrix"
            module = build(env, directory, mode, fixture)
            run(env, module, (fixture / "expect.txt").read_text().splitlines())
            # Keep the original method-summary DCE covered alongside Full LTO;
            # the interface matrix above exercises LLVM GoGlobalDCE instead.
            module = build(env, directory, mode, ROOT / "internal/build/testdata/wasm-size",
                           extra_flags=("-deadcodedrop", "-globaldce=false"))
            check_size_probe(env, module)
            for fixture, marker in (
                ("wasm-type-fields", "wasm type fields passed"),
                ("wasm-type-fields-reflect", "wasm type fields reflection passed"),
                ("wasm-map-metadata", "wasm map metadata passed"),
                ("wasm-finalizer-direct", "wasm direct finalizers passed"),
                ("wasm-root-sharing", "wasm root sharing passed"),
            ):
                module = build(env, directory, mode, ROOT / "internal/build/testdata" / fixture,
                               extra_flags=("-deadcodedrop", "-globaldce=false"))
                run(env, module, [marker])
                if fixture == "wasm-root-sharing":
                    # Frame helpers are outlined only by size optimization.
                    sized = build(env, directory, mode, ROOT / "internal/build/testdata" / fixture,
                                  extra_flags=("-deadcodedrop", "-globaldce=false"), opt="Oz")
                    run(env, sized, [marker])
                if fixture == "wasm-finalizer-direct":
                    run(env, module, ["panic: fatal error: runtime.SetFinalizer: first argument is nil"],
                        expected_exits=(1, 2), args=("nil",))
            for fixture, marker in (
                ("interface-domain", "wasm interface domain passed"),
                ("reflect-method-value", "wasm reflect method values: 42 42 42 42 42"),
                ("reflect-method-alias", "wasm reflect method aliases: 42 42"),
                ("reflect-method-alias/named", "wasm named method aliases: 42 42"),
            ):
                module = build(env, directory, mode, ROOT / "test/wasm" / fixture,
                               extra_flags=("-deadcodedrop", "-globaldce=false"))
                run(env, module, [marker])
                if mode == "full":
                    module = build(env, directory, mode, ROOT / "test/wasm" / fixture)
                    run(env, module, [marker])
            module = build(env, directory, mode, ROOT / "internal/build/testdata/wasm-interface-panic",
                           extra_flags=("-deadcodedrop", "-globaldce=false"))
            for kind, message in (
                ("float32", "main.F32(1.25)"),
                ("float64", "main.F64(1.25)"),
                ("complex64", "main.C64 (1.25+2.5i)"),
                ("complex128", "main.C128 (1.25+2.5i)"),
            ):
                for args in ((kind,), (kind, "reflect")):
                    run(env, module, ["panic: " + message], expected_exits=(1, 2), args=args)
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

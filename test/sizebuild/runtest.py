#!/usr/bin/env python3
"""Build the size proposal's existing regression cases with a dev LLGo."""

import argparse
import os
import pathlib
import shutil
import subprocess
import tempfile


ROOT = pathlib.Path(__file__).resolve().parents[2]


def command(args, env, timeout=300):
    try:
        result = subprocess.run(args, cwd=ROOT, env=env, capture_output=True,
                                text=True, timeout=timeout)
    except subprocess.TimeoutExpired as error:
        # TimeoutExpired may hold bytes even with text=True, or None when the
        # process did not write to a stream before the deadline.
        output = "".join(part.decode(errors="replace") if isinstance(part, bytes)
                         else part or "" for part in (error.stdout, error.stderr))
        raise RuntimeError(f"{args!r} timed out after {timeout}s\n{output}") from error
    if result.returncode:
        raise RuntimeError(f"{args!r} failed ({result.returncode})\n"
                           f"{result.stdout}{result.stderr}")
    return result.stdout + result.stderr


def runtime_free_cache_probe(llgo, wasmer, directory, env):
    marker = "llgo: runtime startup omitted: proven allocation-free WASI program"
    # The same helper package contains a pure path and a panic path. Reuse its
    # normal cache while alternating whole-program runtime requirements.
    for fixture in ("wasm-runtime-free", "wasm-runtime-required", "wasm-runtime-free"):
        module = directory / f"{fixture}.wasm"
        log = command([str(llgo), "build", "-compiler-verbose", "-target=wasi", "-debug-artifact=none",
                       "-o", str(module), f"./internal/build/testdata/{fixture}"], env)
        minimal = fixture == "wasm-runtime-free"
        if (marker in log) != minimal:
            raise RuntimeError(f"{fixture}: incorrect runtime policy\n{log}")
        result = subprocess.run([wasmer, "run", "--enable-exceptions", "--enable-simd",
                                 "--stack-size=1048576", str(module)], cwd=ROOT, env=env,
                                capture_output=True, timeout=90)
        if minimal:
            expected = "zero\x00 世界 42 false -9223372036854775808 18446744073709551615\n".encode()
            if result.returncode or result.stdout or result.stderr != expected:
                raise RuntimeError(f"minimal program changed stderr/output: {result}")
        elif (result.returncode == 0 or b"runtime-free cache sentinel" not in result.stderr
              or b"helper.Fail" not in result.stderr or b"helper.go:6" not in result.stderr):
            raise RuntimeError(f"cached helper lost runtime or panic location: {result}")
    print("WASI runtime-free/full-runtime cache switching and stderr semantics passed", flush=True)


def controlled_environment(source):
    env = dict(source, LLGO_ROOT=str(ROOT), LLGO_BUILD_CACHE="1")
    # Ambient compiler/linker flags can invalidate the runtime-free proof.
    for name in ("LLGO_LTO_PLUGIN", "LLGO_PLAN9ASM_PKGS", "CCFLAGS", "CFLAGS", "LDFLAGS"):
        env.pop(name, None)
    return env


def wasi_full_lto_size_probe(llgo, wasmer, directory, env):
    cases = (
        ("wasm-fmt-literal-callers",
         "stdout captured before formatting\nliteral format caller parity passed"),
        ("wasm-wasi-thread-startup", "wasi thread startup ok"),
        ("wasm-fmt-map", "wasi default value formatting ok"),
    )
    for fixture, expected in cases:
        module = directory / f"{fixture}-full-size.wasm"
        log = command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(module), f"./internal/build/testdata/{fixture}"], env)
        if "llgo: enable closed-program WASI size pruning" not in log:
            raise RuntimeError(f"{fixture}: WASI Full LTO skipped Go reachability pruning\n{log}")
        if fixture == "wasm-fmt-map" and "llgo: specialize default value formatting" not in log:
            raise RuntimeError(f"{fixture}: closed WASI build skipped value formatting specialization\n{log}")
        actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                          "--stack-size=1048576", str(module)], env, timeout=90)
        if actual.strip() != expected:
            raise RuntimeError(f"{fixture}: expected {expected!r}, got {actual!r}")
    print("WASI Full LTO size pruning preserved formatter callers and worker startup", flush=True)


def reflect_method_size_probe(llgo, wasmer, directory, env):
    module = directory / "reflect-method-value.wasm"
    command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full",
             "-globaldce=false", "-debug-artifact=none",
             "-o", str(module), "./test/wasm/reflect-method-value"], env)
    actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                      "--stack-size=1048576", str(module)], env, timeout=90)
    if actual.strip() != "wasm reflect method values: 42 42 42 42 42":
        raise RuntimeError(f"reflect method values under WASI Full LTO: {actual!r}")


def full_lto_codec_probe(llgo, directory, env, target, runner, suffix, browser=False):
    binary_marker = "llgo: specialize fixed-size binary encoding"
    json_marker = "llgo: specialize statically typed JSON encoding"
    fixtures = [
        ("binary-fixed", "fixed binary codec ok", binary_marker, True),
        ("binary-mixed", "generic binary codec ok", binary_marker, False),
        ("json-static", "typed JSON encoding ok", json_marker, True),
        ("json-mixed", "generic JSON encoding ok", json_marker, False),
    ]
    if target != "wasi":
        fixtures.append(("wasm-fmt-map", "wasi default value formatting ok",
                         "llgo: specialize default value formatting", True))
    for fixture, expected, marker, specialized in fixtures:
        artifact = directory / f"{target}-{fixture}{suffix}"
        flags = ["-target", target] if target != "native" else []
        log = command([str(llgo), "build", *flags, "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), f"./internal/build/testdata/{fixture}"], env)
        if (marker in log) != specialized:
            raise RuntimeError(f"{target} {fixture}: incorrect specialization policy\n{log}")
        actual = command([*runner, str(artifact)], env, timeout=90)
        expected_output = expected
        if fixture == "json-static":
            expected_output = command(["go", "run", "./internal/build/testdata/json-static"], env).strip()
        if actual.strip() != expected_output:
            raise RuntimeError(f"{target} {fixture}: expected {expected_output!r}, got {actual!r}")
        if browser:
            node = env.get("NODE", "node")
            command([node, str(ROOT / "dev/test_wasm_browser.mjs"), str(artifact),
                     expected, "single"], env, timeout=90)
        print(f"{target} Full LTO {fixture} passed", flush=True)

def wasi_specialization_probe(llgo, directory, env, runner):
    for case, marker, want in (
        ("fmt-dynamic", "llgo: specialize default value formatting", True),
        ("fmt-method-fallback", "llgo: specialize default value formatting", False),
        ("json-static", "llgo: omit unused JSON package initialization", True),
        ("json-init-linked", "llgo: omit unused JSON package initialization", False),
    ):
        fixture = f"./internal/build/testdata/{case}"
        artifact = directory / (case + ".wasm")
        log = command([str(llgo), "build", "-target=wasi", "-Oz", "-lto=full", "-deadcodedrop",
                       "-globaldce=false", "-debug-artifact=none", "-compiler-verbose",
                       "-o", str(artifact), fixture], env)
        if (marker in log) != want:
            raise RuntimeError(f"{case}: unexpected size specialization policy\n{log}")
        variants = ([], ["%[2]*.[3]*[1]f", "%w", "%p"]) if case == "fmt-dynamic" else ([],)
        for args in variants:
            expected = command(["go", "run", fixture, *args], env)
            actual = command([*runner, str(artifact), *(["--", *args] if args else [])], env, timeout=90)
            if actual != expected:
                raise RuntimeError(f"{case}: expected {expected!r}, got {actual!r}")
        print(f"WASI {case} specialization policy and Go parity passed", flush=True)



def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=("wasi", "emscripten", "native", "esp32c3"), required=True)
    parser.add_argument("--browser", action="store_true", help="also run Emscripten products in Chrome")
    args = parser.parse_args()
    env = controlled_environment(os.environ)
    with tempfile.TemporaryDirectory(prefix="llgo-size-build-") as directory:
        directory = pathlib.Path(directory)
        suffix = ".exe" if os.name == "nt" else ""
        llgo = directory / f"llgo{suffix}"
        command(["go", "build", "-tags=dev", "-o", str(llgo), "./cmd/llgo"], env)
        if args.target == "emscripten":
            env["LLGO_WASM_WORKERS"] = "1"
            node = env.get("NODE", "node")
            for target in ("emscripten", "emscripten-memory64"):
                runner = [node, str(ROOT / "targets" / f"{target}-runner.mjs")]
                full_lto_codec_probe(llgo, directory, env, target, runner, ".mjs", args.browser)
            return
        if args.target == "native":
            full_lto_codec_probe(llgo, directory, env, "native", [], "")
            return
        if args.target == "esp32c3":
            # This entry never builds the Go runtime. Its preloaded runtime
            # packages must not become DCE inputs with missing metadata.
            for cold in (True, False):
                output = directory / "cprintf.elf"
                flags = ["-a"] if cold else []
                command([str(llgo), "build", *flags, "-target=esp32c3",
                         "-deadcodedrop", "-o", str(output),
                         "./benchmark/binary_size/cprintf"], env)
                if output.read_bytes()[:4] != b"\x7fELF":
                    raise RuntimeError("C-only DCE did not produce an ELF image")
            print("ESP32-C3 C-only DCE cold/warm builds passed (build-only)")
            return

        wasmer = env.get("WASMER", "wasmer")
        if not shutil.which(wasmer):
            raise RuntimeError(f"Wasmer runner not found: {wasmer}")
        env["RUST_LOG"] = "off"
        runtime_free_cache_probe(llgo, wasmer, directory, env)
        wasi_full_lto_size_probe(llgo, wasmer, directory, env)
        reflect_method_size_probe(llgo, wasmer, directory, env)
        wasi_runner = [wasmer, "run", "--enable-exceptions", "--enable-simd",
                       "--stack-size=1048576"]
        full_lto_codec_probe(llgo, directory, env, "wasi", wasi_runner, ".wasm")
        wasi_specialization_probe(llgo, directory, env, wasi_runner)
        cases = (("off", "cprintf"), ("off", "println"),
                 ("thin", "cprintf"), ("full", "cprintf"),
                 ("full", "println"), ("thin", "println"),
                 ("full", "fmtprintf"), ("full", "goexit-defer"),
                 ("thin", "goexit-defer"), ("full", "threaded-gc"))
        probes = {
            "goexit-defer": ("./internal/build/testdata/wasm-wasi-goexit-defer",
                             "wasi worker defer ok"),
            "threaded-gc": ("./internal/build/testdata/wasm-wasi-threaded-gc",
                            "wasi threaded gc ok"),
        }
        for mode, sample in cases:
            fixture, expected = probes.get(
                sample, (f"./benchmark/binary_size/{sample}", "Hello, world"))
            # Reuse the LTO packages for the runtime probes; the smaller
            # samples above independently check forced and cached builds.
            builds = (False,) if sample in probes else (True, False)
            for cold in builds:
                output = directory / f"{sample}-{mode}.wasm"
                flags = ["-a"] if cold else []
                if mode != "off":
                    flags.append(f"-lto={mode}")
                command([str(llgo), "build", *flags, "-target=wasi",
                         "-o", str(output), fixture], env)
                actual = command([wasmer, "run", "--enable-exceptions", "--enable-simd",
                                  "--stack-size=1048576", "--volume=" + str(ROOT),
                                  "--volume=/tmp", str(output)], env, timeout=90)
                if actual.strip() != expected:
                    raise RuntimeError(f"{sample}/{mode}: expected {expected!r}, got {actual!r}")
            print(f"WASI {sample}/{mode} builds and Wasmer runs passed", flush=True)


if __name__ == "__main__":
    main()

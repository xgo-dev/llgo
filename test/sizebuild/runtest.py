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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=("wasi", "esp32c3"), required=True)
    args = parser.parse_args()
    env = controlled_environment(os.environ)
    with tempfile.TemporaryDirectory(prefix="llgo-size-build-") as directory:
        directory = pathlib.Path(directory)
        suffix = ".exe" if os.name == "nt" else ""
        llgo = directory / f"llgo{suffix}"
        command(["go", "build", "-tags=dev", "-o", str(llgo), "./cmd/llgo"], env)
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

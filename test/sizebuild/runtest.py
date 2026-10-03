#!/usr/bin/env python3
"""Build the size proposal's existing regression cases with a dev LLGo."""

import argparse
import os
import pathlib
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--target", choices=("wasi", "esp32c3"), required=True)
    args = parser.parse_args()
    env = dict(os.environ, LLGO_ROOT=str(ROOT), LLGO_BUILD_CACHE="1")
    env.pop("LLGO_LTO_PLUGIN", None)
    env.pop("LLGO_PLAN9ASM_PKGS", None)
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

        iwasm = env.get("IWASM", "iwasm")
        cases = (("full", "println"), ("thin", "println"),
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
                command([str(llgo), "build", *flags, "-target=wasi",
                         f"-lto={mode}", "-o", str(output), fixture], env)
                actual = command([iwasm, "--max-threads=128", "--stack-size=1048576",
                                  "--heap-size=0", str(output)], env, timeout=90)
                if actual.strip() != expected:
                    raise RuntimeError(f"{sample}/{mode}: expected {expected!r}, got {actual!r}")
            print(f"WASI {sample}/{mode} builds and WAMR runs passed", flush=True)


if __name__ == "__main__":
    main()

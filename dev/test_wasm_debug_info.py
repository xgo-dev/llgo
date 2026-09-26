#!/usr/bin/env python3

"""Check final WebAssembly DWARF after linking and Binaryen rewrites."""

import argparse
import os
from pathlib import Path
import re
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
FIXTURE = ROOT / "internal/build/testdata/wasm-debug"
PROFILES = {"j32": "emscripten", "j64": "emscripten-memory64", "w32": "wasi"}
SOURCE_LINES = {"main.go": 12, "probe.cpp": 4}


def run(command, *, env=None, timeout=180):
    result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise RuntimeError(
            f"{' '.join(map(str, command))} exited {result.returncode}:\n"
            f"{result.stdout}\n{result.stderr}"
        )
    return result.stdout + result.stderr


def check_source_lines(module, debug_line, addr2line):
    # DWARF line tables use code-section offsets. The Wasm symbol table uses
    # different offsets, so resolve actual line-table rows, not symbol values.
    tables = re.split(r"(?=^debug_line\[)", debug_line, flags=re.MULTILINE)
    for filename, line in SOURCE_LINES.items():
        matches = []
        line_pattern = re.compile(
            rf"^0x([0-9a-fA-F]+)\s+{line}\s+\d+\s+1\s",
            re.MULTILINE,
        )
        for table in tables:
            if f'name: "{filename}"' not in table:
                continue
            for match in line_pattern.finditer(table):
                matches.append(match.group(1))
        for address in matches:
            location = run([addr2line, "-e", str(module), "-f", f"0x{address}"])
            if re.search(rf"{re.escape(filename)}:{line}\b", location):
                break
        else:
            raise RuntimeError(
                f"{module}: {filename}:{line} is not resolvable with llvm-addr2line "
                f"({len(matches)} line-table rows)"
            )


def check_module(module, *, dwarfdump, addr2line):
    verification = run([dwarfdump, "--verify", str(module)])
    if "No errors." not in verification or re.search(r"\b(?:warning|error):", verification, re.I):
        raise RuntimeError(f"{module}: invalid final DWARF:\n{verification}")

    info = run([dwarfdump, "--debug-info", str(module)])
    for name in ("main.goProbe", "cppResult", "llgo_debug_cpp_probe", "cpp_local"):
        if not re.search(rf'DW_AT_name\s+\("{re.escape(name)}"\)', info):
            raise RuntimeError(f"{module}: missing Go/C++ DWARF entry {name}")
    line_table = run([dwarfdump, "--debug-line", str(module)])
    check_source_lines(module, line_table, addr2line)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", action="append", choices=PROFILES, dest="profiles")
    parser.add_argument("--opt", action="append", choices=("0", "2"), dest="opts")
    options = parser.parse_args()
    profiles = options.profiles or list(PROFILES)
    opts = options.opts or ["0", "2"]

    llgo = os.environ.get("LLGO", "llgo")
    dwarfdump = os.environ.get("LLVM_DWARFDUMP", "llvm-dwarfdump")
    addr2line = os.environ.get("LLVM_ADDR2LINE", "llvm-addr2line")
    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    with tempfile.TemporaryDirectory(prefix="llgo-wasm-debug-") as directory:
        for profile in profiles:
            for opt in opts:
                stem = Path(directory) / f"{profile}-O{opt}"
                module = stem.with_suffix(".wasm")
                output = module if profile == "w32" else stem.with_suffix(".mjs")
                run(
                    [llgo, "build", "-target", PROFILES[profile], f"-O{opt}",
                     "-ldflags=-w=false", "-o", str(output), str(FIXTURE)],
                    env=env,
                )
                check_module(module, dwarfdump=dwarfdump, addr2line=addr2line)
                if profile == "w32":
                    command = [os.environ.get("WASMTIME", "wasmtime"), "run", "-W", "exceptions=y", str(module)]
                else:
                    runner = "emscripten-memory64-runner.mjs" if profile == "j64" else "emscripten-runner.mjs"
                    command = [os.environ.get("NODE", "node"), str(ROOT / "targets" / runner), str(output)]
                result = run(command, timeout=60)
                if "wasm debug ok" not in result.splitlines():
                    raise RuntimeError(f"{profile} O{opt}: fixture did not complete:\n{result}")
                print(f"{profile} O{opt}: final DWARF, Go/C++ source lines and runtime passed", flush=True)


if __name__ == "__main__":
    main()

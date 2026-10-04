#!/usr/bin/env python3

"""Check final WebAssembly DWARF after linking and Binaryen rewrites."""

import argparse
import itertools
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
FIXTURE = ROOT / "internal/build/testdata/wasm-debug"
PROFILES = {"j32": "emscripten", "j64": "emscripten-memory64", "w32": "wasi"}
SOURCE_LINES = {"main.go": 12, "probe.cpp": 4}


def wasm_sections(module):
    raw = module.read_bytes()
    if raw[:8] != b"\0asm\1\0\0\0":
        raise RuntimeError(f"{module}: invalid Wasm header")
    def uleb(offset):
        value = 0
        for shift in range(0, 35, 7):
            byte = raw[offset]
            offset += 1
            value |= (byte & 127) << shift
            if not byte & 128:
                return value, offset
        raise RuntimeError(f"{module}: invalid section length")
    sections, offset = [], 8
    while offset < len(raw):
        start = offset
        section_id = raw[offset]
        size, payload = uleb(offset + 1)
        offset = payload + size
        if offset > len(raw):
            raise RuntimeError(f"{module}: truncated section")
        name = None
        if section_id == 0:
            length, payload = uleb(payload)
            name = raw[payload:payload + length].decode()
            payload += length
        sections.append((section_id, name, raw[payload:offset], raw[start:offset]))
    return sections


def check_external_pair(module, sidecar):
    main, debug = wasm_sections(module), wasm_sections(sidecar)
    def custom(sections, name):
        values = [content for _, key, content, _ in sections if key == name]
        if len(values) != 1:
            raise RuntimeError(f"{module}: expected one {name} custom section")
        return values[0]
    for name in ("build_id", "llgo.debugger"):
        if custom(main, name) != custom(debug, name):
            raise RuntimeError(f"{module}: mismatched {name} identity")
    if any(name and name.startswith(".debug_") for _, name, _, _ in main):
        raise RuntimeError(f"{module}: external executable still contains DWARF")
    if sidecar.name.encode() not in custom(main, "external_debug_info"):
        raise RuntimeError(f"{module}: external_debug_info points to a different sidecar")
    if [raw for kind, _, _, raw in main if kind] != [raw for kind, _, _, raw in debug if kind]:
        raise RuntimeError(f"{module}: externalization changed standard module sections")


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
        for table in tables:
            # DWARF v4/v5 and linked units do not assign a fixed file index.
            files = re.findall(r'file_names\[\s*(\d+)\]:\s*name: "([^"\n]+)"', table)
            indexes = {int(index) for index, name in files if Path(name).name == filename}
            for match in re.finditer(r'^0x([0-9a-fA-F]+)\s+(\d+)\s+\d+\s+(\d+)\s', table, re.MULTILINE):
                if int(match.group(2)) == line and int(match.group(3)) in indexes:
                    matches.append(match.group(1))
        for address in matches:
            location = run([addr2line, "-e", str(module), "-f", f"0x{address}"])
            if re.search(rf"{re.escape(filename)}:{line}\b", location):
                break
        else:
            raise RuntimeError(
                f"{module}: {filename}:{line} is not resolvable with llvm-addr2line "
                f"({len(matches)} line-table rows)\n"
                + "\n".join(table for table in tables if filename in table)
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


def check_artifact_report(output, profile, artifact, build_log):
    module = output.with_suffix(".wasm")
    role = "debug+deployment" if artifact == "embedded" else "deployment"
    expected = {module: (role, "wasm")}
    if output != module:
        expected[output] = ("deployment", "html" if output.suffix == ".html" else "javascript")
    if output.suffix == ".html":
        expected[output.with_suffix(".js")] = ("deployment", "javascript")
    if profile != "w32":
        expected[output.parent / "wasm_fs.js"] = ("deployment", "javascript")
    if artifact == "external":
        expected[output.with_suffix(".debug.wasm")] = ("debug", "wasm-dwarf")
    entries = re.findall(
        r'^llgo: artifact role=(\S+) format=(\S+) size=(\d+) path=(".*")$',
        build_log, re.MULTILINE,
    )
    reported = {Path(json.loads(path)): (role, format_, int(size))
                for role, format_, size, path in entries}
    if len(entries) != len(reported) or reported != {
        path: (*kind, path.stat().st_size) for path, kind in expected.items()
    }:
        raise RuntimeError(f"{output}: incomplete or inaccurate artifact report:\n{build_log}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--profile", action="append", choices=PROFILES, dest="profiles")
    parser.add_argument("--opt", action="append", choices=("0", "2"), dest="opts")
    parser.add_argument("--artifact", action="append", choices=("embedded", "external"), dest="artifacts")
    options = parser.parse_args()
    profiles = options.profiles or list(PROFILES)
    opts = options.opts or ["0", "2"]
    artifacts = options.artifacts or ["embedded"]

    llgo = os.environ.get("LLGO", "llgo")
    dwarfdump = os.environ.get("LLVM_DWARFDUMP", "llvm-dwarfdump")
    addr2line = os.environ.get("LLVM_ADDR2LINE", "llvm-addr2line")
    env = os.environ.copy()
    env["LLGO_ROOT"] = str(ROOT)
    env["RUST_LOG"] = "off"
    with tempfile.TemporaryDirectory(prefix="llgo-wasm-debug-") as directory:
        for profile, opt, artifact in itertools.product(profiles, opts, artifacts):
            stem = Path(directory) / f"{profile}-O{opt}-{artifact}"
            module = stem.with_suffix(".wasm")
            output = module if profile == "w32" else stem.with_suffix(".mjs")
            build_log = run(
                [llgo, "build", "-target", PROFILES[profile], f"-O{opt}",
                 f"-debug-artifact={artifact}", "-o", str(output), str(FIXTURE)],
                env=env,
            )
            check_artifact_report(output, profile, artifact, build_log)
            debug_module = module
            if artifact == "external":
                debug_module = stem.with_suffix(".debug.wasm")
                check_external_pair(module, debug_module)
            check_module(debug_module, dwarfdump=dwarfdump, addr2line=addr2line)
            if profile == "w32":
                command = [os.environ.get("WASMER", "wasmer"), "run", "--v8" if os.name == "nt" else "--cranelift", "--enable-exceptions", "--enable-simd",
                           "--stack-size=1048576", str(module)]
            else:
                runner = "emscripten-memory64-runner.mjs" if profile == "j64" else "emscripten-runner.mjs"
                command = [os.environ.get("NODE", "node"), str(ROOT / "targets" / runner), str(output)]
            result = run(command, env=env, timeout=60)
            if "wasm debug ok" not in result.splitlines():
                raise RuntimeError(f"{profile} O{opt}: fixture did not complete:\n{result}")
            print(f"{profile} O{opt} {artifact}: final DWARF, Go/C++ source lines and runtime passed", flush=True)

        # HTML has an additional owned .js loader; the regular matrix exercises
        # .mjs. Verify this second browser packaging contract once per run.
        if "j32" in profiles:
            output = Path(directory) / "html-report.html"
            build_log = run(
                [llgo, "build", "-target", PROFILES["j32"], f"-O{opts[0]}",
                 f"-debug-artifact={artifacts[0]}", "-o", str(output), str(FIXTURE)],
                env=env,
            )
            check_artifact_report(output, "j32", artifacts[0], build_log)
            print("j32 HTML: complete deployment artifacts and byte sizes passed", flush=True)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Controlled native macOS size probe; run separately for each compiler revision.

Build timing includes LLGo's PCLN processing, but excludes the common final
strip/sign step. -a forces package rebuilds; OS/tool caches may still be warm.
The output directory must be unique per compiler. Results are saved after each
run, including failed builds. No runtime-performance claim is made.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import resource
import shlex
import shutil
import subprocess
import time


CASES = {
    "reflectmetadata": "Decode 1 1\n",
    "reflectcall": '{"answer":42}\n',
    "fmtprintf": "Hello, world\n",
    "texttemplate": "Names: Gamora, Groot, Nebula, Rocket, Star-Lord\n",
}


def capture(args, **kwargs):
    return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT, **kwargs)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--compiler", required=True, type=Path)
    parser.add_argument("--plugin", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--linker", type=Path, help="explicit ld64.lld (plugin must match its LLVM version)")
    parser.add_argument("--cases", nargs="+", choices=CASES, default=list(CASES))
    parser.add_argument("--modes", nargs="+", choices=["drop", "thin", "full", "fullp"],
                        default=["drop", "thin", "full", "fullp"])
    parser.add_argument("--repeat", type=int, default=3)
    args = parser.parse_args()
    if platform.system() != "Darwin" or args.repeat < 1:
        parser.error("requires macOS and a positive repetition count")
    root = Path(__file__).resolve().parent.parent
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    if (output / "results.json").exists():
        parser.error("output already contains results; use a new directory")
    compiler, plugin = args.compiler.resolve(), args.plugin.resolve()
    linkflags = "-Wl,-no_exported_symbols"
    if args.linker:
        linkflags += " " + shlex.quote("--ld-path=" + str(args.linker.absolute()))
    env = dict(os.environ, LLGO_ROOT=str(root), LDFLAGS=linkflags)
    flags = {"drop": ["-deadcodedrop"], "thin": ["-lto=thin", "-deadcodedrop"],
             "full": ["-lto=full"],
             "fullp": ["-lto=full", "-lto-pass-plugin=" + str(plugin)]}
    results = {"host": platform.platform(), "commit": capture(["git", "rev-parse", "HEAD"], cwd=root).strip(),
               "compiler": str(compiler), "compiler_sha256": hashlib.sha256(compiler.read_bytes()).hexdigest(),
               "plugin_sha256": hashlib.sha256(plugin.read_bytes()).hexdigest(),
               "go": capture(["go", "version"]).strip(),
               "llvm": capture(["llvm-config", "--version"]).strip(),
               "linker": capture([str(args.linker.absolute()) if args.linker else "ld64.lld", "--version"]).strip(),
               "ldflags": linkflags, "runs": []}
    (output / "worktree.diff").write_text(capture(["git", "diff"], cwd=root))
    (output / "worktree.status").write_text(capture(["git", "status", "--short"], cwd=root))
    for repeat in range(args.repeat):
        # Rotate mode order to avoid assigning every first run to one variant.
        modes = args.modes[repeat % len(args.modes):] + args.modes[:repeat % len(args.modes)]
        for case in args.cases:
            for mode in modes:
                name = f"{case}-{mode}-{repeat}"
                exe = output / "probe"
                command = [str(compiler), "build", "-a", "-Os", "-debug-artifact=none",
                           *flags[mode], "-o", str(exe), "./benchmark/binary_size/" + case]
                row = {"case": case, "mode": mode, "repeat": repeat, "command": command}
                cpu_before = resource.getrusage(resource.RUSAGE_CHILDREN)
                start = time.monotonic()
                with (output / (name + ".log")).open("w") as log:
                    build = subprocess.run(command, cwd=root, env=env, stdout=log, stderr=subprocess.STDOUT)
                row.update(seconds=time.monotonic() - start, returncode=build.returncode)
                cpu_after = resource.getrusage(resource.RUSAGE_CHILDREN)
                row["cpu_seconds"] = cpu_after.ru_utime + cpu_after.ru_stime - cpu_before.ru_utime - cpu_before.ru_stime
                # Save the build record even if a postprocessing tool fails.
                results["runs"].append(row)
                (output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
                if build.returncode == 0:
                    row["raw_bytes"] = exe.stat().st_size
                    symbols = capture(["nm", "--defined-only", str(exe)])
                    (output / (name + ".nm")).write_text(symbols)
                    row["decode_symbols"] = [line for line in symbols.splitlines() if "main.payload.Decode" in line]
                    sections = capture(["otool", "-l", str(exe)])
                    (output / (name + ".sections")).write_text(sections)
                    lines = sections.splitlines()
                    row["sections"] = {}
                    for i, line in enumerate(lines):
                        if line.strip().startswith("sectname "):
                            section = line.split()[1]
                            segment = lines[i + 1].split()[1]
                            size = int(lines[i + 3].split()[1], 16)
                            row["sections"][segment + "," + section] = size
                    capture(["strip", "-x", str(exe)])
                    capture(["codesign", "-f", "-s", "-", str(exe)])
                    capture(["codesign", "--verify", str(exe)])
                    row["bytes"] = exe.stat().st_size
                    run = subprocess.run([str(exe)], text=True, capture_output=True, timeout=30)
                    row.update(run_returncode=run.returncode, stdout=run.stdout, stderr=run.stderr)
                    row["output_ok"] = run.returncode == 0 and (CASES[case] is None or run.stdout + run.stderr == CASES[case])
                    shutil.copy2(exe, output / name)
                (output / "results.json").write_text(json.dumps(results, indent=2) + "\n")
                print(name, json.dumps({k: row[k] for k in ("seconds", "bytes", "output_ok", "returncode") if k in row}), flush=True)
                if build.returncode or not row.get("output_ok"):
                    raise SystemExit("failed build or output check: " + name)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Export Drop's LiveSlots from two compiler checkouts without editing either.

Build temporary diagnostic compilers using Go overlays. These binaries are for
analysis comparison only; use uninstrumented compilers for performance results.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shlex
import subprocess


def diagnostic(root, destination, name):
    source = root / "internal/deadcode/analyze.go"
    text = source.read_text()
    anchor = "\treturn Analysis{LiveSlots: out, info: info, visible: pass.usedInIface, typeLookup: typeLookup}"
    if text.count(anchor) != 1:
        raise RuntimeError("AnalyzeProgram changed; update the diagnostic overlay")
    text = text.replace("import (\n", 'import (\n\t"encoding/json"\n\t"os"\n', 1)
    injection = '''
    if path := os.Getenv("LLGO_EXPERIMENT_LIVE_SLOTS"); path != "" {
        payload, err := json.MarshalIndent(out, "", "  ")
        if err != nil { panic(err) }
        if err := os.WriteFile(path, payload, 0644); err != nil { panic(err) }
    }
'''
    rewritten = destination / (name + "-analyze.go")
    rewritten.write_text(text.replace(anchor, injection + anchor))
    overlay = destination / (name + "-overlay.json")
    overlay.write_text(json.dumps({"Replace": {str(source): str(rewritten)}}))
    compiler = destination / (name + "-compiler")
    subprocess.run(["go", "build", "-tags", "dev", "-overlay=" + str(overlay),
                    "-o", str(compiler), "./cmd/llgo"], cwd=root, check=True)
    return compiler


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--strong-root", type=Path, required=True)
    p.add_argument("--thin-root", type=Path, required=True)
    p.add_argument("--gorm-module", type=Path, required=True)
    p.add_argument("--linker", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    args = p.parse_args()
    destination = args.output.resolve()
    destination.mkdir(parents=True, exist_ok=True)
    if (destination / "comparison.json").exists():
        p.error("use a new output directory")
    roots = {mode: getattr(args, mode + "_root").resolve() for mode in ("strong", "thin")}
    compilers = {mode: diagnostic(root, destination, mode) for mode, root in roots.items()}
    flags = "-Wl,-no_exported_symbols " + shlex.quote("--ld-path=" + str(args.linker.absolute()))
    comparisons = []
    for case in ("reflectcallbacks", "gormschema"):
        slots = {}
        for mode, compiler in compilers.items():
            stem = destination / (case + "-" + mode)
            snapshot = stem.with_suffix(".slots.json")
            env = dict(os.environ, LLGO_ROOT=str(roots["strong"]), LDFLAGS=flags,
                       LLGO_EXPERIMENT_LIVE_SLOTS=str(snapshot))
            gorm = case == "gormschema"
            command = [str(compiler), *(["test", "-c"] if gorm else ["build"]),
                       "-a", "-Os", "-p=10", "-deadcodedrop", "-debug-artifact=none", "-compiler-verbose"]
            if mode == "thin":
                command += ["-lto=thin"]
            command += ["-o", str(destination / "probe"), "gorm.io/gorm/schema" if gorm else "./benchmark/binary_size/reflectcallbacks"]
            with stem.with_suffix(".log").open("w") as log:
                subprocess.run(command, cwd=args.gorm_module if gorm else roots["strong"], env=env,
                               stdout=log, stderr=subprocess.STDOUT, check=True)
            slots[mode] = json.loads(snapshot.read_text())
        a, b = slots["strong"], slots["thin"]
        canonical = json.dumps(a, sort_keys=True, separators=(",", ":")).encode()
        comparisons.append({"case": case, "equal": a == b, "types": len(a),
                            "slots": sum(map(len, a.values())), "sha256": hashlib.sha256(canonical).hexdigest(),
                            "different_types": sorted(k for k in a.keys() | b.keys() if a.get(k) != b.get(k))})
        (destination / "comparison.json").write_text(json.dumps(comparisons, indent=2) + "\n")
        print(json.dumps(comparisons[-1]), flush=True)


if __name__ == "__main__":
    main()

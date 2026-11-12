#!/usr/bin/env python3
"""Compare Strong and Thin Drop LiveSlots without editing compiler sources.

Use --root to build one diagnostic compiler and select both backends with its
LTO flags. The historical --strong-root/--thin-root pair compares two checkouts.
Go overlays add diagnostics; use uninstrumented compilers for performance work.
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
    p.add_argument("--root", type=Path, help="one checkout providing both backends")
    p.add_argument("--strong-root", type=Path, help="historical Strong checkout (requires --thin-root)")
    p.add_argument("--thin-root", type=Path, help="historical Thin checkout (requires --strong-root)")
    p.add_argument("--gorm-module", type=Path, required=True)
    p.add_argument("--linker", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    p.add_argument("--jobs", type=int, default=3)
    args = p.parse_args()
    if args.root is not None:
        if args.strong_root is not None or args.thin_root is not None:
            p.error("--root cannot be combined with --strong-root or --thin-root")
        roots = dict.fromkeys(("strong", "thin"), args.root.resolve())
    else:
        if args.strong_root is None or args.thin_root is None:
            p.error("provide --root, or both --strong-root and --thin-root")
        roots = {mode: getattr(args, mode + "_root").resolve() for mode in ("strong", "thin")}
    if args.jobs < 1:
        p.error("--jobs must be positive")
    destination = args.output.resolve()
    if destination.exists() and any(destination.iterdir()):
        p.error("use a new or empty output directory")
    destination.mkdir(parents=True, exist_ok=True)
    if args.root is not None:
        compiler = diagnostic(roots["strong"], destination, "unified")
        compilers = dict.fromkeys(("strong", "thin"), compiler)
    else:
        compilers = {mode: diagnostic(root, destination, mode) for mode, root in roots.items()}
    flags = "-Wl,-no_exported_symbols " + shlex.quote("--ld-path=" + str(args.linker.absolute()))
    record = {"comparison": "one compiler, two backends" if args.root is not None else "two historical checkouts",
              "source_root": str(roots["strong"]), "jobs": args.jobs,
              "compilers": {mode: {"root": str(roots[mode]), "path": str(compiler),
                                   "sha256": hashlib.sha256(compiler.read_bytes()).hexdigest()}
                            for mode, compiler in compilers.items()}}
    (destination / "audit.json").write_text(json.dumps(record, indent=2) + "\n")
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
                       "-a", "-Os", "-p=" + str(args.jobs), "-deadcodedrop", "-debug-artifact=none", "-compiler-verbose"]
            if mode == "thin":
                command += ["-lto=thin"]
            command += ["-o", str(destination / "probe"), "gorm.io/gorm/schema" if gorm else "./benchmark/binary_size/reflectcallbacks"]
            with stem.with_suffix(".log").open("w") as log:
                subprocess.run(command, cwd=args.gorm_module.resolve() if gorm else roots["strong"], env=env,
                               stdout=log, stderr=subprocess.STDOUT, check=True)
            slots[mode] = json.loads(snapshot.read_text())
        a, b = slots["strong"], slots["thin"]
        metrics = {}
        for mode, mapping in slots.items():
            canonical = json.dumps(mapping, sort_keys=True, separators=(",", ":")).encode()
            metrics[mode] = {"types": len(mapping), "slots": sum(map(len, mapping.values())),
                             "sha256": hashlib.sha256(canonical).hexdigest()}
        comparisons.append({"case": case, "equal": a == b, "types": len(a),
                            "slots": sum(map(len, a.values())), "sha256": metrics["strong"]["sha256"],
                            "backends": metrics,
                            "different_types": sorted(k for k in a.keys() | b.keys() if a.get(k) != b.get(k))})
        (destination / "comparison.json").write_text(json.dumps(comparisons, indent=2) + "\n")
        print(json.dumps(comparisons[-1]), flush=True)
    if not all(row["equal"] for row in comparisons):
        raise SystemExit("Strong and Thin LiveSlots differ; see comparison.json")


if __name__ == "__main__":
    main()

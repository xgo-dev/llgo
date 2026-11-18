#!/usr/bin/env python3
"""Exercise source exports, SSA omission and real Wasm execution on J32/J64/W32."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[3]
PROFILES = {"j32": ("emscripten", "emscripten-runner.mjs"),
            "j64": ("emscripten-memory64", "emscripten-memory64-runner.mjs"),
            "w32": ("wasi", None)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--llgo", default=os.environ.get("LLGO", "llgo"))
    parser.add_argument("--profile", action="append", choices=PROFILES)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    env = dict(os.environ, LLGO_ROOT=str(ROOT), LLGO_ESCAPE_CACHE="1")
    with tempfile.TemporaryDirectory(prefix="llgo-frontend-cache-") as temporary:
        output = args.output.resolve() if args.output else Path(temporary)
        output.mkdir(parents=True, exist_ok=True)
        fixture = output / "source"
        shutil.copytree(ROOT / "internal/build/testdata/frontendcache", fixture)
        report = []
        for profile in args.profile or PROFILES:
            target, runner = PROFILES[profile]
            original = (fixture / "main.go").read_text()
            phases = [("seed", 1), ("reuse", 8)]
            if profile == "w32":
                phases += [("context-change", 8), ("context-reuse", 8)]
            for phase, workers in phases:
                if phase == "context-change":
                    changed = original.replace(
                        "if f.Call([]reflect.Value{reflect.ValueOf(12)})[0].Int() != 15 {",
                        "if f.Type().NumIn() != 1 {")
                    assert changed != original
                    (fixture / "main.go").write_text(changed)
                name = profile + "-" + phase
                artifact = output / (name + (".wasm" if runner is None else ".mjs"))
                trace = output / (name + ".json")
                command = [args.llgo, "build", "-target", target, "-p=" + str(workers),
                           "-debug-trace", str(trace), "-o", str(artifact), "."]
                built = subprocess.run(command, cwd=fixture, env=env, capture_output=True,
                                       text=True, timeout=600)
                (output / (name + ".log")).write_text(built.stdout + built.stderr)
                if built.returncode:
                    raise RuntimeError(name + " build failed: " + built.stdout + built.stderr)
                events = json.loads(trace.read_text())
                facts = next(e["args"] for e in events if e["name"] == "certify frontend exports")
                ssa = [e.get("args", {}).get("package_id") for e in events
                       if e.get("args", {}).get("stage") == "ssa"]
                if phase.startswith("context-"):
                    assert not facts["wasm_reflect_bridges"], "stale whole-program reflection decision"
                if phase in ("reuse", "context-reuse"):
                    assert facts["skipped_ssa"] > 0, name + " reconstructed all dependency SSA"
                    assert facts["type_exports"] > 0, name + " did not import checked types"
                    assert "example.com/frontprobe/dep" not in ssa
                    assert "example.com/frontprobe/inline" in ssa, "inline body was omitted"
                if runner is None:
                    run = [os.environ.get("WASMER", "wasmer"), "run", "--enable-exceptions",
                           "--enable-simd", "--stack-size=1048576", "--volume=/tmp", str(artifact)]
                else:
                    run = [os.environ.get("NODE", "node"), str(ROOT / "targets" / runner), str(artifact)]
                executed = subprocess.run(run, cwd=fixture, env=env, capture_output=True,
                                          text=True, timeout=120)
                (output / (name + "-run.log")).write_text(executed.stdout + executed.stderr)
                if executed.returncode or "frontend-ok" not in executed.stdout + executed.stderr:
                    raise RuntimeError(name + " runtime failed: " + executed.stdout + executed.stderr)
                item = dict(profile=profile, phase=phase, workers=workers, ssa_packages=len(ssa), **facts)
                report.append(item)
                print(json.dumps(item), flush=True)
            (fixture / "main.go").write_text(original)
        (output / "report.json").write_text(json.dumps(report, indent=2) + "\n")


if __name__ == "__main__":
    main()

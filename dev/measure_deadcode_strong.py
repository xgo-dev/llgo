#!/usr/bin/env python3
"""Compare ordinary Drop, IR-refined strong overrides, and Drop+ThinLTO on macOS.

Use compilers built against the same upstream revision. All variants compile
the same sources and runtime. Timings include compilation/linking/PCLN, exclude
downloads, symbol inspection, stripping/signing and execution. Results include
failed runs. Function counts are unique __TEXT,__text entries, not Go methods.
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import resource
import shlex
import shutil
import statistics
import subprocess
import time


def output(command, **kwargs):
    return subprocess.check_output(command, text=True, stderr=subprocess.STDOUT, **kwargs)


def inspect(exe, destination):
    nm = output(["nm", "--defined-only", str(exe)])
    sections = output(["otool", "-l", str(exe)])
    destination.with_suffix(".nm").write_text(nm)
    destination.with_suffix(".sections").write_text(sections)
    lines = sections.splitlines()
    values = {}
    for index, line in enumerate(lines):
        if line.strip().startswith("sectname "):
            section, segment = line.split()[1], lines[index + 1].split()[1]
            values[segment + "," + section] = {
                "address": int(lines[index + 2].split()[1], 16),
                "bytes": int(lines[index + 3].split()[1], 16),
            }
    text = values["__TEXT,__text"]
    names = {}
    for line in nm.splitlines():
        fields = line.split(maxsplit=2)
        if len(fields) != 3 or fields[1] not in ("t", "T"):
            continue
        address = int(fields[0], 16)
        if text["address"] <= address < text["address"] + text["bytes"]:
            names[fields[2]] = address
    starts_text = output(["llvm-objdump", "--macho", "--function-starts=addrs", str(exe)])
    starts = {int(line, 16) for line in starts_text.splitlines()
              if re.fullmatch(r"[0-9a-fA-F]{16}", line)}
    if starts != set(names.values()):
        raise RuntimeError("nm code entries differ from LC_FUNCTION_STARTS")
    destination.with_suffix(".functions.json").write_text(json.dumps(names, indent=2) + "\n")
    return {"function_bodies": len(starts), "function_names": len(names),
            "text_bytes": text["bytes"], "sections": values,
            "unrelated_json_present": any("UnrelatedJSON" in name for name in names)}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--baseline", required=True, type=Path)
    p.add_argument("--strong", required=True, type=Path)
    p.add_argument("--thin", required=True, type=Path)
    p.add_argument("--source-root", type=Path, default=Path(__file__).resolve().parent.parent)
    p.add_argument("--gorm-module", required=True, type=Path)
    p.add_argument("--linker", required=True, type=Path)
    p.add_argument("--output", required=True, type=Path)
    p.add_argument("--repeat", type=int, default=3)
    args = p.parse_args()
    if platform.system() != "Darwin" or args.repeat < 1:
        p.error("requires macOS and a positive repeat count")
    root, directory = args.source_root.resolve(), args.output.resolve()
    directory.mkdir(parents=True, exist_ok=True)
    if (directory / "results.json").exists():
        p.error("use a new output directory")
    compilers = {name: getattr(args, name).resolve() for name in ("baseline", "strong", "thin")}
    flags = "-Wl,-no_exported_symbols " + shlex.quote("--ld-path=" + str(args.linker.absolute()))
    env = dict(os.environ, LLGO_ROOT=str(root), LDFLAGS=flags)
    record = {"host": platform.platform(), "source_revision": output(["git", "rev-parse", "HEAD"], cwd=root).strip(),
              "linker": output([str(args.linker.absolute()), "--version"]).strip(),
              "compilers": {name: {"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()}
                            for name, path in compilers.items()}, "runs": []}
    (directory / "source.diff").write_text(output(["git", "diff", "HEAD"], cwd=root))
    module = json.loads(output(["go", "list", "-m", "-json", "gorm.io/gorm"], cwd=args.gorm_module))
    record["gorm"] = module
    def save():
        (directory / "results.json").write_text(json.dumps(record, indent=2) + "\n")
    modes = list(compilers)
    for repeat in range(args.repeat):
        for case in ("reflectcallbacks", "gormschema"):
            for mode in modes[repeat % 3:] + modes[:repeat % 3]:
                stem = directory / f"{case}-{mode}-{repeat}"
                exe = directory / "probe"
                gorm = case == "gormschema"
                command = [str(compilers[mode]), *(["test", "-c"] if gorm else ["build"]),
                           "-a", "-Os", "-p=10", "-debug-artifact=none", "-deadcodedrop"]
                if mode == "thin":
                    command += ["-lto=thin"]
                command += ["-o", str(exe), "gorm.io/gorm/schema" if gorm else "./benchmark/binary_size/reflectcallbacks"]
                before = resource.getrusage(resource.RUSAGE_CHILDREN)
                start = time.monotonic()
                with stem.with_suffix(".log").open("w") as log:
                    build = subprocess.run(command, cwd=args.gorm_module if gorm else root,
                                           env=env, stdout=log, stderr=subprocess.STDOUT)
                elapsed = time.monotonic() - start
                after = resource.getrusage(resource.RUSAGE_CHILDREN)
                row = {"case": case, "mode": mode, "repeat": repeat, "command": command,
                       "seconds": elapsed, "cpu_seconds": after.ru_utime + after.ru_stime - before.ru_utime - before.ru_stime,
                       "returncode": build.returncode}
                record["runs"].append(row)
                save()
                if build.returncode == 0:
                    row.update(inspect(exe, stem))
                    output(["strip", "-x", str(exe)])
                    output(["codesign", "-f", "-s", "-", str(exe)])
                    output(["codesign", "--verify", str(exe)])
                    row["bytes"] = exe.stat().st_size
                    shutil.copy2(exe, stem)
                    command = [str(exe)] + (["-test.v", "-test.timeout=120s"] if gorm else [])
                    try:
                        run = subprocess.run(command, cwd=Path(module["Dir"]) / "schema" if gorm else root,
                                             text=True, capture_output=True, timeout=140)
                        log = run.stdout + run.stderr
                        row.update(run_returncode=run.returncode,
                                   output_ok=run.returncode == 0 and (gorm or log == "before-create\nafter-find\n"))
                        row["tests_passed"] = len(re.findall(r"^--- PASS:", log, re.M)) if gorm else None
                    except subprocess.TimeoutExpired as exc:
                        log = str(exc)
                        row.update(run_returncode=None, output_ok=False, timed_out=True)
                    stem.with_suffix(".run.log").write_text(log)
                save()
                print(json.dumps({k: row[k] for k in ("case", "mode", "repeat", "seconds", "bytes", "function_bodies", "output_ok") if k in row}), flush=True)
    summary = []
    for case in ("reflectcallbacks", "gormschema"):
        for mode in modes:
            rows = [r for r in record["runs"] if r["case"] == case and r["mode"] == mode]
            item = {"case": case, "mode": mode, "valid": all(r.get("output_ok") for r in rows)}
            for metric in ("bytes", "seconds", "cpu_seconds", "function_bodies", "text_bytes"):
                if all(metric in row for row in rows):
                    item[metric] = statistics.median(row[metric] for row in rows)
            summary.append(item)
    (directory / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Compare documented Go CLI flags with the reviewed compatibility snapshot.

Read public help, then discard references to flags of another command using
parse-only probes. Each probe ends in -h, before any command can perform work.
Private debugging flags absent from public help are deliberately excluded.
"""
import argparse
import difflib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
SNAPSHOT = ROOT / "cmd/internal/compat/go1.27.json"
ENV = dict(os.environ, GOENV="off", GOFLAGS=" ", GOWORK="off", GOTOOLCHAIN="local")


def go(*args):
    result = subprocess.run(["go", *args], capture_output=True, text=True, env=ENV)
    return result.stdout + result.stderr


def subcommands(text):
    section = text.split("The commands are:", 1)[1].split('Use "go help', 1)[0]
    return re.findall(r"^\s+([a-z]+)\s{2,}\S", section, re.M)


def documented(command):
    text = go("help", *command.split())
    if command == "build":
        text = text.split("The -asmflags, -gccgoflags, -gcflags, and -ldflags flags accept")[0]
        names = set(re.findall(r"^\s+-([A-Za-z][A-Za-z0-9]*)(?:\s|,|=|$)", text, re.M))
    elif command == "test":
        text += "\n" + go("help", "testflag")
        names = set(re.findall(r"^\s+-([A-Za-z][A-Za-z0-9]*)(?:\s|,|=|$)", text, re.M))
        names.add("args")
    else:
        names = set(re.findall(r'''(?<![\w/])-([A-Za-z][A-Za-z0-9]*)(?=[\s=,\]\|'".;]|$)''', text))
    names.update(re.findall(r"-([A-Za-z][A-Za-z0-9]*)", text.split("\n\n")[0]))
    return names


def inventory():
    pinned = (ROOT / ".go-version").read_text().strip()
    actual = go("env", "GOVERSION").strip()
    if actual != "go" + pinned:
        raise RuntimeError(f"use pinned Go {pinned}, found {actual}")
    shared = documented("build") - {"o", "json"}
    commands = {}
    for command in subcommands(go("help")):
        names = documented(command)
        if command in {"build", "clean", "get", "install", "list", "run", "test", "fix", "generate", "vet"}:
            names |= shared
        commands[command] = names
        if command in {"mod", "work"}:
            for sub in subcommands(go("help", command)):
                full = command + " " + sub
                commands[full] = documented(full)
    # Unlike prose help, compiler usage lists its registered flags directly.
    # Its -h means halt-on-error, not parse-only help: never probe it with
    # artifact/profile flags, which could create files before reporting usage.
    commands["tool compile"] = set(re.findall(r"^\s+-([^\s]+)", go("tool", "compile", "-h"), re.M))
    for command, names in commands.items():
        if command in {"test", "tool compile"}:
            # go test forwards unknown flags to the test binary. Only the
            # documented flag headings above describe its public contract.
            continue
        for name in sorted(names.copy()):
            output = go(*command.split(), f"-{name}=__llgo_probe__", "-h")
            if "flag provided but not defined" in output or "unknown flag" in output:
                names.remove(name)
    prefix = "test." if "optional 'test.' prefix" in go("help", "testflag") else ""
    return {"go_version": pinned, "test_flag_prefix": prefix, "commands": {k: sorted(v) for k, v in sorted(commands.items())}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--update", action="store_true", help="write a snapshot for review after a Go upgrade")
    args = parser.parse_args()
    updated = json.dumps(inventory(), indent=2) + "\n"
    if args.update:
        SNAPSHOT.write_text(updated)
    elif updated != SNAPSHOT.read_text():
        sys.stdout.writelines(difflib.unified_diff(SNAPSHOT.read_text().splitlines(True), updated.splitlines(True), fromfile="reviewed Go surface", tofile="pinned toolchain surface"))
        return 1
    else:
        print("Go CLI compatibility snapshot matches the pinned toolchain")
    return 0


if __name__ == "__main__":
    sys.exit(main())

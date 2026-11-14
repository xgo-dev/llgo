#!/usr/bin/env python3
# Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

"""Qualify the real threaded W32/EH module with Wasmtime's native JIT DWARF."""

import argparse
import json
import os
from pathlib import Path
import shlex
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[3]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path)
    args = parser.parse_args()
    llgo = os.environ.get('LLGO', 'llgo')
    engine = os.environ.get('LLGO_WASMTIME', 'wasmtime')
    lldb = os.environ.get('LLGO_LLDB', 'lldb')
    environment = dict(os.environ, LLGO_ROOT=str(ROOT))
    report = {name: subprocess.check_output(command, text=True).strip() for name, command in {
        'go': ['go', 'version'], 'llgo': [llgo, 'version'],
        'wasmtime': [engine, '--version'], 'lldb': [lldb, '--version'],
        'node': ['node', '--version'], 'clang': ['clang', '--version'],
    }.items()}
    report['sessions'] = []
    with tempfile.TemporaryDirectory(prefix='llgo-wasi-debug-') as directory:
        directory = Path(directory)
        for mode in ('embedded', 'external'):
            module, result = directory / (mode + '.wasm'), directory / (mode + '.result')
            script = ROOT / 'test/debug/wasi/accept.py'
            command = [llgo, 'debug', '-target', 'wasi', '-wasmtime', engine,
                       '-lldb', lldb, '-debug-artifact=' + mode, '-o', str(module),
                       '.', '--', '--batch',
                       '-o', 'command script import ' + shlex.quote(str(script)),
                       '-o', 'script accept.run(lldb.debugger, ' + repr(str(result)) + ')']
            subprocess.run(command, cwd=ROOT / 'test/debug/wasm', env=environment, check=True, timeout=300)
            if not result.exists() or result.read_text(encoding='utf-8') != 'ok':
                raise SystemExit(mode + ' WASI debugger acceptance failed')
            report['sessions'].append({'mode': mode, 'deployment_bytes': module.stat().st_size,
                                       'files': {p.name: p.stat().st_size for p in directory.glob(mode + '*')
                                                 if p.suffix != '.result'}})
    encoded = json.dumps(report, indent=2, ensure_ascii=False) + '\n'
    print(encoded, end='')
    if args.report:
        args.report.write_text(encoded, encoding='utf-8')


if __name__ == '__main__':
    main()

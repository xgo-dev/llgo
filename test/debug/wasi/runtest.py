#!/usr/bin/env python3
# Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
# Licensed under the Apache License, Version 2.0.

"""Qualify the real threaded W32/EH module with Wasmtime's native JIT DWARF."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[3]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--report', type=Path)
    args = parser.parse_args()
    llgo = os.environ.get('LLGO', 'llgo')
    engine = os.environ.get('LLGO_WASI_HOST', 'llgo-wasi-debug')
    lldb = os.environ.get('LLGO_LLDB', 'lldb')
    environment = dict(os.environ, LLGO_ROOT=str(ROOT))
    report = {name: subprocess.check_output(command, text=True).strip() for name, command in {
        'go': ['go', 'version'], 'llgo': [llgo, 'version'],
        'wasi_host': [engine, '--version'], 'lldb': [lldb, '--version'],
        'node': ['node', '--version'], 'clang': ['clang', '--version'],
    }.items()}
    report['sessions'] = []
    report['source_commit'] = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    report['source_dirty'] = bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True).strip())
    report['compiler_sha256'] = hashlib.sha256(Path(shutil.which(llgo) or llgo).read_bytes()).hexdigest()
    report['wasi_host_sha256'] = hashlib.sha256(Path(shutil.which(engine) or engine).read_bytes()).hexdigest()
    fixture = ROOT / 'test/debug/wasm'
    report['fixture_sha256'] = {str(path.relative_to(fixture)): hashlib.sha256(path.read_bytes()).hexdigest()
                                for path in sorted(fixture.rglob('*'))
                                if path.is_file() and path.suffix in ('.go', '.cpp', '.mod')}
    host_sources = ROOT / 'test/debug/wasi/host'
    report['host_source_sha256'] = {
        str(path.relative_to(host_sources)): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in sorted(host_sources.rglob('*'))
        if path.is_file() and 'target' not in path.relative_to(host_sources).parts
        and path.suffix in ('.rs', '.toml', '.lock')}
    with tempfile.TemporaryDirectory(prefix='llgo-wasi-debug-') as directory:
        directory = Path(directory)
        fd_module = directory / 'fd-sharing.wasm'
        subprocess.run([llgo, 'build', '-target=wasi', '-o', str(fd_module), '.'],
                       cwd=ROOT / 'test/debug/wasi/fdsharing', env=environment, check=True, timeout=300)
        shared_fd = subprocess.run([engine, str(fd_module)], capture_output=True, text=True, timeout=60)
        if shared_fd.returncode or 'wasi-shared-fd-ok' not in shared_fd.stdout + shared_fd.stderr:
            raise SystemExit('WASI descriptor-table acceptance failed:\n' + shared_fd.stdout + shared_fd.stderr)
        report['shared_fd'] = 'worker created before open; cross-instance write and close passed'
        report['shared_fd_module_sha256'] = hashlib.sha256(fd_module.read_bytes()).hexdigest()
        for mode, expected in (('exit', 19), ('worker-exit', 23)):
            result = subprocess.run([engine, str(fd_module), mode], capture_output=True,
                                    text=True, timeout=60)
            if result.returncode != expected:
                raise SystemExit(f'WASI {mode}: expected exit {expected}, got {result.returncode}:\n'
                                 + result.stdout + result.stderr)
        report['process_exit'] = 'main and worker proc_exit preserve nonzero process status'
        for mode in ('embedded', 'external'):
            module, result = directory / (mode + '.wasm'), directory / (mode + '.result')
            script = ROOT / 'test/debug/wasi/accept.py'
            command = [llgo, 'debug', '-target', 'wasi', '-wasi-host', engine,
                       '-lldb', lldb, '-debug-artifact=' + mode, '-o', str(module),
                       '.', '--', '--batch',
                       '-o', 'command script import ' + shlex.quote(str(script)),
                       '-o', 'script accept.run(lldb.debugger, ' + repr(str(result)) + ')']
            subprocess.run(command, cwd=ROOT / 'test/debug/wasm', env=environment, check=True, timeout=300)
            if not result.exists() or result.read_text(encoding='utf-8') != 'ok':
                raise SystemExit(mode + ' WASI debugger acceptance failed')
            report['sessions'].append({'mode': mode, 'deployment_bytes': module.stat().st_size,
                                       'files': {p.name: p.stat().st_size for p in directory.glob(mode + '*')
                                                 if p.suffix != '.result'},
                                       'sha256': {p.name: hashlib.sha256(p.read_bytes()).hexdigest()
                                                  for p in directory.glob(mode + '*') if p.suffix != '.result'}})
    encoded = json.dumps(report, indent=2, ensure_ascii=False) + '\n'
    print(encoded, end='')
    if args.report:
        args.report.write_text(encoded, encoding='utf-8')


if __name__ == '__main__':
    main()

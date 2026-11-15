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
        inherited = subprocess.run([engine, str(fd_module), 'args-env', '--flag', 'two words'],
                                   env=dict(os.environ, LLGO_WASI_DEBUG_ENV='inherited value', LLGO_WASI_NON_UTF8=''),
                                   capture_output=True, text=True, timeout=60)
        if inherited.returncode or 'wasi-args-env-ok' not in inherited.stdout + inherited.stderr:
            raise SystemExit('WASI guest arguments/environment acceptance failed:\n'
                             + inherited.stdout + inherited.stderr)
        guest_env = dict(os.environ, LLGO_WASI_DEBUG_ENV='inherited value', LLGO_WASI_NON_UTF8='')
        if os.name == 'posix':
            encoded_env = {os.fsencode(key): os.fsencode(value) for key, value in guest_env.items()}
            encoded_env[b'LLGO_WASI_NON_UTF8'] = b'\xff'
            encoded_env[b'LLGO_WASI_BAD_\xff'] = b'value'
            malformed = subprocess.run([engine, str(fd_module), 'args-env', '--flag', 'two words'],
                                       env=encoded_env, capture_output=True, text=True, timeout=60)
            if malformed.returncode or 'wasi-args-env-ok' not in malformed.stdout + malformed.stderr:
                raise SystemExit('WASI non-UTF-8 environment acceptance failed:\n'
                                 + malformed.stdout + malformed.stderr)
        lldb_args = subprocess.run([
            lldb, '--batch', '-o', 'target create ' + shlex.quote(str(shutil.which(engine) or engine)),
            '-o', 'settings set target.disable-aslr false',
            '-o', 'settings set -- target.run-args ' + shlex.quote(str(fd_module)),
            '-o', 'settings append target.run-args args-env --flag "two words"',
            '-o', 'run', '-o',
            'script p = lldb.debugger.GetSelectedTarget().GetProcess(); '
            'assert p.GetState() == lldb.eStateExited and p.GetExitStatus() == 0; '
            'print("wasi-args-env-lldb-ok")',
        ], env=guest_env, capture_output=True, text=True, timeout=90)
        lldb_lines = {line.strip() for line in (lldb_args.stdout + lldb_args.stderr).splitlines()}
        if lldb_args.returncode or not {'wasi-args-env-ok', 'wasi-args-env-lldb-ok'} <= lldb_lines:
            raise SystemExit('WASI LLDB guest argument acceptance failed:\n'
                             + lldb_args.stdout + lldb_args.stderr)
        report['guest_process'] = 'argv with spaces/flags, LLDB forwarding and inherited worker environment passed'
        report['environment_encoding'] = 'non-UTF-8 host keys/values omitted' if os.name == 'posix' else 'Unicode host environment'
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

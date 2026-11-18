#!/usr/bin/env python3
"""Build firmware with the ESP runtime contained in an installed Unix archive."""
import argparse
import json
import os
from pathlib import Path
import platform
import subprocess
import tempfile


def run(command, env, cwd, log):
    result = subprocess.run([str(arg) for arg in command], env=env, cwd=cwd,
                            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                            text=True, timeout=900)
    log.write_text(result.stdout)
    if result.returncode:
        raise RuntimeError(f"{command} exited {result.returncode}:\n{result.stdout[-24000:]}")
    return result.stdout


def validate(root, output):
    pins = json.loads((Path(__file__).resolve().parents[2] /
                       'internal/crosscompile/espressif.json').read_text())
    goos = {'Darwin': 'darwin', 'Linux': 'linux'}[platform.system()]
    arch = {'arm64': 'aarch64', 'aarch64': 'aarch64',
            'x86_64': 'x86_64', 'AMD64': 'x86_64'}[platform.machine()]
    target = arch + {'darwin': '-apple-darwin', 'linux': '-linux-gnu'}[goos]
    binary = root / 'bin/llgo'
    bundled = root / 'crosscompile/clang'
    manifest = (bundled / 'LLGO-LLVM-MANIFEST.txt').read_text().splitlines()
    for line in (f"payload_version={pins['release']}", f'host_target={target}',
                 f"llvm_expected_version={pins['llvm_version']}", 'payload_profile=runtime'):
        if line not in manifest:
            raise ValueError('Bundled runtime is missing ' + line)
    if not (bundled / 'LICENSE-LLVM.txt').is_file():
        raise ValueError('Bundled runtime is missing its LLVM license')
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='esp-bundled-', dir=output) as temporary:
        source = Path(temporary)
        (source / 'go.mod').write_text('module esp-bundled-smoke\n\ngo 1.27.0\n')
        (source / 'main.go').write_text('package main\nfunc main() {}\n')
        env = os.environ.copy()
        for name in ('LLGO_ROOT', 'LLVM_CONFIG', 'CC', 'CXX', 'CGO_CPPFLAGS', 'CGO_CFLAGS',
                     'CGO_CXXFLAGS', 'CGO_LDFLAGS', 'LD_LIBRARY_PATH',
                     'DYLD_LIBRARY_PATH', 'DYLD_FALLBACK_LIBRARY_PATH', 'GOOS', 'GOARCH'):
            env.pop(name, None)
        env.update(LLGO_BUILD_CACHE='0', GOTOOLCHAIN='local', CGO_ENABLED='1',
                   XDG_CACHE_HOME=str(source / 'empty cache'))
        results = []
        print(f"Testing bundled ESP {pins['release']} on {target}", flush=True)
        for board, machine in (('esp32-coreboard-v2', 94), ('esp32c3', 243)):
            for mode in ('default', 'full'):
                name = f'{board}-{mode}'
                command = [binary, 'build', '-x']
                if mode == 'full':
                    command.append('-lto=full')
                command += ['-target', board, '-o', source / (name + '.out'), '.']
                trace = run(command, env, source, output / (name + '.log'))
                if str(bundled / 'bin') not in trace:
                    raise ValueError('Firmware did not use the installed bundled toolchain')
                if 'ESP Clang not found' in trace or 'Downloading ESP Clang' in trace:
                    raise ValueError('Installed package attempted an ESP toolchain download')
                header = (source / (name + '.out.elf')).read_bytes()[:20]
                if header[:6] != b'\x7fELF\x01\x01' or int.from_bytes(header[18:20], 'little') != machine:
                    raise ValueError('Firmware has the wrong Xtensa/RISC-V ELF machine')
                results.append(name)
                print('PASS ' + name, flush=True)
        (output / 'results.json').write_text(json.dumps({
            'repository': pins['repository'], 'release': pins['release'], 'target': target,
            'source': 'bundled-directory', 'firmware': results, 'status': 'passed',
        }, indent=2) + '\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    validate(args.root.resolve(), args.output.resolve())

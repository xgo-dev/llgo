#!/usr/bin/env python3
"""Exercise external runtime symbol maps with real Wasm hosts."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def run_load_policies(llgo, target, directory, flags, build_environment, environment, browser):
    cases = []
    for policy in ['required', 'background', 'none']:
        work = directory / policy
        work.mkdir()
        output = work / ('app.wasm' if target == 'wasi' else 'app.mjs')
        command = [llgo, 'build', *flags, '-pclntab-load=' + policy, '-o', str(output),
                   './cmd/compiler/build/testdata/wasm-pclntab-policy']
        subprocess.run(command, cwd=ROOT, env=build_environment, check=True, timeout=300)
        if target == 'wasi':
            wasm_runner = os.environ.get('WASMER', 'wasmer')
            run = [wasm_runner, 'run', '--enable-exceptions', '--enable-simd', '--stack-size=1048576',
                   '--volume=' + str(work) + ':/home', '--env=PWD=/home', './app.wasm', '--', policy]
        else:
            run = ['node', str(ROOT / 'targets/emscripten-memory64-runner.mjs'), str(output), policy]

        def execute():
            return subprocess.run(run, cwd=work, env=environment, capture_output=True, text=True, timeout=60)

        result = execute()
        if result.returncode or 'pcln-policy-ok' not in result.stdout + result.stderr:
            raise RuntimeError(f'{target}/{policy}: exit {result.returncode}\n{result.stdout}{result.stderr}')
        cases.append(policy)
        if policy == 'required':
            sidecar = work / 'app.wasm.pclntab'
            original = sidecar.read_bytes()
            if target == 'wasi' and len(original) > 64 * 1024:
                raise RuntimeError(f'WASI logical-location map grew to {len(original)} bytes')
            for failure in ['missing', 'corrupt']:
                if failure == 'missing':
                    sidecar.unlink()
                else:
                    sidecar.write_bytes(original[:-1] + bytes([original[-1] ^ 1]))
                result = execute()
                combined = result.stdout + result.stderr
                if (not result.returncode or 'unable to preload required PCLN metadata' not in combined
                        or 'pcln-policy-init' in combined):
                    raise RuntimeError(f'{target}/{policy}/{failure}: exit {result.returncode}\n{combined}')
                cases.append(policy + '-' + failure)
            sidecar.write_bytes(original)
        if browser:
            response_mode = {'background': 'delayed', 'none': 'no-symbols', 'required': 'delayed'}[policy]
            argument = 'background-pending' if policy == 'background' else policy
            browser_env = dict(environment, LLGO_BROWSER_REQUIRE_EXIT='1', LLGO_BROWSER_PCLN=response_mode)
            subprocess.run(['node', str(ROOT / 'dev/test_wasm_browser.mjs'), str(output), 'pcln-policy-ok', argument],
                           cwd=ROOT, env=browser_env, check=True, timeout=70)
            cases.append('browser-' + policy)
    return cases


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--target', choices=['all', 'wasi', 'gojs', 'emscripten', 'emscripten-memory64'], default='all')
    parser.add_argument('--report', type=Path)
    parser.add_argument('--browser', action='store_true', help='also require real browser HTTP acceptance')
    args = parser.parse_args()
    llgo = os.environ.get('LLGO', 'llgo')
    targets = ['wasi', 'gojs', 'emscripten', 'emscripten-memory64'] if args.target == 'all' else [args.target]
    environment = dict(os.environ, LLGO_ROOT=str(ROOT))
    reports = []
    with tempfile.TemporaryDirectory(prefix='llgo-pcln-') as temporary:
        for target in targets:
            directory = Path(temporary) / target
            directory.mkdir()
            module = directory / 'app.wasm'
            output = module if target == 'wasi' else directory / 'app.mjs'
            flags = ['-Oz', '-debug-artifact=none', '-ldflags=-s', '-pclntab=external']
            build_environment = dict(environment)
            build_environment.pop('LLGO_WASM_WORKERS', None)
            if target == 'gojs':
                build_environment.update(GOOS='js', GOARCH='wasm')
            else:
                flags += ['-target=' + target]
            if target in ('emscripten', 'emscripten-memory64'):
                build_environment['LLGO_WASM_WORKERS'] = '2'
            command = [llgo, 'build', *flags, '-o', str(output), './cmd/compiler/build/testdata/wasm-pclntab']
            subprocess.run(command, cwd=ROOT, env=build_environment, check=True, timeout=300)
            sidecar = Path(str(module) + '.pclntab')
            original = sidecar.read_bytes()
            if target == 'wasi':
                wasmer = os.environ.get('WASMER', 'wasmer')
                run = [wasmer, 'run', '--enable-exceptions', '--enable-simd', '--stack-size=1048576',
                       '--volume=' + str(directory) + ':/home', '--env=PWD=/home', './app.wasm']
                guest_sidecar = '/home/app.wasm.pclntab'
            else:
                runner = 'emscripten-memory64-runner.mjs' if target == 'emscripten-memory64' else 'emscripten-runner.mjs'
                run = ['node', str(ROOT / 'targets' / runner)]
                if target == 'gojs':
                    run += ['--browser-only']
                run += [str(output)]
                guest_sidecar = str(sidecar)

            def execute(mode, status=0, output='pcln-ok'):
                result = subprocess.run(run + [mode, guest_sidecar], cwd=directory, env=environment,
                                        capture_output=True, text=True, timeout=60)
                combined = result.stdout + result.stderr
                if result.returncode != status or output not in combined:
                    raise RuntimeError(f'{target}/{mode}: exit {result.returncode}\n{combined}')
                return combined

            cases = []

            # A valid map exists, but a fatal traceback must neither start a
            # load nor wait for one. Resolve the recorded IDs on the host.
            fatal = execute('panic', status=2, output='panic: pcln-fatal')
            if '[llgo-location=' not in fatal or 'pcln-callsite.go:' in fatal:
                raise RuntimeError(f'{target}: fatal path loaded source metadata\n{fatal}')
            cases.append('fatal-without-load')
            resolved = subprocess.run([llgo, 'tool', 'symbolize', str(module)], input=fatal, cwd=ROOT,
                                      env=environment, capture_output=True, text=True, timeout=60)
            if resolved.returncode or 'pcln-callsite.go:' not in resolved.stdout or 'panic: pcln-fatal' not in resolved.stdout:
                raise RuntimeError(f'{target}: offline symbolization failed\n{resolved.stdout}{resolved.stderr}')
            cases.append('offline-symbolization')
            for mode in ['available', 'cache-reuse', 'concurrent', 'cached-success', 'cached-failure']:
                execute(mode)
                if sidecar.read_bytes() != original:
                    raise RuntimeError('fixture did not restore its sidecar')
                cases.append(mode)
            def damaged_record(section, relative, value, width=4):
                # Recompute the checksum so these cases reach the decoder's
                # record/ID validation rather than only testing its header.
                raw = bytearray(original)
                offset = int.from_bytes(raw[96 + section * 16:104 + section * 16], 'little') + relative
                raw[offset:offset + width] = value.to_bytes(width, 'little')
                checksum = 14695981039346656037
                for byte in raw[256:]:
                    checksum = ((checksum ^ byte) * 1099511628211) & ((1 << 64) - 1)
                raw[32:40] = checksum.to_bytes(8, 'little')
                return bytes(raw)

            for name, damaged in [('truncated', original[:64]), ('magic', b'?' + original[1:]),
                                  ('version', original[:8] + bytes([255]) + original[9:]),
                                  ('identity', original[:64] + bytes([original[64] ^ 1]) + original[65:]),
                                  ('checksum', original[:-1] + bytes([original[-1] ^ 1])),
                                  ('descriptor', original[:96] + (len(original) + 1).to_bytes(8, 'little') + original[104:]),
                                  ('record-string-id', damaged_record(0, 0, 0xffffffff)),
                                  ('pcline-owner', damaged_record(1, 8, 0xffffffff)),
                                  ('string-offset', damaged_record(3, 0, 0xffffffff))]:
                try:
                    sidecar.write_bytes(damaged)
                    execute('missing')
                    cases.append(name)
                finally:
                    sidecar.write_bytes(original)
            backup = sidecar.with_suffix('.saved')
            sidecar.rename(backup)
            try:
                execute('missing')
                cases.append('missing')
                sidecar.mkdir()
                execute('missing')
                sidecar.rmdir()
                cases.append('directory')
                with sidecar.open('wb') as file:
                    file.truncate((64 << 20) + 1)
                execute('missing')
                sidecar.unlink()
                cases.append('oversized')
                if hasattr(os, 'mkfifo'):
                    os.mkfifo(sidecar)
                    execute('missing')
                    sidecar.unlink()
                    cases.append('fifo')
            finally:
                if sidecar.is_dir():
                    sidecar.rmdir()
                elif sidecar.exists():
                    sidecar.unlink()
                backup.rename(sidecar)
            if target == 'wasi':
                # The real CLI uses a separate temporary artifact directory.
                # It must preopen the map rather than rely on cwd visibility.
                cli_environment = dict(build_environment)
                if Path(wasmer).is_absolute():
                    cli_environment['PATH'] = str(Path(wasmer).parent) + os.pathsep + cli_environment.get('PATH', '')
                for subcommand, expected in [('run', 'pcln-ok'), ('test', 'PASS')]:
                    command = [llgo, subcommand, *flags, '-emulator']
                    if subcommand == 'test':
                        command += ['-v']
                    result = subprocess.run(command + ['./cmd/compiler/build/testdata/wasm-pclntab'],
                                            cwd=ROOT, env=cli_environment, capture_output=True, text=True, timeout=300)
                    if result.returncode or expected not in result.stdout + result.stderr:
                        raise RuntimeError(f'WASI CLI {subcommand}: {result.stdout}{result.stderr}')
                    cases.append('cli-' + subcommand)
            browser_cases = []
            if args.browser and target != 'wasi':
                if target == 'gojs':
                    # Exercise the actual GoJS host, whose Emscripten runtime
                    # intentionally has no onExit callback in ordinary builds.
                    output = directory / 'browser' / 'app.wasm'
                    subprocess.run([llgo, 'build', *flags, '-o', str(output), './cmd/compiler/build/testdata/wasm-pclntab'],
                                   cwd=ROOT, env=build_environment, check=True, timeout=300)
                for mode in ['available', 'cache-reuse', 'concurrent', 'no-symbols', 'missing', 'corrupt', 'stale', 'oversized', 'timeout']:
                    argument = mode if mode in ('available', 'cache-reuse', 'concurrent', 'no-symbols') else 'missing'
                    browser_env = dict(environment, LLGO_BROWSER_REQUIRE_EXIT='1', LLGO_BROWSER_PCLN=mode)
                    subprocess.run(['node', str(ROOT / 'dev/test_wasm_browser.mjs'), str(output), 'pcln-ok', argument],
                                   cwd=ROOT, env=browser_env, check=True, timeout=70)
                    browser_cases.append(mode)
            policy_cases = []
            if target in ('wasi', 'emscripten-memory64'):
                policy_cases = run_load_policies(llgo, target, directory, flags, build_environment, environment,
                                                  args.browser and target != 'wasi')
            elif args.browser and target == 'gojs':
                # An import-free main cannot supply the standard runtime's
                # timer hooks. Delay the map so startup must yield to JS.
                minimal = directory / 'minimal'
                minimal.mkdir()
                source = minimal / 'main.go'
                source.write_text('package main\nfunc main() { println("pcln-required-ok") }\n', encoding='utf-8')
                output = minimal / 'app.wasm'
                minimal_flags = [flag for flag in flags if flag != '-Oz']
                subprocess.run([llgo, 'build', '-O0', *minimal_flags, '-pclntab-load=required', '-o', str(output), str(source)],
                               cwd=ROOT, env=build_environment, check=True, timeout=300)
                browser_env = dict(environment, LLGO_BROWSER_REQUIRE_EXIT='1', LLGO_BROWSER_PCLN='delayed')
                subprocess.run(['node', str(ROOT / 'dev/test_wasm_browser.mjs'), str(output), 'pcln-required-ok'],
                               cwd=ROOT, env=browser_env, check=True, timeout=70)
                policy_cases.append('browser-required-without-imports')
            reports.append({'target': target, 'module_bytes': module.stat().st_size,
                            'sidecar_bytes': len(original), 'cases': cases, 'browser_cases': browser_cases,
                            'load_policy_cases': policy_cases})
            print(json.dumps(reports[-1]), flush=True)
    if args.report:
        args.report.write_text(json.dumps(reports, indent=2) + '\n')


if __name__ == '__main__':
    main()

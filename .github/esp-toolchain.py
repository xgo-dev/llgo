#!/usr/bin/env python3
"""Stage pinned ESP runtime payloads separately from LLGo's build SDKs."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import tarfile
import tempfile
from urllib.request import urlopen
import uuid


ROOT = Path(__file__).resolve().parents[1]
PLATFORMS = {
    'darwin/amd64': 'x86_64-apple-darwin',
    'darwin/arm64': 'aarch64-apple-darwin',
    'linux/amd64': 'x86_64-linux-gnu',
    'linux/arm64': 'aarch64-linux-gnu',
}
MARKER = 'LLGO-ESP-STAGE.json'


def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def cached_archive(url, expected, cache):
    cache.mkdir(parents=True, exist_ok=True)
    archive = cache / (expected + '.tar.xz')
    if not archive.exists():
        partial = archive.with_name(archive.name + '.' + uuid.uuid4().hex + '.partial')
        try:
            with urlopen(url, timeout=120) as source, partial.open('wb') as destination:
                shutil.copyfileobj(source, destination)
            if digest(partial) != expected:
                raise ValueError('ESP runtime archive checksum mismatch: ' + url)
            os.replace(partial, archive)
        finally:
            partial.unlink(missing_ok=True)
    else:
        # Existing cache entries are untrusted; fresh downloads were checked above.
        if digest(archive) != expected:
            raise ValueError('Cached ESP runtime archive checksum mismatch: ' + str(archive))
    return archive


def normalize_hardlinks(runtime):
    # GoReleaser's tar.gz writer stores regular files separately. Use relative
    # symlinks for the tar hard links in producer archives, retaining all names.
    seen = {}
    for path in sorted([*(runtime / 'bin').iterdir(), *(runtime / 'lib').iterdir()]):
        if path.is_symlink() or not path.is_file():
            continue
        info = path.stat()
        identity = (info.st_dev, info.st_ino)
        if identity in seen:
            target = os.path.relpath(seen[identity], path.parent)
            path.unlink()
            path.symlink_to(target)
        else:
            seen[identity] = path


def validate_runtime(runtime, pins, target):
    manifest = (runtime / 'LLGO-LLVM-MANIFEST.txt').read_text().splitlines()
    for line in (f"payload_version={pins['release']}", f'host_target={target}',
                 f"llvm_expected_version={pins['llvm_version']}", 'payload_profile=runtime'):
        if line not in manifest:
            raise ValueError('ESP runtime manifest is missing ' + line)
    for name in ('bin/clang', 'bin/clang++', 'bin/ld.lld',
                 f"lib/clang/{pins['llvm_version'].split('.')[0]}/include/stddef.h"):
        if not (runtime / name).is_file():
            raise ValueError('ESP runtime is missing ' + name)
    for name in ('llvm', 'llvm-c', 'clang', 'clang-c'):
        if (runtime / 'include' / name).exists():
            raise ValueError('ESP runtime contains development headers: ' + name)
    if list((runtime / 'lib').glob('libLLVM*.a')) or list((runtime / 'lib').glob('libclang*.a')):
        raise ValueError('ESP runtime contains development libraries')
    for path in runtime.rglob('*'):
        if path.is_symlink() and not path.resolve(strict=True).is_relative_to(runtime.resolve()):
            raise ValueError('ESP runtime contains an external symbolic link: ' + str(path))


def stage_unix(pins, cache, output, license_file, base_url=None):
    output = output.resolve()
    if ROOT.is_relative_to(output):
        raise ValueError('Runtime staging must not contain the source repository')
    if output.exists() and any(output.iterdir()) and not (output / MARKER).is_file():
        raise ValueError('Refusing to replace an unowned runtime staging directory')
    output.parent.mkdir(parents=True, exist_ok=True)
    base_url = base_url or f"https://github.com/{pins['repository']}/releases/download/{pins['release']}"
    with tempfile.TemporaryDirectory(prefix='esp-stage-', dir=output.parent) as temporary:
        stage = Path(temporary) / 'stage'
        stage.mkdir()
        for platform, target in PLATFORMS.items():
            pin = pins['payloads'][target]
            archive = cached_archive(base_url + '/' + pin['asset'], pin['sha256'], cache)
            unpack = Path(temporary) / 'unpack' / platform
            unpack.mkdir(parents=True)
            with tarfile.open(archive, 'r:xz') as stream:
                stream.extractall(unpack, filter='data')
            runtime = unpack / 'esp-clang'
            validate_runtime(runtime, pins, target)
            normalize_hardlinks(runtime)
            shutil.copyfile(license_file, runtime / 'LICENSE-LLVM.txt')
            destination = stage / platform
            destination.parent.mkdir(parents=True, exist_ok=True)
            runtime.rename(destination)
            if platform.startswith('linux/'):
                payload = stage / 'payloads' / platform / pin['asset']
                payload.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(archive, payload)
            print('Staged pinned ESP runtime: ' + platform, flush=True)
        (stage / MARKER).write_text(json.dumps(pins, indent=2) + '\n')
        backup = output.with_name(output.name + '.old-' + uuid.uuid4().hex)
        if output.exists():
            output.rename(backup)
        try:
            stage.rename(output)
        except OSError:
            if backup.exists():
                backup.rename(output)
            raise
        if backup.exists():
            shutil.rmtree(backup)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cache', type=Path, default=Path('.esp-runtime-downloads'))
    parser.add_argument('--output', type=Path, default=Path('.release-toolchains'))
    args = parser.parse_args()
    pins = json.loads((ROOT / 'internal/crosscompile/espressif.json').read_text())
    stage_unix(pins, args.cache, args.output, ROOT / 'LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt')

"""Prepare reviewable WinGet manifests from six checksummed Windows release ZIPs.

This never uploads release assets or submits to winget-pkgs.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import zipfile

PROFILES = {'msvc': 'MSVC', 'mingw': 'MinGW'}
ARCHITECTURES = {'amd64': 'x64', 'arm64': 'arm64', '386': 'x86'}
SCHEMA = '1.9.0'
REPOSITORY = 'https://github.com/xgo-dev/llgo'


def release_info(artifacts, version, arch, profile):
    path = artifacts / f'llgo{version}.windows-{arch}-{profile}.zip'
    with path.open('rb') as stream:
        checksum = hashlib.file_digest(stream, 'sha256').hexdigest()
    if path.with_suffix('.zip.sha256').read_text().strip() != f'{checksum}  {path.name}':
        raise ValueError(f'ZIP checksum does not match: {path.name}')
    with zipfile.ZipFile(path) as archive:
        for required in ('bin/llgo.exe', 'runtime/go.mod', 'WINDOWS.md'):
            if not archive.getinfo(required).file_size:
                raise ValueError(f'Empty release component: {required}')
        metadata = json.loads(archive.read('release.json'))
        with archive.open('bin/llgo.exe') as compiler:
            compiler_hash = hashlib.file_digest(compiler, 'sha256').hexdigest()
    expected = {'version': version, 'goos': 'windows', 'goarch': arch, 'abi': profile}
    if any(metadata.get(k) != v for k, v in expected.items()):
        raise ValueError(f'ZIP metadata does not match {path.name}')
    if not re.fullmatch(r'[0-9a-f]{40}', metadata.get('commit', '')):
        raise ValueError(f'Missing release commit: {path.name}')
    if metadata.get('compiler_sha256') != compiler_hash:
        raise ValueError(f'Compiler checksum does not match: {path.name}')
    return metadata['commit'], f'{REPOSITORY}/releases/download/v{version}/{path.name}', checksum, compiler_hash


def yaml_scalar(value):
    # JSON quoted strings are also YAML scalars, including versions and URLs.
    return json.dumps(value, ensure_ascii=False)


def write_manifest(directory, filename, kind, body):
    header = f'# yaml-language-server: $schema=https://aka.ms/winget-manifest.{kind}.{SCHEMA}.schema.json\n'
    footer = f'ManifestType: {kind}\nManifestVersion: {SCHEMA}\n'
    (directory / filename).write_text(header + body + footer, encoding='utf-8')


def prepare(artifacts, output, version):
    # Match build-release.ps1: GoReleaser snapshots may use a short commit
    # hash instead of a numeric tag, while paths must remain single components.
    if not re.fullmatch(r'[0-9A-Za-z][0-9A-Za-z.+-]*', version):
        raise ValueError('Invalid release version (omit the leading v)')
    # Verify all six inputs before emitting anything; ABI/architecture mixing
    # and stale hashes must fail before a candidate can reach winget-pkgs.
    releases = {(p, a): release_info(artifacts, version, a, p)
                for p in PROFILES for a in ARCHITECTURES}
    if len({r[0] for r in releases.values()}) != 1:
        raise ValueError('Windows ZIPs belong to different source commits')
    for arch in ARCHITECTURES:
        if len({releases[p, arch][3] for p in PROFILES}) != 1:
            raise ValueError(f'Windows {arch} profiles use different host compilers')
    for profile, label in PROFILES.items():
        identifier = f'XGo.LLGo.{label}'
        directory = output / 'manifests' / 'x' / 'XGo' / 'LLGo' / label / version
        directory.mkdir(parents=True, exist_ok=True)
        common = f'PackageIdentifier: {identifier}\nPackageVersion: {yaml_scalar(version)}\n'
        write_manifest(directory, f'{identifier}.yaml', 'version', common + 'DefaultLocale: en-US\n')
        notes = ('Native compilation requires Go, native Clang, SDK/CRT, pkg-config, '
                 f'and libraries for the {label} ABI. See WINDOWS.md in the package. '
                 'The bundled crosscompile/clang is the ESP toolchain. '
                 'Use one LLGo ABI profile on PATH at a time.')
        locale = common + 'PackageLocale: en-US\nPublisher: XGo\n'
        locale += f'PackageName: {yaml_scalar(f"LLGo ({label})")}\n'
        locale += f'PackageUrl: {REPOSITORY}\nLicense: Apache-2.0\n'
        locale += f'LicenseUrl: {REPOSITORY}/blob/v{version}/LICENSE\n'
        locale += 'ShortDescription: Go compiler based on LLVM.\n'
        locale += f'InstallationNotes: {yaml_scalar(notes)}\n'
        locale += f'Documentations:\n- DocumentLabel: Windows dependencies\n  DocumentUrl: {REPOSITORY}/blob/v{version}/WINDOWS.md\n'
        write_manifest(directory, f'{identifier}.locale.en-US.yaml', 'defaultLocale', locale)
        installer = common + 'InstallerType: zip\nNestedInstallerType: portable\n'
        if profile == 'mingw':
            installer += 'ArchiveBinariesDependOnPath: true\n'
        installer += ('NestedInstallerFiles:\n- RelativeFilePath: bin/llgo.exe\n'
                      '  PortableCommandAlias: llgo\nUpgradeBehavior: uninstallPrevious\n'
                      'Commands:\n- llgo\nInstallers:\n')
        for arch, winget_arch in ARCHITECTURES.items():
            _, url, checksum, _ = releases[profile, arch]
            installer += f'- Architecture: {winget_arch}\n  InstallerUrl: {yaml_scalar(url)}\n  InstallerSha256: {checksum}\n'
        write_manifest(directory, f'{identifier}.installer.yaml', 'installer', installer)
    print(f'Prepared local WinGet candidates in {output}; release publication and submission are separate steps.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--version', required=True)
    args = parser.parse_args()
    prepare(args.artifacts, args.output, args.version)

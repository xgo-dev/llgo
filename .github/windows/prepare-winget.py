"""Prepare WinGet manifests from three checksummed universal Windows ZIPs with bundled ESP.

This never uploads release assets or submits to winget-pkgs.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import zipfile

PACKAGE_ID = 'XGo.LLGo'
ARCHITECTURES = {'amd64': 'x64', 'arm64': 'arm64', '386': 'x86'}
ESP_HOSTS = {'amd64': 'x86_64-w64-mingw32', 'arm64': 'aarch64-w64-mingw32', '386': 'i686-w64-mingw32'}
SCHEMA = '1.12.0'
REPOSITORY = 'https://github.com/xgo-dev/llgo'
# Keep WinGet's x86 scanner away from the legacy ESP/LLVM SDK bundles.
MAX_ZIP_SIZE = 128 * 1024 * 1024


def archive_name(version, arch):
    return f'llgo{version}.windows-{arch}.zip'


def release_info(artifacts, version, arch):
    path = artifacts / archive_name(version, arch)
    sidecar = path.with_suffix('.zip.sha256')
    for required in (path, sidecar):
        if not required.is_file():
            raise ValueError(f'Missing release artifact: {required.name}')
    if path.stat().st_size > MAX_ZIP_SIZE:
        raise ValueError(f'Windows ZIP exceeds the 128 MiB package budget: {path.name}')
    with path.open('rb') as stream:
        checksum = hashlib.file_digest(stream, 'sha256').hexdigest()
    if sidecar.read_text().strip() != f'{checksum}  {path.name}':
        raise ValueError(f'ZIP checksum does not match: {path.name}')
    with zipfile.ZipFile(path) as archive:
        for required in ('bin/llgo.exe', 'runtime/go.mod', 'WINDOWS.md'):
            if not archive.getinfo(required).file_size:
                raise ValueError(f'Empty release component: {required}')
        metadata = json.loads(archive.read('release.json'))
        esp_version = metadata.get('esp_clang_version')
        esp_asset = f'clang-esp-{esp_version}-{ESP_HOSTS[arch]}.tar.xz'
        if (not isinstance(esp_version, str) or not esp_version or
                metadata.get('esp_clang_host') != ESP_HOSTS[arch] or
                metadata.get('esp_clang_asset') != esp_asset or
                not re.fullmatch(r'[0-9a-f]{64}', metadata.get('esp_clang_sha256', ''))):
            raise ValueError(f'Invalid bundled ESP metadata: {path.name}')
        payload = 'crosscompile/' + esp_asset
        if payload not in archive.namelist():
            raise ValueError(f'Missing bundled ESP archive: {path.name}')
        with archive.open(payload) as stream:
            payload_hash = hashlib.file_digest(stream, 'sha256').hexdigest()
        if payload_hash != metadata['esp_clang_sha256']:
            raise ValueError(f'Bundled ESP checksum does not match: {path.name}')
        if any(name.startswith('crosscompile/clang/') for name in archive.namelist()):
            raise ValueError(f'ESP tools must use the bundled archive: {path.name}')
        with archive.open('bin/llgo.exe') as compiler:
            compiler_hash = hashlib.file_digest(compiler, 'sha256').hexdigest()
    expected = {'version': version, 'goos': 'windows', 'goarch': arch, 'supported_abis': ['msvc', 'mingw'],
                'distribution': 'full'}
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
    # Verify every architecture before emitting anything.
    releases = {arch: release_info(artifacts, version, arch) for arch in ARCHITECTURES}
    if len({r[0] for r in releases.values()}) != 1:
        raise ValueError('Windows ZIPs belong to different source commits')
    write_manifests(output, version, {key: (value[1], value[2]) for key, value in releases.items()})
    print(f'Prepared local WinGet candidates in {output}; release publication and submission are separate steps.')


def write_manifests(output, version, installers):
    """Render the same manifests from verified local or published installers."""
    identifier = PACKAGE_ID
    directory = output / 'manifests' / 'x' / 'XGo' / 'LLGo' / version
    directory.mkdir(parents=True, exist_ok=True)
    common = f'PackageIdentifier: {identifier}\nPackageVersion: {yaml_scalar(version)}\n'
    write_manifest(directory, f'{identifier}.yaml', 'version', common + 'DefaultLocale: en-US\n')
    notes = ('Native compilation requires Go, native Clang, SDK/CRT, pkg-config, '
             'and libraries for the selected MSVC or MinGW target. See WINDOWS.md in the package. '
             'The bundled ESP toolchain is verified and unpacked locally on the first embedded build. '
             'One compiler automatically selects the target ABI from the native Clang toolchain.')
    locale = common + 'PackageLocale: en-US\nPublisher: XGo\n'
    locale += 'PackageName: LLGo\nMoniker: llgo\n'
    locale += f'PackageUrl: {REPOSITORY}\nLicense: Apache-2.0\n'
    locale += f'LicenseUrl: {REPOSITORY}/blob/v{version}/LICENSE\n'
    locale += 'ShortDescription: Go compiler based on LLVM.\n'
    locale += f'InstallationNotes: {yaml_scalar(notes)}\n'
    locale += f'Documentations:\n- DocumentLabel: Windows dependencies\n  DocumentUrl: {REPOSITORY}/blob/v{version}/WINDOWS.md\n'
    write_manifest(directory, f'{identifier}.locale.en-US.yaml', 'defaultLocale', locale)
    installer = common + 'InstallerType: zip\nNestedInstallerType: portable\n'
    # Both ABI targets share this package and its host compiler. Release tests qualify
    # the ZIP's portable alias with an isolated DLL search PATH.
    installer += ('NestedInstallerFiles:\n- RelativeFilePath: bin/llgo.exe\n'
                  '  PortableCommandAlias: llgo\nUpgradeBehavior: uninstallPrevious\n'
                  'Commands:\n- llgo\nInstallers:\n')
    for arch, winget_arch in ARCHITECTURES.items():
        url, checksum = installers[arch]
        installer += f'- Architecture: {winget_arch}\n  InstallerUrl: {yaml_scalar(url)}\n  InstallerSha256: {checksum}\n'
    write_manifest(directory, f'{identifier}.installer.yaml', 'installer', installer)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--version', required=True)
    args = parser.parse_args()
    try:
        prepare(args.artifacts, args.output, args.version)
    except (ValueError, OSError) as error:
        parser.exit(1, f'{error}\n')

import importlib.util
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

spec = importlib.util.spec_from_file_location('prepare_winget', Path(__file__).with_name('prepare-winget.py'))
winget = importlib.util.module_from_spec(spec)
spec.loader.exec_module(winget)


class WinGetTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.output = self.root / 'output'
        for profile in winget.PROFILES:
            for arch in winget.ARCHITECTURES:
                self.fixture(profile, arch)

    def fixture(self, profile, arch, version='1.2.3', compiler=None, **overrides):
        path = self.root / f'llgo{version}.windows-{arch}-{profile}.zip'
        compiler = compiler if compiler is not None else f'host-compiler-{arch}'.encode()
        metadata = dict(version=version, goos='windows', goarch=arch, abi=profile, commit='a' * 40,
                        compiler_sha256=hashlib.sha256(compiler).hexdigest())
        metadata.update(overrides)
        with zipfile.ZipFile(path, 'w') as archive:
            archive.writestr('release.json', json.dumps(metadata))
            archive.writestr('bin/llgo.exe', compiler)
            for name in ('runtime/go.mod', 'WINDOWS.md'):
                archive.writestr(name, 'fixture')
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        path.with_suffix('.zip.sha256').write_text(f'{digest}  {path.name}\n')
        return path

    def test_two_profiles_three_architectures_real_hashes(self):
        winget.prepare(self.root, self.output, '1.2.3')
        files = list(self.output.rglob('*.yaml'))
        self.assertEqual(len(files), 6)
        for label in ('MSVC', 'MinGW'):
            path = self.output / 'manifests/x/XGo/LLGo' / label / '1.2.3'
            text = (path / f'XGo.LLGo.{label}.installer.yaml').read_text()
            self.assertIn('Architecture: x64', text)
            self.assertIn('Architecture: arm64', text)
            self.assertIn('Architecture: x86', text)
            self.assertEqual('ArchiveBinariesDependOnPath: true' in text, label == 'MinGW')
            self.assertNotIn('ProductCode', text)
            self.assertNotIn('Scope:', text)
            for arch in ('amd64', 'arm64', '386'):
                archive = self.root / f'llgo1.2.3.windows-{arch}-{label.lower()}.zip'
                self.assertIn(hashlib.sha256(archive.read_bytes()).hexdigest(), text)
                self.assertIn('/releases/download/v1.2.3/' + archive.name, text)

    def test_snapshot_and_prerelease_versions(self):
        # GoReleaser's .Summary can be a short commit hash without a tag.
        for version in ('fb4049f', '123abcd', '1.2.3-rc.1'):
            with self.subTest(version=version):
                for profile in winget.PROFILES:
                    for arch in winget.ARCHITECTURES:
                        self.fixture(profile, arch, version=version)
                winget.prepare(self.root, self.output, version)
                for profile, label in winget.PROFILES.items():
                    directory = self.output / 'manifests/x/XGo/LLGo' / label / version
                    self.assertEqual(len(list(directory.glob('*.yaml'))), 3)
                    text = (directory / f'XGo.LLGo.{label}.installer.yaml').read_text()
                    self.assertIn(f'PackageVersion: "{version}"', text)
                    for arch in winget.ARCHITECTURES:
                        name = f'llgo{version}.windows-{arch}-{profile}.zip'
                        self.assertIn(f'/releases/download/v{version}/{name}', text)
                        self.assertIn(hashlib.sha256((self.root / name).read_bytes()).hexdigest(), text)

    def test_unsafe_versions_rejected_before_output(self):
        for version in ('', '../escape', '/absolute', 'a/b', 'a\\b', 'a b', 'a\nb'):
            with self.subTest(version=version):
                with self.assertRaisesRegex(ValueError, 'Invalid release version'):
                    winget.prepare(self.root, self.output, version)
                self.assertFalse(self.output.exists())

    def test_corrupted_archive_rejected_before_output(self):
        path = self.root / 'llgo1.2.3.windows-arm64-mingw.zip'
        path.write_bytes(path.read_bytes() + b'tampered')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            winget.prepare(self.root, self.output, '1.2.3')
        self.assertFalse(self.output.exists())

    def test_wrong_architecture_rejected(self):
        self.fixture('mingw', 'arm64', goarch='amd64')
        with self.assertRaisesRegex(ValueError, 'metadata'):
            winget.prepare(self.root, self.output, '1.2.3')

    def test_different_host_compilers_rejected_before_output(self):
        self.fixture('mingw', '386', compiler=b'different host compiler')
        with self.assertRaisesRegex(ValueError, 'different host compilers'):
            winget.prepare(self.root, self.output, '1.2.3')
        self.assertFalse(self.output.exists())

    def test_stale_compiler_hash_rejected_before_output(self):
        self.fixture('msvc', 'amd64', compiler_sha256='0' * 64)
        with self.assertRaisesRegex(ValueError, 'Compiler checksum'):
            winget.prepare(self.root, self.output, '1.2.3')
        self.assertFalse(self.output.exists())

    def test_mixed_commits_rejected(self):
        self.fixture('mingw', 'arm64', commit='b' * 40)
        with self.assertRaisesRegex(ValueError, 'different source commits'):
            winget.prepare(self.root, self.output, '1.2.3')


if __name__ == '__main__':
    unittest.main()

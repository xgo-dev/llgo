import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import tarfile
import tempfile
import unittest


spec = importlib.util.spec_from_file_location('esp_toolchain', Path(__file__).resolve().parents[1] / 'esp-toolchain.py')
esp = importlib.util.module_from_spec(spec)
spec.loader.exec_module(esp)


class ESPStagingTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.sources = self.root / 'published assets'
        self.sources.mkdir()
        self.cache = self.root / 'cache'
        self.output = self.root / 'runtime staging'
        self.license = self.root / 'LLVM license.txt'
        self.license.write_text('Complete LLVM license')
        self.pins = json.loads((esp.ROOT / 'internal/crosscompile/espressif.json').read_text())
        for platform, target in esp.PLATFORMS.items():
            self.fixture(target)

    def fixture(self, target, manifest_target=None, development=False):
        pin = self.pins['payloads'][target]
        archive = self.sources / pin['asset']
        files = {
            'bin/clang': b'compiler', 'bin/ld.lld': b'linker', 'lib/libLLVM.dylib': b'library',
            'lib/clang/22/include/stddef.h': b'resource header',
            'LLGO-LLVM-MANIFEST.txt': (f"payload_version={self.pins['release']}\n"
                f"host_target={manifest_target or target}\nllvm_expected_version={self.pins['llvm_version']}\n"
                'payload_profile=runtime\n').encode(),
        }
        if development:
            files['include/llvm/IR.h'] = b'development SDK file'
        with tarfile.open(archive, 'w:xz') as stream:
            for name, content in files.items():
                entry = tarfile.TarInfo('esp-clang/' + name)
                entry.size = len(content)
                entry.mode = 0o755 if name.startswith('bin/') else 0o644
                stream.addfile(entry, io.BytesIO(content))
            for name, link in [('bin/clang++', 'bin/clang'), ('lib/libLLVM-22.dylib', 'lib/libLLVM.dylib')]:
                entry = tarfile.TarInfo('esp-clang/' + name)
                entry.type = tarfile.LNKTYPE
                entry.linkname = 'esp-clang/' + link
                entry.mode = 0o755 if name.startswith('bin/') else 0o644
                stream.addfile(entry)
        pin['sha256'] = hashlib.sha256(archive.read_bytes()).hexdigest()

    def stage(self):
        esp.stage_unix(self.pins, self.cache, self.output, self.license, self.sources.as_uri())

    def test_stage_runtime_and_original_archives_preserving_aliases(self):
        for _ in range(2):
            self.stage()
            for platform, target in esp.PLATFORMS.items():
                runtime = self.output / platform
                self.assertFalse((runtime / 'stale').exists())
                self.assertEqual((runtime / 'bin/clang++').read_bytes(), b'compiler')
                self.assertTrue(os.access(runtime / 'bin/clang++', os.X_OK))
                self.assertTrue((runtime / 'bin/clang++').is_symlink())
                self.assertFalse(os.path.isabs(os.readlink(runtime / 'bin/clang++')))
                self.assertEqual((runtime / 'lib/libLLVM-22.dylib').read_bytes(), b'library')
                self.assertEqual((runtime / 'LICENSE-LLVM.txt').read_text(), 'Complete LLVM license')
                pin = self.pins['payloads'][target]
                bundled = self.output / 'payloads' / platform / pin['asset']
                if platform.startswith('linux/'):
                    self.assertEqual(esp.digest(bundled), pin['sha256'])
                    self.assertFalse(os.path.samefile(bundled, self.cache / (pin['sha256'] + '.tar.xz')))
                else:
                    self.assertFalse(bundled.parent.exists())
            (self.output / 'linux/arm64/stale').touch()

    def test_cached_archive_revalidated_without_changing_good_output(self):
        self.stage()
        target = esp.PLATFORMS['linux/amd64']
        pin = self.pins['payloads'][target]
        bundled = self.output / 'payloads/linux/amd64' / pin['asset']
        before = bundled.read_bytes()
        (self.cache / (pin['sha256'] + '.tar.xz')).write_bytes(b'tampered cache')
        with self.assertRaisesRegex(ValueError, 'Cached ESP runtime archive checksum mismatch'):
            self.stage()
        self.assertEqual(bundled.read_bytes(), before)

    def test_bad_download_checksum_removes_partial_archive(self):
        target = esp.PLATFORMS['darwin/amd64']
        self.pins['payloads'][target]['sha256'] = '0' * 64
        with self.assertRaisesRegex(ValueError, 'checksum mismatch'):
            self.stage()
        self.assertFalse(self.output.exists())
        self.assertEqual(list(self.cache.iterdir()), [])

    def test_wrong_platform_or_development_sdk_cannot_replace_runtime(self):
        self.stage()
        before = (self.output / esp.MARKER).read_bytes()
        target = esp.PLATFORMS['darwin/amd64']
        for changes, error in [({'manifest_target': 'wrong-host'}, 'host_target'),
                               ({'development': True}, 'development headers')]:
            with self.subTest(changes=changes):
                self.fixture(target, **changes)
                with self.assertRaisesRegex(ValueError, error):
                    self.stage()
                self.assertEqual((self.output / esp.MARKER).read_bytes(), before)

    def test_unowned_output_directory_is_preserved(self):
        self.output.mkdir()
        user_file = self.output / 'user file'
        user_file.write_text('preserve')
        with self.assertRaisesRegex(ValueError, 'unowned'):
            self.stage()
        self.assertEqual(user_file.read_text(), 'preserve')


if __name__ == '__main__':
    unittest.main()

import tempfile
from pathlib import Path
import unittest
import zipfile

import release_zip


class ReleaseZIPTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.stage = self.root / 'release 空格'
        (self.stage / 'empty').mkdir(parents=True)
        (self.stage / 'bin').mkdir()
        (self.stage / 'bin/llgo.exe').write_bytes(bytes(range(256)))
        (self.stage / '.hidden').write_text('hidden release file')
        (self.stage / '说明.txt').write_text('UTF-8 content', encoding='utf-8')
        self.archive = self.root / 'release.zip'

    def test_roundtrip_and_stored_empty_directories(self):
        release_zip.create(self.stage, self.archive)
        with zipfile.ZipFile(self.archive) as archive:
            directory = archive.getinfo('empty/')
            self.assertEqual((directory.file_size, directory.compress_size, directory.compress_type), (0, 0, 0))
            archive.extractall(self.root / 'extracted')
        release_zip.verify(self.archive, self.root / 'extracted')

    def test_bundled_xz_payload_is_stored_byte_for_byte(self):
        payload = self.stage / 'crosscompile/clang-esp-payload.tar.xz'
        payload.parent.mkdir()
        payload.write_bytes(b'already compressed ESP runtime')
        release_zip.create(self.stage, self.archive)
        with zipfile.ZipFile(self.archive) as archive:
            info = archive.getinfo('crosscompile/clang-esp-payload.tar.xz')
            self.assertEqual(info.compress_type, zipfile.ZIP_STORED)
            self.assertEqual(info.compress_size, payload.stat().st_size)
            self.assertEqual(archive.read(info), payload.read_bytes())

    def test_deflated_directory_regression(self):
        release_zip.create(self.stage, self.archive)
        broken = self.root / 'broken.zip'
        with zipfile.ZipFile(self.archive) as source, zipfile.ZipFile(broken, 'w') as target:
            for entry in source.infolist():
                target.writestr(entry.filename, source.read(entry),
                                compress_type=zipfile.ZIP_DEFLATED)
        with self.assertRaisesRegex(ValueError, 'directory must be empty and stored'):
            release_zip.verify(broken, self.stage)

    def test_changed_file(self):
        release_zip.create(self.stage, self.archive)
        (self.stage / 'bin/llgo.exe').write_bytes(b'other compiler')
        with self.assertRaisesRegex(ValueError, 'differs from release'):
            release_zip.verify(self.archive, self.stage)

    def test_missing_file(self):
        release_zip.create(self.stage, self.archive)
        (self.stage / 'new-file').write_text('must also be packaged')
        with self.assertRaisesRegex(ValueError, 'missing release entries'):
            release_zip.verify(self.archive, self.stage)

    def test_extra_path_rejected(self):
        release_zip.create(self.stage, self.archive)
        with zipfile.ZipFile(self.archive, 'a') as archive:
            archive.writestr('../outside', 'not a release entry')
        with self.assertRaisesRegex(ValueError, 'Unexpected ZIP entry'):
            release_zip.verify(self.archive, self.stage)

    def test_no_output_inside_source(self):
        with self.assertRaisesRegex(ValueError, 'outside the release directory'):
            release_zip.create(self.stage, self.stage / 'recursive.zip')

    def test_release_rejects_symbolic_links(self):
        link = self.stage / 'linked-file'
        link.symlink_to(self.stage / '.hidden')
        with self.assertRaisesRegex(ValueError, 'symbolic link'):
            release_zip.create(self.stage, self.archive)


if __name__ == '__main__':
    unittest.main()

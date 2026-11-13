from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPT = Path(__file__).with_name('stage-toolchains.sh')


class SystemLLVMTargetTests(unittest.TestCase):
    def test_target_overlays_preserve_sdk_and_staged_runtimes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, output = root / 'SDK with spaces', root / 'release output'
            originals = {}
            for arch in ('amd64', 'arm64'):
                for name, macro in [('Targets', 'LLVM_TARGET'), ('AsmParsers', 'LLVM_ASM_PARSER'),
                                    ('AsmPrinters', 'LLVM_ASM_PRINTER')]:
                    path = source / f'linux/{arch}/crosscompile/clang/include/llvm/Config/{name}.def'
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(f'{macro}(X86)\n{macro}(AArch64)\n{macro}(Xtensa)\n')
                    originals[path] = path.read_bytes()
            retained = {}
            for name in ('darwin/arm64/bin/clang', 'linux/amd64/lib/libLLVM.so.22.1',
                         'payloads/linux/arm64/clang-esp.tar.xz', 'LLGO-ESP-STAGE.json'):
                path = output / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(name.encode())
                retained[path] = path.read_bytes()
            for _ in range(2):
                subprocess.run(['bash', str(SCRIPT), str(source), str(output)], check=True, capture_output=True)
                self.assertFalse((output / 'system-linux/stale').exists())
                for path, content in originals.items():
                    self.assertEqual(path.read_bytes(), content)
                for path, content in retained.items():
                    self.assertEqual(path.read_bytes(), content)
                for arch in ('amd64', 'arm64'):
                    for name in ('Targets', 'AsmParsers', 'AsmPrinters'):
                        definition = output / f'system-linux/{arch}/include/llvm/Config/{name}.def'
                        self.assertNotIn('(Xtensa)', definition.read_text())
                        self.assertIn('(AArch64)', definition.read_text())
                (output / 'system-linux/stale').touch()

    def test_source_and_staging_must_not_overlap(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / 'SDK'
            source.mkdir()
            for output in (source, source / 'runtime', source.parent):
                result = subprocess.run(['bash', str(SCRIPT), str(source), str(output)], capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('build sysroot', result.stderr)
                self.assertTrue(source.is_dir())


if __name__ == '__main__':
    unittest.main()

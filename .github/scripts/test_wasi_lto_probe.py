"""The WASI LTO probe must verify fresh backend output without changing the ABI."""

from pathlib import Path
import runpy
import shlex
import sys
import tempfile
import unittest
from unittest import mock


class WasiltoProbeTests(unittest.TestCase):
    def setUp(self):
        dev = Path(__file__).resolve().parents[2] / "dev"
        with mock.patch.object(sys, "path", [str(dev), *sys.path]):
            self.build = runpy.run_path(str(dev / "test_wasm_wasi_lto.py"))["build"]

    def backend(self, command, *, env, **kwargs):
        flags = shlex.split(env["LDFLAGS"])
        self.assertIn("-Wl,--export=probe", flags)
        if "-lto=thin" in command:
            self.assertFalse(any("--lto-obj-path=" in flag for flag in flags))
            cache = Path(next(flag.split("=", 1)[1] for flag in flags
                              if flag.startswith("-Wl,--thinlto-cache-dir=")))
            self.assertEqual(list(cache.iterdir()), [])
            self.caches.append(cache)
            if self.output is not None:
                (cache / "llvmcache-probe").write_bytes(self.output)
        else:
            self.assertIn("-Wl,--save-temps", flags)
            obj = Path(next(flag.split("=", 1)[1] for flag in flags
                            if flag.startswith("-Wl,--lto-obj-path=")))
            obj.write_bytes(self.output)
            module = Path(command[command.index("-o") + 1])
            Path(str(module) + ".0.0.preopt.bc").write_bytes(b"BC\xc0\xde")

    def test_fresh_backend_objects_for_each_build(self):
        self.output = b"\0asm\1\0\0\0"
        self.caches = []
        with tempfile.TemporaryDirectory(prefix="wasi lto ") as temporary:
            directory = Path(temporary)
            with mock.patch("subprocess.run", side_effect=self.backend):
                for mode in ("thin", "thin", "full"):
                    self.build({"LDFLAGS": "-Wl,--export=probe"}, directory,
                               mode, directory / "fixture")
        self.assertEqual(len(set(self.caches)), 2)

    def test_missing_or_invalid_backend_output_fails(self):
        for self.output in (None, b"", b"not a Wasm object"):
            with self.subTest(output=self.output), tempfile.TemporaryDirectory() as temporary:
                self.caches = []
                directory = Path(temporary)
                with mock.patch("subprocess.run", side_effect=self.backend):
                    with self.assertRaisesRegex(SystemExit, "LTO backend object"):
                        self.build({"LDFLAGS": "-Wl,--export=probe"}, directory,
                                   "thin", directory / "fixture")


if __name__ == "__main__":
    unittest.main()

import pathlib
import re
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]


class PixiDevEnvTests(unittest.TestCase):
    def test_single_dev_env_manifest(self):
        self.assertTrue((ROOT / "dev" / "pixi.toml").is_file())
        self.assertTrue((ROOT / "dev" / "pixi.lock").is_file())
        self.assertFalse((ROOT / "dev" / "mise.toml").exists())
        self.assertFalse((ROOT / "dev" / "mise.lock").exists())

    def test_pixi_go_version_matches_go_version_file(self):
        go_version = (ROOT / ".go-version").read_text(encoding="utf-8").strip()
        m = re.match(r"^(\d+)\.(\d+)(?:\.\d+)?$", go_version)
        self.assertIsNotNone(m, f"unexpected .go-version: {go_version!r}")
        major, minor = int(m.group(1)), int(m.group(2))
        expected = f'go = ">={major}.{minor},<{major}.{minor + 1}"'
        pixi_toml = (ROOT / "dev" / "pixi.toml").read_text(encoding="utf-8")
        self.assertIn(expected, pixi_toml)


if __name__ == "__main__":
    unittest.main()

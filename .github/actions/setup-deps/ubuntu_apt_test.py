#!/usr/bin/env python3

import pathlib
import tempfile
import unittest

from ubuntu_apt import configure


class UbuntuAPTTest(unittest.TestCase):
    def test_runner_mirror_lists(self):
        for arch, archive, security in (
            ("amd64", "https://azure.archive.ubuntu.com/ubuntu/\tpriority:1\n"
             "https://archive.ubuntu.com/ubuntu/\tpriority:2\n",
             "https://azure.archive.ubuntu.com/ubuntu/\tpriority:1\n"
             "https://security.ubuntu.com/ubuntu/\tpriority:2\n"),
            ("i386", "https://azure.archive.ubuntu.com/ubuntu/\tpriority:1\n"
             "https://archive.ubuntu.com/ubuntu/\tpriority:2\n",
             "https://azure.archive.ubuntu.com/ubuntu/\tpriority:1\n"
             "https://security.ubuntu.com/ubuntu/\tpriority:2\n"),
            ("arm64", "https://ports.ubuntu.com/ubuntu-ports/\tpriority:1\n",
             "https://ports.ubuntu.com/ubuntu-ports/\tpriority:1\n"),
        ):
            with self.subTest(arch=arch), tempfile.TemporaryDirectory() as tmp:
                root = pathlib.Path(tmp)
                (root / "apt.conf.d").mkdir()
                for name in ("apt-mirrors.txt", "apt-mirrors-security.txt"):
                    (root / name).write_text(
                        "https:/us.archive.ubuntu.com/ubuntu\n"
                        "https://mirrors.edge.kernel.org/ubuntu\n")
                sources = root / "ubuntu.sources"
                source_text = "URIs: mirror+file:/etc/apt/apt-mirrors.txt\nSigned-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\n"
                sources.write_text(source_text)
                third_party = root / "custom-mirrors.txt"
                third_party.write_text("https://apt.llvm.org/noble/\n")
                configure(root, arch)
                configure(root, arch)
                self.assertEqual((root / "apt-mirrors.txt").read_text(), archive)
                self.assertEqual((root / "apt-mirrors-security.txt").read_text(), security)
                self.assertEqual(sources.read_text(), source_text)
                self.assertEqual(third_party.read_text(), "https://apt.llvm.org/noble/\n")

    def test_direct_sources_need_no_mirror_lists(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "apt.conf.d").mkdir()
            configure(root, "amd64")
            self.assertFalse((root / "apt-mirrors.txt").exists())
            self.assertFalse((root / "apt-mirrors-security.txt").exists())
            self.assertTrue((root / "apt.conf.d" / "80-llgo-downloads").is_file())


if __name__ == "__main__":
    unittest.main()

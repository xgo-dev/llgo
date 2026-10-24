#!/usr/bin/env python3
"""Overlay the pinned BDWGC port with Windows collector read-fault handling."""

import json
from pathlib import Path
import shutil
import sys


def prepare(vcpkg, overlay):
    source = Path(vcpkg) / "ports" / "bdwgc"
    metadata = json.loads((source / "vcpkg.json").read_text())
    if metadata.get("version") != "8.2.12":
        raise ValueError("review the BDWGC fault-guard patch before upgrading the pinned port")
    port = (source / "portfile.cmake").read_text()
    anchor = "    HEAD_REF master\n)"
    if port.count(anchor) != 1:
        raise ValueError("unrecognized pinned BDWGC source declaration")
    port = port.replace(anchor, '    HEAD_REF master\n'
                        '    PATCHES "${CMAKE_CURRENT_LIST_DIR}/llgo-windows-fault-guard.patch"\n)')
    destination = Path(overlay) / "bdwgc"
    shutil.copytree(source, destination, dirs_exist_ok=True)
    (destination / "portfile.cmake").write_text(port)
    shutil.copyfile(Path(__file__).with_name("bdwgc-windows-fault-guard.patch"),
                    destination / "llgo-windows-fault-guard.patch")


if __name__ == "__main__":
    prepare(*sys.argv[1:])

#!/usr/bin/env python3
"""Wrap a linked ESP32-S31 baremetal ELF into a flash image esp-emu boots.

Mirrors buildESPEmuFlashImage in cl/embed_s31_test.go: an ESP image header
(chip image id 0x0020) at flash offset 0x2000 followed by every allocated
PROGBITS-style section, padded to the checksum position, then the XOR
checksum and the SHA-256 of the image; the rest of the 4 MiB flash is
zero-filled. Stdlib only.

Usage: build_flash.py app.elf flash.bin
"""
import hashlib
import struct
import sys

FLASH_OFF = 0x2000
IMAGE_ID = 0x0020  # esp32s31 chip id in the ESP image header
SHT_PROGBITS, SHT_INIT_ARRAY, SHT_FINI_ARRAY, SHT_PREINIT_ARRAY = 1, 14, 15, 16
SHT_NOBITS = 8
SHF_ALLOC = 0x2


def main():
    elf_path, out_path = sys.argv[1], sys.argv[2]
    data = open(elf_path, "rb").read()
    if data[:4] != b"\x7fELF" or data[4] != 1:
        sys.exit("ELF32 required")

    (entry,) = struct.unpack_from("<I", data, 0x18)
    (shoff,) = struct.unpack_from("<I", data, 0x20)
    shentsize, shnum = struct.unpack_from("<HH", data, 0x2E)

    segs = []
    for i in range(shnum):
        off = shoff + i * shentsize
        _name, stype, flags, addr, offset, size = struct.unpack_from("<IIIIII", data, off)
        if size == 0 or not flags & SHF_ALLOC:
            continue
        if stype == SHT_NOBITS:  # occupies RAM but is not flashed
            continue
        if stype not in (SHT_PROGBITS, SHT_INIT_ARRAY, SHT_FINI_ARRAY, SHT_PREINIT_ARRAY):
            continue
        blob = bytearray(data[offset:offset + size])
        while len(blob) % 4:
            blob.append(0)
        segs.append((addr, bytes(blob)))
    segs.sort()

    csum = 0xEF
    for _, blob in segs:
        for b in blob:
            csum ^= b

    img = bytearray(b"\xE9" + bytes([len(segs), 2, 0x1F]))
    img += struct.pack("<I", entry)
    img += b"\xEE\x00\x00\x00"
    img += struct.pack("<H", IMAGE_ID)
    img += b"\x00"
    img += b"\x00" * 8
    img += b"\x01"
    for addr, blob in segs:
        img += struct.pack("<I", addr) + struct.pack("<I", len(blob)) + blob
    while len(img) % 16 != 15:
        img.append(0)
    img.append(csum)
    img += hashlib.sha256(img).digest()

    flash = bytearray(4 * 1024 * 1024)
    flash[FLASH_OFF:FLASH_OFF + len(img)] = img
    with open(out_path, "wb") as f:
        f.write(flash)
    print(f"flash image: {out_path} ({len(img)} bytes, {len(segs)} segments, entry 0x{entry:08x})")


main()

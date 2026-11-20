// Portions of this file are derived from tinygo/builder/esp.go.
// Copyright (c) 2018-2025 The TinyGo Authors. All rights reserved.
// See ../../LICENSES/TinyGo-BSD-3-Clause.txt for license terms.

package firmware

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strings"
)

type espImageSegment struct {
	addr uint32
	data []byte
}

// makeESPFirmareImage converts an input ELF file to an image file for an ESP32 or
// ESP8266 chip. This is a special purpose image format just for the ESP chip
// family, and is parsed by the on-chip mask ROM bootloader.
//
// The following documentation has been used:
// https://github.com/espressif/esptool/wiki/Firmware-Image-Format
// https://github.com/espressif/esp-idf/blob/8fbb63c2a701c22ccf4ce249f43aded73e134a34/components/bootloader_support/include/esp_image_format.h#L58
// https://github.com/espressif/esptool/blob/master/esptool.py
func makeESPFirmareImage(infile, outfile, format string) error {
	if strings.Contains(format, "-dboot") {
		return makeDirectBootImage(infile, outfile)
	}
	inf, err := elf.Open(infile)
	if err != nil {
		return err
	}
	defer inf.Close()

	// Load all segments to be written to the image. These are actually ELF
	// sections, not true ELF segments (similar to how esptool does it).
	var segments []*espImageSegment
	for _, section := range inf.Sections {
		if section.Size == 0 || section.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		switch section.Type {
		case elf.SHT_PROGBITS, elf.SHT_INIT_ARRAY, elf.SHT_PREINIT_ARRAY, elf.SHT_FINI_ARRAY:
			// Keep allocatable data sections that may contain startup tables.
			// With some linker layouts, .rodata can be emitted as INIT/PREINIT/FINI_ARRAY
			// type when constructor arrays are merged into it.
		default:
			continue
		}
		data, err := section.Data()
		if err != nil {
			return fmt.Errorf("failed to read section data: %w", err)
		}
		for len(data)%4 != 0 {
			// Align segment to 4 bytes.
			data = append(data, 0)
		}
		if uint64(uint32(section.Addr)) != section.Addr {
			return fmt.Errorf("section address too big: 0x%x", section.Addr)
		}
		segments = append(segments, &espImageSegment{
			addr: uint32(section.Addr),
			data: data,
		})
	}

	// Sort the segments by address. This is what esptool does too.
	sort.SliceStable(segments, func(i, j int) bool { return segments[i].addr < segments[j].addr })

	// Calculate checksum over the segment data. This is used in the image
	// footer.
	checksum := uint8(0xef)
	for _, segment := range segments {
		for _, b := range segment.data {
			checksum ^= b
		}
	}

	// Write first to an in-memory buffer, primarily so that we can easily
	// calculate a hash over the entire image.
	// An added benefit is that we don't need to check for errors all the time.
	outf := &bytes.Buffer{}

	// Separate esp32 and esp32-img. The -img suffix indicates we should make an
	// image, not just a binary to be flashed at 0x1000 for example.
	chip := format
	makeImage := false
	if strings.HasSuffix(format, "-img") {
		makeImage = true
		chip = format[:len(format)-len("-img")]
	}

	if makeImage {
		// For QEMU emulation, we need to place the image at the correct offset.
		// ESP32 (Xtensa): bootloader starts at 0x1000, so we pad 4KB of zeros.
		// ESP32-C3 (RISC-V): bootloader starts at 0x0, so no padding needed.
		// ESP32-S31 (RISC-V): bootloader starts at 0x2000 (the first two flash
		// sectors are reserved), so we pad 8KB of zeros.
		if chip == "esp32" {
			outf.Write(make([]byte, 4096))
		} else if chip == "esp32s31" {
			outf.Write(make([]byte, 8192))
		}
	}

	// Chip IDs. Source:
	// https://github.com/espressif/esp-idf/blob/master/components/bootloader_support/include/esp_app_format.h
	chip_id := map[string]uint16{
		"esp32":    0x0000,
		"esp32c3":  0x0005,
		"esp32s31": 0x0020,
	}[chip]

	// Image header.
	switch chip {
	case "esp32", "esp32c3", "esp32s31":
		// Header format:
		// https://github.com/espressif/esp-idf/blob/v4.3/components/bootloader_support/include/esp_app_format.h#L71
		// Note: not adding a SHA256 hash as the binary is modified by
		// esptool.py while flashing and therefore the hash won't be valid
		// anymore.
		binary.Write(outf, binary.LittleEndian, struct {
			magic          uint8
			segment_count  uint8
			spi_mode       uint8
			spi_speed_size uint8
			entry_addr     uint32
			wp_pin         uint8
			spi_pin_drv    [3]uint8
			chip_id        uint16
			min_chip_rev   uint8
			reserved       [8]uint8
			hash_appended  bool
		}{
			magic:          0xE9,
			segment_count:  byte(len(segments)),
			spi_mode:       2,    // ESP_IMAGE_SPI_MODE_DIO
			spi_speed_size: 0x1f, // ESP_IMAGE_SPI_SPEED_80M, ESP_IMAGE_FLASH_SIZE_2MB
			entry_addr:     uint32(inf.Entry),
			wp_pin:         0xEE, // disable WP pin
			chip_id:        chip_id,
			hash_appended:  true, // add a SHA256 hash
		})
	case "esp8266":
		// Header format:
		// https://github.com/espressif/esptool/wiki/Firmware-Image-Format
		// Basically a truncated version of the ESP32 header.
		binary.Write(outf, binary.LittleEndian, struct {
			magic          uint8
			segment_count  uint8
			spi_mode       uint8
			spi_speed_size uint8
			entry_addr     uint32
		}{
			magic:          0xE9,
			segment_count:  byte(len(segments)),
			spi_mode:       0,    // irrelevant, replaced by esptool when flashing
			spi_speed_size: 0x20, // spi_speed, spi_size: replaced by esptool when flashing
			entry_addr:     uint32(inf.Entry),
		})
	default:
		return fmt.Errorf("builder: unknown binary format %#v, expected esp32 or esp8266", format)
	}

	// Write all segments to the image.
	// https://github.com/espressif/esptool/wiki/Firmware-Image-Format#segment
	for _, segment := range segments {
		binary.Write(outf, binary.LittleEndian, struct {
			addr   uint32
			length uint32
		}{
			addr:   segment.addr,
			length: uint32(len(segment.data)),
		})
		outf.Write(segment.data)
	}

	// Footer, including checksum.
	// The entire image size must be a multiple of 16, so pad the image to one
	// byte less than that before writing the checksum.
	outf.Write(make([]byte, 15-outf.Len()%16))
	outf.WriteByte(checksum)

	if chip != "esp8266" {
		// SHA256 hash (to protect against image corruption, not for security).
		hash := sha256.Sum256(outf.Bytes())
		outf.Write(hash[:])
	}

	// QEMU (or more precisely, qemu-system-xtensa from Espressif) expects the
	// image to be a certain size.
	if makeImage {
		// Use a default image size of 4MB.
		grow := 4096*1024 - outf.Len()
		if grow > 0 {
			outf.Write(make([]byte, grow))
		}
	}

	// Write the image to the output file.
	return os.WriteFile(outfile, outf.Bytes(), 0666)
}

// makeDirectBootImage flattens an ELF into a raw flash image for the
// ESP32-S31 direct-boot path. The mask ROM maps the whole flash 1:1 into the
// XIP window at 0x40000000 and jumps into it, so every byte must sit at
// exactly its load address: flash-resident sections at their identity-mapped
// offset, and writable sections at their LMA (the linker script places them
// behind the read-only image for the boot-time copy into SRAM). Sections with
// no file contents (.bss, .stack) start out zero and are not written.
func makeDirectBootImage(infile, outfile string) error {
	const (
		flashSize   = 4 * 1024 * 1024
		xipWindow   = 0x40000000
		erasedFlash = 0xFF
	)

	inf, err := elf.Open(infile)
	if err != nil {
		return err
	}
	defer inf.Close()

	// The load addresses come from the program headers: a section's bytes
	// live at PAddr + (Addr - VAddr) of the PT_LOAD segment carrying them.
	type loadSegment struct {
		vaddr, paddr, filesz uint32
	}
	var loads []loadSegment
	for _, prog := range inf.Progs {
		if prog.Type == elf.PT_LOAD && prog.Filesz > 0 {
			loads = append(loads, loadSegment{
				vaddr:  uint32(prog.Vaddr),
				paddr:  uint32(prog.Paddr),
				filesz: uint32(prog.Filesz),
			})
		}
	}

	flash := make([]byte, flashSize)
	for i := range flash {
		flash[i] = erasedFlash
	}
	written := make([][2]int, 0, 8)
	for _, section := range inf.Sections {
		if section.Size == 0 || section.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		switch section.Type {
		case elf.SHT_PROGBITS, elf.SHT_INIT_ARRAY, elf.SHT_PREINIT_ARRAY, elf.SHT_FINI_ARRAY:
		default:
			continue
		}
		data, err := section.Data()
		if err != nil {
			return fmt.Errorf("failed to read section %s: %w", section.Name, err)
		}
		if len(data) == 0 {
			continue
		}
		addr := uint32(section.Addr)
		off := -1
		for _, load := range loads {
			if addr >= load.vaddr && addr < load.vaddr+load.filesz {
				off = int(load.paddr + (addr - load.vaddr))
				break
			}
		}
		if off < 0 {
			return fmt.Errorf("section %s (0x%x) is not covered by a loadable segment", section.Name, addr)
		}
		if off >= xipWindow {
			// A flash-resident section whose load address stayed equal to its
			// virtual address is identity-mapped by the ROM.
			off -= xipWindow
		}
		if off < 0 || off+len(data) > flashSize {
			return fmt.Errorf("section %s (load address 0x%x, %d bytes) does not fit in the %d MiB flash image", section.Name, addr, len(data), flashSize/(1024*1024))
		}
		for _, w := range written {
			if off < w[1] && w[0] < off+len(data) {
				return fmt.Errorf("section %s (flash offset 0x%x) overlaps a section written at 0x%x", section.Name, off, w[0])
			}
		}
		copy(flash[off:], data)
		written = append(written, [2]int{off, off + len(data)})
	}

	return os.WriteFile(outfile, flash, 0666)
}

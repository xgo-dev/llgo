/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pclnpost

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"fmt"
)

// compactCarrier removes the unused file-backed suffix of an isolated entry
// carrier while preserving its original virtual address range. Mach-O images
// that are not embedded LTO executables deliberately keep the carrier in
// __DATA and retain their physical layout after the logical table rewrite.
// Only these deliberately constrained LLGo layouts are accepted; unfamiliar
// shapes fail before the caller publishes any bytes. raw must be an owned
// staging buffer and may be modified even when compaction returns an error.
func compactCarrier(raw []byte, info *binaryInfo, entryUsed uint64) ([]byte, uint64, error) {
	if entryUsed > info.entryVMSize {
		return nil, 0, fmt.Errorf("compact size entry=%#x/%#x", entryUsed, info.entryVMSize)
	}
	switch info.format {
	case ExternalFormatMachO:
		return compactMachO(raw, info, entryUsed)
	case ExternalFormatELF:
		return compactELF(raw, info, entryUsed)
	default:
		return nil, 0, fmt.Errorf("unsupported binary format %q", info.format)
	}
}

type machoSectionLayout struct {
	name, segment string
	header        uint64
	addr, size    uint64
	offset        uint64
	reloff        uint64
	nreloc        uint32
}

type machoSegmentLayout struct {
	name              string
	header            uint64
	vmaddr, vmsize    uint64
	fileoff, filesize uint64
	sections          []machoSectionLayout
}

type machoOffsetField struct {
	pos   uint64
	width uint8
	label string
}

type machoLayout struct {
	segments []machoSegmentLayout
	offsets  []machoOffsetField
}

func compactMachO(raw []byte, info *binaryInfo, entryUsed uint64) ([]byte, uint64, error) {
	layout, err := parseMachOLayout(raw)
	if err != nil {
		return nil, 0, err
	}
	var carrier *machoSegmentLayout
	var entry *machoSectionLayout
	for si := range layout.segments {
		seg := &layout.segments[si]
		for sj := range seg.sections {
			sec := &seg.sections[sj]
			if sec.name == "__llgo_fie" {
				entry, carrier = sec, seg
			}
		}
	}
	if carrier == nil || entry == nil {
		return nil, 0, fmt.Errorf("missing Mach-O funcinfo carrier")
	}
	oldEnd, ok := checkedEnd(carrier.fileoff, carrier.filesize, uint64(len(raw)))
	if carrier.filesize == 0 || !ok {
		return nil, 0, fmt.Errorf("Mach-O %s file range [%#x,+%#x) is invalid", carrier.name, carrier.fileoff, carrier.filesize)
	}
	if !rangeContained(carrier.fileoff, oldEnd, entry.offset, entry.size) {
		return nil, 0, fmt.Errorf("Mach-O entry carrier is outside %s", carrier.name)
	}
	if carrier.name == "__DATA" {
		return raw, 0, nil
	}
	if carrier.name != "__LLGO" {
		return nil, 0, fmt.Errorf("Mach-O funcinfo carrier is in unsupported segment %s", carrier.name)
	}
	for i := range carrier.sections {
		sec := &carrier.sections[i]
		if sec.name != "__llgo_fie" && sec.size != 0 {
			return nil, 0, fmt.Errorf("Mach-O __LLGO contains unexpected section %s", sec.name)
		}
	}
	for i := range layout.segments {
		seg := &layout.segments[i]
		if seg.header != carrier.header && seg.filesize != 0 && seg.fileoff >= oldEnd && seg.name != "__LINKEDIT" {
			return nil, 0, fmt.Errorf("Mach-O segment %s follows __LLGO", seg.name)
		}
	}
	lastUsed, ok := checkedEnd(entry.offset, entryUsed, oldEnd)
	if !ok || lastUsed < carrier.fileoff {
		return nil, 0, fmt.Errorf("Mach-O compact payload %#x is outside __LLGO [%#x,%#x)", lastUsed, carrier.fileoff, oldEnd)
	}
	page := machoPageSize(raw)
	newFilesz := alignUp(lastUsed-carrier.fileoff, page)
	if newFilesz > carrier.filesize {
		return nil, 0, fmt.Errorf("Mach-O compact __LLGO size %#x exceeds %#x", newFilesz, carrier.filesize)
	}
	cutStart := carrier.fileoff + newFilesz
	removed := oldEnd - cutStart

	out := raw
	binary.LittleEndian.PutUint64(out[entry.header+40:], entryUsed)
	if removed == 0 {
		return out, 0, nil
	}
	if err := patchMachOFileOffsets(out, layout, carrier.header, newFilesz, cutStart, oldEnd); err != nil {
		return nil, 0, err
	}
	copy(out[cutStart:], out[oldEnd:])
	out = out[:uint64(len(out))-removed]
	if _, err := macho.NewFile(bytes.NewReader(out)); err != nil {
		return nil, 0, fmt.Errorf("verify compact Mach-O: %w", err)
	}
	return out, removed, nil
}

func parseMachOLayout(raw []byte) (machoLayout, error) {
	var layout machoLayout
	if len(raw) < 32 || binary.LittleEndian.Uint32(raw) != uint32(macho.Magic64) {
		return layout, fmt.Errorf("compact Mach-O requires a little-endian 64-bit image")
	}
	ncmds := binary.LittleEndian.Uint32(raw[16:])
	off := uint64(32)
	layout.segments = make([]machoSegmentLayout, 0, 8)
	add32 := func(pos uint64, label string) {
		layout.offsets = append(layout.offsets, machoOffsetField{pos, 4, label})
	}
	add64 := func(pos uint64, label string) {
		layout.offsets = append(layout.offsets, machoOffsetField{pos, 8, label})
	}
	for i := uint32(0); i < ncmds; i++ {
		if off > uint64(len(raw)) || uint64(len(raw))-off < 8 {
			return layout, fmt.Errorf("Mach-O load command %d is truncated", i)
		}
		cmd := binary.LittleEndian.Uint32(raw[off:])
		cmdsz := uint64(binary.LittleEndian.Uint32(raw[off+4:]))
		if cmdsz < 8 || cmdsz > uint64(len(raw))-off {
			return layout, fmt.Errorf("Mach-O load command %d has invalid size %#x", i, cmdsz)
		}
		require := func(size uint64) error {
			if cmdsz < size {
				return fmt.Errorf("Mach-O load command %#x has size %#x, want at least %#x", cmd, cmdsz, size)
			}
			return nil
		}
		switch cmd {
		case 0x19: // LC_SEGMENT_64
			if cmdsz < 72 {
				return layout, fmt.Errorf("truncated LC_SEGMENT_64")
			}
			seg := machoSegmentLayout{
				name:     cString16(raw[off+8 : off+24]),
				header:   off,
				vmaddr:   binary.LittleEndian.Uint64(raw[off+24:]),
				vmsize:   binary.LittleEndian.Uint64(raw[off+32:]),
				fileoff:  binary.LittleEndian.Uint64(raw[off+40:]),
				filesize: binary.LittleEndian.Uint64(raw[off+48:]),
			}
			nsects := binary.LittleEndian.Uint32(raw[off+64:])
			if uint64(nsects) > (cmdsz-72)/80 {
				return layout, fmt.Errorf("Mach-O segment %s has truncated sections", seg.name)
			}
			for j := uint32(0); j < nsects; j++ {
				h := off + 72 + uint64(j)*80
				seg.sections = append(seg.sections, machoSectionLayout{
					name:    cString16(raw[h : h+16]),
					segment: cString16(raw[h+16 : h+32]),
					header:  h,
					addr:    binary.LittleEndian.Uint64(raw[h+32:]),
					size:    binary.LittleEndian.Uint64(raw[h+40:]),
					offset:  uint64(binary.LittleEndian.Uint32(raw[h+48:])),
					reloff:  uint64(binary.LittleEndian.Uint32(raw[h+56:])),
					nreloc:  binary.LittleEndian.Uint32(raw[h+60:]),
				})
			}
			layout.segments = append(layout.segments, seg)
		case 0x2: // LC_SYMTAB
			if err := require(24); err != nil {
				return layout, err
			}
			add32(off+8, "symoff")
			add32(off+16, "stroff")
		case 0xb: // LC_DYSYMTAB
			if err := require(80); err != nil {
				return layout, err
			}
			for _, p := range []uint64{32, 40, 48, 56, 64, 72} {
				add32(off+p, "dysymtab")
			}
		case 0x22, 0x80000022: // LC_DYLD_INFO[_ONLY]
			if err := require(48); err != nil {
				return layout, err
			}
			for _, p := range []uint64{8, 16, 24, 32, 40} {
				add32(off+p, "dyld info")
			}
		case 0x1d, 0x1e, 0x26, 0x29, 0x2b, 0x2e, 0x80000033, 0x80000034:
			if err := require(16); err != nil {
				return layout, err
			}
			add32(off+8, "linkedit data")
		case 0x16: // LC_TWOLEVEL_HINTS
			if err := require(16); err != nil {
				return layout, err
			}
			add32(off+8, "twolevel hints")
		case 0x31: // LC_NOTE
			if err := require(40); err != nil {
				return layout, err
			}
			add64(off+24, "note")
		case 0x21, 0x2c: // LC_ENCRYPTION_INFO[_64]
			if err := require(20); err != nil {
				return layout, err
			}
			if binary.LittleEndian.Uint32(raw[off+16:]) != 0 {
				return layout, fmt.Errorf("cannot compact encrypted Mach-O")
			}
			add32(off+8, "encryption info")
		case 0x35, 0x80000035: // LC_FILESET_ENTRY
			return layout, fmt.Errorf("cannot compact Mach-O fileset")
		default:
			if !machoLoadCommandHasNoFileOffset(cmd) {
				return layout, fmt.Errorf("unsupported Mach-O load command %#x", cmd)
			}
		}
		off += cmdsz
	}
	return layout, nil
}

func patchMachOFileOffsets(raw []byte, layout machoLayout, carrierHeader, newFilesz, cutStart, cutEnd uint64) error {
	delta := cutEnd - cutStart
	shift32 := func(pos uint64, label string) error {
		if pos > uint64(len(raw)) || uint64(len(raw))-pos < 4 {
			return fmt.Errorf("Mach-O %s offset field is truncated", label)
		}
		v := uint64(binary.LittleEndian.Uint32(raw[pos:]))
		if v == 0 {
			return nil
		}
		if v >= cutEnd {
			binary.LittleEndian.PutUint32(raw[pos:], uint32(v-delta))
			return nil
		}
		if v >= cutStart {
			return fmt.Errorf("Mach-O %s offset %#x falls in compacted range", label, v)
		}
		return nil
	}
	shift64 := func(pos uint64, label string) error {
		if pos > uint64(len(raw)) || uint64(len(raw))-pos < 8 {
			return fmt.Errorf("Mach-O %s offset field is truncated", label)
		}
		v := binary.LittleEndian.Uint64(raw[pos:])
		if v == 0 {
			return nil
		}
		if v >= cutEnd {
			binary.LittleEndian.PutUint64(raw[pos:], v-delta)
			return nil
		}
		if v >= cutStart {
			return fmt.Errorf("Mach-O %s offset %#x falls in compacted range", label, v)
		}
		return nil
	}
	for _, seg := range layout.segments {
		if seg.header == carrierHeader {
			binary.LittleEndian.PutUint64(raw[seg.header+48:], newFilesz)
		} else if err := shift64(seg.header+40, seg.name+" fileoff"); err != nil {
			return err
		}
		for _, sec := range seg.sections {
			if sec.name != "__llgo_fie" {
				if err := shift32(sec.header+48, sec.segment+","+sec.name+" offset"); err != nil {
					return err
				}
			}
			if sec.nreloc != 0 {
				if err := shift32(sec.header+56, sec.segment+","+sec.name+" reloff"); err != nil {
					return err
				}
			}
		}
	}

	for _, field := range layout.offsets {
		var err error
		if field.width == 8 {
			err = shift64(field.pos, field.label)
		} else {
			err = shift32(field.pos, field.label)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func machoLoadCommandHasNoFileOffset(cmd uint32) bool {
	// LC_REQ_DYLD is an attribute bit, not part of the command layout.
	switch cmd &^ 0x80000000 {
	case 0x1, // LC_SEGMENT (32-bit, rejected by parser)
		0x3, 0x4, 0x5, 0x6, // thread/unixthread/loadfvmlib/idfvmlib
		0xc, 0xd, 0xe, 0xf, // dylib/id/load dylinker/prebound dylib
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
		0x17, 0x18, 0x19, 0x1b, 0x1c,
		0x20, 0x23, 0x24, 0x25, 0x27, 0x28,
		0x2a, 0x2d, 0x2f, 0x30, 0x32:
		return true
	}
	return false
}

func machoPageSize(raw []byte) uint64 {
	// Current LLGo Darwin targets use 16 KiB pages on arm64 and 4 KiB on
	// amd64. Unknown Mach-O CPU types deliberately take the conservative
	// 4 KiB fallback and still pass the staged-image verifier.
	const cpuTypeARM64 = uint32(0x0100000c)
	if len(raw) >= 8 && binary.LittleEndian.Uint32(raw[4:]) == cpuTypeARM64 {
		return 0x4000
	}
	return 0x1000
}

func cString16(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

type elfSectionLayout struct {
	name                string
	header              uint64
	typ                 uint32
	flags, addr, offset uint64
	size, align         uint64
}

type elfProgramLayout struct {
	header                              uint64
	typ, flags                          uint32
	offset, vaddr, filesz, memsz, align uint64
}

func compactELF(raw []byte, info *binaryInfo, entryUsed uint64) ([]byte, uint64, error) {
	sections, programs, shoff, err := parseELFLayout(raw)
	if err != nil {
		return nil, 0, err
	}
	var entry *elfSectionLayout
	for i := range sections {
		if sections[i].name == "llgo_funcinfo_entry" {
			entry = &sections[i]
		}
	}
	if entry == nil {
		return nil, 0, fmt.Errorf("missing ELF entry carrier")
	}
	var carrier *elfProgramLayout
	for i := range programs {
		p := &programs[i]
		if p.typ != uint32(elf.PT_LOAD) || p.filesz == 0 {
			continue
		}
		end := p.offset + p.filesz // parseELFLayout already validated this range.
		if rangeContained(p.offset, end, entry.offset, entry.size) {
			carrier = p
			break
		}
	}
	if carrier == nil {
		return nil, 0, fmt.Errorf("ELF funcinfo carrier is not in a PT_LOAD")
	}
	oldEnd := carrier.offset + carrier.filesz
	lastUsed, ok := checkedEnd(entry.offset, entryUsed, oldEnd)
	if !ok || lastUsed < carrier.offset {
		return nil, 0, fmt.Errorf("ELF compact payload %#x is outside PT_LOAD [%#x,%#x)", lastUsed, carrier.offset, oldEnd)
	}
	for _, sec := range sections {
		if sec.typ == uint32(elf.SHT_NOBITS) || sec.size == 0 || sec.name == entry.name {
			continue
		}
		secEnd := sec.offset + sec.size // parseELFLayout already validated this range.
		if sec.flags&uint64(elf.SHF_ALLOC) != 0 && sec.offset < oldEnd && secEnd > lastUsed {
			return nil, 0, fmt.Errorf("ELF allocated section %s follows compact payload in carrier PT_LOAD", sec.name)
		}
	}
	carrierMemEnd, ok := checkedEnd(carrier.vaddr, carrier.memsz, ^uint64(0))
	entryMemEnd, entryOK := checkedEnd(info.entryVMAddr, info.entryVMSize, ^uint64(0))
	if !ok || !entryOK || carrierMemEnd < entryMemEnd {
		return nil, 0, fmt.Errorf("ELF carrier virtual tail is not covered by PT_LOAD memory")
	}
	// The linker script makes the carrier the final file-backed part of the
	// final PT_LOAD. Program segments therefore never need to move; only
	// non-allocated sections and the section-header table follow it.
	for _, p := range programs {
		if p.header == carrier.header || p.offset == 0 {
			continue
		}
		if p.offset >= oldEnd {
			return nil, 0, fmt.Errorf("ELF program header type %d follows the funcinfo carrier", p.typ)
		}
	}
	cutAlign := uint64(8) // ELF64 section headers and symbol tables.
	for _, sec := range sections {
		if sec.offset < oldEnd || sec.align <= cutAlign {
			continue
		}
		if sec.align&(sec.align-1) != 0 {
			return nil, 0, fmt.Errorf("ELF section %s has invalid alignment %#x", sec.name, sec.align)
		}
		cutAlign = sec.align
	}
	removed := alignDown(oldEnd-lastUsed, cutAlign)
	cutStart := oldEnd - removed
	out := raw
	binary.LittleEndian.PutUint64(out[entry.header+32:], entryUsed)
	if removed == 0 {
		return out, 0, nil
	}
	binary.LittleEndian.PutUint64(out[carrier.header+32:], carrier.filesz-removed)
	for _, p := range programs {
		pEnd := p.offset + p.filesz
		if p.header != carrier.header && p.filesz != 0 && p.offset < oldEnd && pEnd > cutStart {
			return nil, 0, fmt.Errorf("ELF program header type %d overlaps compacted range", p.typ)
		}
	}
	for _, sec := range sections {
		if sec.name == entry.name {
			continue
		}
		if sec.offset >= oldEnd {
			binary.LittleEndian.PutUint64(out[sec.header+24:], sec.offset-removed)
		} else if sec.typ == uint32(elf.SHT_NOBITS) && sec.offset >= cutStart {
			binary.LittleEndian.PutUint64(out[sec.header+24:], cutStart)
		} else if sec.typ != uint32(elf.SHT_NOBITS) && sec.size != 0 {
			secEnd := sec.offset + sec.size
			if secEnd <= cutStart {
				continue
			}
			return nil, 0, fmt.Errorf("ELF section %s overlaps compacted range", sec.name)
		}
	}
	if shoff >= oldEnd {
		binary.LittleEndian.PutUint64(out[40:], shoff-removed)
	} else if shoff >= cutStart {
		return nil, 0, fmt.Errorf("ELF section headers overlap compacted range")
	}
	copy(out[cutStart:], out[oldEnd:])
	out = out[:uint64(len(out))-removed]
	if _, err := elf.NewFile(bytes.NewReader(out)); err != nil {
		return nil, 0, fmt.Errorf("verify compact ELF: %w", err)
	}
	return out, removed, nil
}

func parseELFLayout(raw []byte) ([]elfSectionLayout, []elfProgramLayout, uint64, error) {
	if len(raw) < 64 || !bytes.Equal(raw[:4], []byte{0x7f, 'E', 'L', 'F'}) || raw[4] != byte(elf.ELFCLASS64) || raw[5] != byte(elf.ELFDATA2LSB) {
		return nil, nil, 0, fmt.Errorf("compact ELF requires a little-endian ELF64 image")
	}
	phoff := binary.LittleEndian.Uint64(raw[32:])
	shoff := binary.LittleEndian.Uint64(raw[40:])
	phentsz := uint64(binary.LittleEndian.Uint16(raw[54:]))
	phnum := uint64(binary.LittleEndian.Uint16(raw[56:]))
	shentsz := uint64(binary.LittleEndian.Uint16(raw[58:]))
	shnum := uint64(binary.LittleEndian.Uint16(raw[60:]))
	shstrndx := uint64(binary.LittleEndian.Uint16(raw[62:]))
	if phnum == 0 || phentsz < 56 || !tableWithinFile(phoff, phnum, phentsz, uint64(len(raw))) {
		return nil, nil, 0, fmt.Errorf("ELF has no valid program header table")
	}
	if shnum == 0 || shentsz < 64 || shstrndx >= shnum || !tableWithinFile(shoff, shnum, shentsz, uint64(len(raw))) {
		return nil, nil, 0, fmt.Errorf("ELF has no valid section header table")
	}
	shstrHeader := shoff + shstrndx*shentsz
	strOff := binary.LittleEndian.Uint64(raw[shstrHeader+24:])
	strSize := binary.LittleEndian.Uint64(raw[shstrHeader+32:])
	strEnd, ok := checkedEnd(strOff, strSize, uint64(len(raw)))
	if !ok {
		return nil, nil, 0, fmt.Errorf("ELF section-name table is outside file")
	}
	strs := raw[strOff:strEnd]
	nameAt := func(off uint32) string {
		if uint64(off) >= uint64(len(strs)) {
			return ""
		}
		b := strs[off:]
		if i := bytes.IndexByte(b, 0); i >= 0 {
			b = b[:i]
		}
		return string(b)
	}
	sections := make([]elfSectionLayout, 0, shnum)
	for i := uint64(0); i < shnum; i++ {
		h := shoff + i*shentsz
		sec := elfSectionLayout{
			name: nameAt(binary.LittleEndian.Uint32(raw[h:])), header: h,
			typ: binary.LittleEndian.Uint32(raw[h+4:]), flags: binary.LittleEndian.Uint64(raw[h+8:]),
			addr: binary.LittleEndian.Uint64(raw[h+16:]), offset: binary.LittleEndian.Uint64(raw[h+24:]), size: binary.LittleEndian.Uint64(raw[h+32:]),
			align: binary.LittleEndian.Uint64(raw[h+48:]),
		}
		if sec.typ != uint32(elf.SHT_NOBITS) && sec.size != 0 {
			if _, ok := checkedEnd(sec.offset, sec.size, uint64(len(raw))); !ok {
				return nil, nil, 0, fmt.Errorf("ELF section %s has an invalid file range", sec.name)
			}
		}
		sections = append(sections, sec)
	}
	programs := make([]elfProgramLayout, 0, phnum)
	for i := uint64(0); i < phnum; i++ {
		h := phoff + i*phentsz
		program := elfProgramLayout{
			header: h, typ: binary.LittleEndian.Uint32(raw[h:]), flags: binary.LittleEndian.Uint32(raw[h+4:]),
			offset: binary.LittleEndian.Uint64(raw[h+8:]), vaddr: binary.LittleEndian.Uint64(raw[h+16:]),
			filesz: binary.LittleEndian.Uint64(raw[h+32:]), memsz: binary.LittleEndian.Uint64(raw[h+40:]), align: binary.LittleEndian.Uint64(raw[h+48:]),
		}
		if program.filesz != 0 {
			if _, ok := checkedEnd(program.offset, program.filesz, uint64(len(raw))); !ok {
				return nil, nil, 0, fmt.Errorf("ELF program header type %d has an invalid file range", program.typ)
			}
		}
		programs = append(programs, program)
	}
	return sections, programs, shoff, nil
}

func alignUp(v, align uint64) uint64 {
	if align <= 1 {
		return v
	}
	if align&(align-1) != 0 || v > ^uint64(0)-(align-1) {
		return ^uint64(0)
	}
	return (v + align - 1) &^ (align - 1)
}

func alignDown(v, align uint64) uint64 {
	if align <= 1 {
		return v
	}
	return v &^ (align - 1)
}

func checkedEnd(off, size, limit uint64) (uint64, bool) {
	if off > limit || size > limit-off {
		return 0, false
	}
	return off + size, true
}

func tableWithinFile(off, count, entrySize, limit uint64) bool {
	return off <= limit && count <= (limit-off)/entrySize
}

func rangeContained(start, end, off, size uint64) bool {
	rangeEnd, ok := checkedEnd(off, size, end)
	return ok && off >= start && rangeEnd <= end
}

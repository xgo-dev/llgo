// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestMachOVirtualLayout(t *testing.T) {
	const page = uint64(16384)
	for _, tt := range []struct {
		name    string
		order   []int
		gap     uint64
		padding uint64
	}{
		{"virtual padding", []int{0, 1, 2}, 0, 2 * page},
		{"aligned virtual gaps", []int{0, 1, 2}, page, page},
		{"independent virtual order", []int{2, 0, 1}, page, page},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw := machoFixture(t)
			const segment = 32
			const sections = segment + 72
			const linkedit = sections + 3*80
			base := binary.LittleEndian.Uint64(raw[segment+24:])
			next := base
			for _, i := range tt.order {
				p := sections + i*80
				next = align(next, page)
				binary.LittleEndian.PutUint64(raw[p+32:], next)
				next += binary.LittleEndian.Uint64(raw[p+40:]) + tt.gap
			}
			memsz := align(next-base, page) + tt.padding
			binary.LittleEndian.PutUint64(raw[segment+32:], memsz)
			binary.LittleEndian.PutUint64(raw[linkedit+24:], base+memsz)
			before, err := macho.NewFile(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			if before.Segment("__DWARF").Memsz <= before.Segment("__DWARF").Filesz {
				t.Fatal("fixture needs independent file and virtual extents")
			}
			original := bytes.Clone(raw)
			out, err := Compress(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(out) >= len(raw) || !bytes.Equal(raw, original) {
				t.Fatal("compression did not shrink the file or modified the input")
			}
			after, err := macho.NewFile(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			seg := after.Segment("__DWARF")
			for _, s := range after.Sections {
				if s.Seg == "__DWARF" && (s.Addr < seg.Addr || s.Size > seg.Memsz || s.Addr-seg.Addr > seg.Memsz-s.Size) {
					t.Errorf("section %s [%#x, %#x) lies outside segment [%#x, %#x)", s.Name, s.Addr, s.Addr+s.Size, seg.Addr, seg.Addr+seg.Memsz)
				}
			}
			if seg.Addr != before.Segment("__DWARF").Addr || seg.Memsz != memsz {
				t.Error("file compression changed the virtual segment range")
			}
			for i, s := range after.Sections {
				if s.Addr != before.Sections[i].Addr {
					t.Errorf("file compression moved section %s's virtual address", s.Name)
				}
			}
			if !reflect.DeepEqual(dwarfEntries(t, before), dwarfEntries(t, after)) {
				t.Fatal("virtual gaps changed the decoded DWARF")
			}
		})
	}
}

func TestMalformedNativeRanges(t *testing.T) {
	mach := machoFixture(t)
	image := peFixture(t, 64)
	const section = 32 + 72
	const segment = 32 + 72 + 3*80
	const symtab = segment + 72
	const coff = 0x84
	const optional = coff + 20
	const peSections = optional + 240
	u16 := binary.LittleEndian.PutUint16
	u32 := binary.LittleEndian.PutUint32
	u64 := binary.LittleEndian.PutUint64
	tests := []struct {
		name string
		raw  []byte
		edit func([]byte)
	}{
		{"Mach-O command envelope", mach, func(b []byte) { u32(b[20:], uint32(len(b))) }},
		{"Mach-O short command", mach, func(b []byte) { u32(b[36:], 4) }},
		{"Mach-O missing command", mach, func(b []byte) { u32(b[16:], 5) }},
		{"Mach-O extra commands", mach, func(b []byte) { u32(b[16:], 3) }},
		{"Mach-O section count", mach, func(b []byte) { u32(b[32+64:], 99) }},
		{"Mach-O segment extent", mach, func(b []byte) { u64(b[32+48:], uint64(len(b))) }},
		{"Mach-O relocation extent", mach, func(b []byte) { u32(b[section+56:], uint32(len(b))); u32(b[section+60:], 1) }},
		{"Mach-O alignment", mach, func(b []byte) { u32(b[section+52:], 15) }},
		{"Mach-O section before segment", mach, func(b []byte) { u32(b[section+48:], 1024) }},
		{"Mach-O symbol overlaps commands", mach, func(b []byte) { u32(b[symtab+8:], 32) }},
		{"Mach-O invalid symbol name", mach, func(b []byte) { u32(b[binary.LittleEndian.Uint32(b[symtab+8:]):], math.MaxUint32) }},
		{"Mach-O debug symbol", mach, func(b []byte) { p := binary.LittleEndian.Uint32(b[symtab+8:]); b[p+5] = 1 }},
		{"Mach-O duplicate compressed name", mach, func(b []byte) { clear(b[section+2*80 : section+2*80+16]); copy(b[section+2*80:], "__zdebug_info") }},
		{"Mach-O empty virtual segment", mach, func(b []byte) { u64(b[32+32:], 0) }},
		{"Mach-O virtual segment overflow", mach, func(b []byte) { u64(b[32+24:], math.MaxUint64) }},
		{"Mach-O section below virtual segment", mach, func(b []byte) { u64(b[section+2*80+32:], 0) }},
		{"Mach-O section above virtual segment", mach, func(b []byte) { u64(b[section+2*80+32:], math.MaxUint64) }},
		{"Mach-O segment begins inside edit", mach, func(b []byte) { u64(b[segment+40:], uint64(binary.LittleEndian.Uint32(b[section+80+48:]))+1) }},
		{"Mach-O segment ends inside edit", mach, func(b []byte) { u64(b[32+48:], 4096) }},
		{"Mach-O linkedit ends inside edit", mach, func(b []byte) {
			u64(b[segment+40:], uint64(binary.LittleEndian.Uint32(b[section+80+48:]))-1)
			u64(b[segment+48:], 2)
		}},
		{"Mach-O unused string pointer inside edit", mach, func(b []byte) {
			u32(b[symtab+12:], 0)
			u32(b[symtab+16:], binary.LittleEndian.Uint32(b[section+80+48:])+1)
			u32(b[symtab+20:], 0)
		}},
		{"Mach-O relocation pointer inside edit", mach, func(b []byte) { u32(b[section+56:], binary.LittleEndian.Uint32(b[section+80+48:])+1) }},
		{"PE short optional header", image, func(b []byte) { u16(b[coff+16:], 80) }},
		{"PE unsupported header", image, func(b []byte) { u16(b[optional:], 0) }},
		{"PE missing directory header", image, func(b []byte) { u16(b[coff+16:], 100) }},
		{"PE directory count", image, func(b []byte) { u32(b[optional+108:], 17) }},
		{"PE file alignment", image, func(b []byte) { u32(b[optional+36:], 3) }},
		{"PE symbol extent", image, func(b []byte) { u32(b[coff+12:], math.MaxUint32) }},
		{"PE truncated strings", image, func(b []byte) { u32(b[coff+8:], uint32(len(b)-2)) }},
		{"PE string size", image, func(b []byte) { u32(b[binary.LittleEndian.Uint32(b[coff+8:]):], 3) }},
		{"PE bad string offset", image, func(b []byte) { clear(b[peSections+40 : peSections+48]); copy(b[peSections+40:], "/999999") }},
		{"PE section overlaps headers", image, func(b []byte) { u32(b[peSections+20:], 32) }},
		{"PE unrelocated DWARF", image, func(b []byte) { u16(b[peSections+2*40+32:], 1) }},
		{"PE virtual tail", image, func(b []byte) { u32(b[peSections+2*40+8:], math.MaxUint32) }},
		{"PE relocation pointer inside edit", image, func(b []byte) { u32(b[peSections+24:], binary.LittleEndian.Uint32(b[peSections+2*40+20:])+1) }},
		{"PE initialized data underflow", image, func(b []byte) { u32(b[optional+8:], 0) }},
		{"PE unmapped debug directory", image, func(b []byte) { u32(b[optional+112+6*8:], math.MaxUint32) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := bytes.Clone(tt.raw)
			tt.edit(raw)
			original := bytes.Clone(raw)
			if _, err := Compress(raw); err == nil {
				t.Fatal("invalid native range accepted")
			}
			if !bytes.Equal(original, raw) {
				t.Fatal("failed rewrite modified input")
			}
		})
	}
}

func appendMachOCommand(raw []byte, kind uint32, size int) int {
	p := 32 + int(binary.LittleEndian.Uint32(raw[20:]))
	binary.LittleEndian.PutUint32(raw[16:], binary.LittleEndian.Uint32(raw[16:])+1)
	binary.LittleEndian.PutUint32(raw[20:], binary.LittleEndian.Uint32(raw[20:])+uint32(size))
	binary.LittleEndian.PutUint32(raw[p:], kind)
	binary.LittleEndian.PutUint32(raw[p+4:], uint32(size))
	return p
}

func TestMachOLinkeditCommands(t *testing.T) {
	// Use the on-disk values independently of the rewriter's named constants.
	// The two dyld commands include LC_REQ_DYLD in their command IDs.
	for _, kind := range []uint32{0x1e, 0x26, 0x29, 0x2b, 0x2e, 0x80000033, 0x80000034} {
		t.Run(fmt.Sprintf("%#x", kind), func(t *testing.T) {
			raw := machoFixture(t)
			const symtab = 32 + 72 + 3*80 + 72
			offset := binary.LittleEndian.Uint32(raw[symtab+8:])
			command := appendMachOCommand(raw, kind, 16)
			binary.LittleEndian.PutUint32(raw[command+8:], offset)
			binary.LittleEndian.PutUint32(raw[command+12:], 8)
			payload := bytes.Clone(raw[offset : offset+8])
			out, err := Compress(raw)
			if err != nil {
				t.Fatal(err)
			}
			next := binary.LittleEndian.Uint32(out[command+8:])
			if next >= offset || !bytes.Equal(out[next:next+8], payload) {
				t.Fatal("linkedit command lost its payload during compression")
			}
			if binary.LittleEndian.Uint32(out[command:]) != kind || binary.LittleEndian.Uint32(out[command+12:]) != 8 {
				t.Fatal("linkedit command kind or payload size changed")
			}
		})
	}
}

func TestMachOMetadataOffsets(t *testing.T) {
	raw := machoFixture(t)
	const symtab = 32 + 72 + 3*80 + 72
	linkedit := binary.LittleEndian.Uint32(raw[symtab+8:])
	dynamic := appendMachOCommand(raw, 0xb, 80)
	for _, field := range []int{32, 40, 48, 56, 64, 72} {
		binary.LittleEndian.PutUint32(raw[dynamic+field:], linkedit)
	}
	functions := appendMachOCommand(raw, 0x26, 16)
	binary.LittleEndian.PutUint32(raw[functions+8:], linkedit)
	note := appendMachOCommand(raw, 0x31, 40)
	binary.LittleEndian.PutUint64(raw[note+24:], uint64(linkedit))
	zeroNote := appendMachOCommand(raw, 0x31, 40)
	out, err := Compress(raw)
	if err != nil {
		t.Fatal(err)
	}
	next := binary.LittleEndian.Uint32(out[symtab+8:])
	if next >= linkedit {
		t.Fatal("metadata fixture did not move linkedit")
	}
	for _, field := range []int{32, 40, 48, 56, 64, 72} {
		if binary.LittleEndian.Uint32(out[dynamic+field:]) != next {
			t.Fatal("dynamic symbol metadata did not follow linkedit")
		}
	}
	if binary.LittleEndian.Uint32(out[functions+8:]) != next || binary.LittleEndian.Uint64(out[note+24:]) != uint64(next) || binary.LittleEndian.Uint64(out[zeroNote+24:]) != 0 {
		t.Fatal("function/note file offsets did not follow linkedit")
	}
	for _, tc := range []struct {
		kind    uint32
		size    int
		field   int
		pointer bool
	}{
		{0x1, 8, 0, false}, {0xb, 8, 0, false}, {0x26, 8, 0, false}, {0x31, 8, 0, false},
		{0xb, 80, 56, true}, {0x26, 16, 8, true}, {0x31, 40, 24, true},
	} {
		bad := machoFixture(t)
		p := appendMachOCommand(bad, tc.kind, tc.size)
		if tc.pointer {
			// Reference the middle of the compressible .debug_info payload.
			off := binary.LittleEndian.Uint32(bad[32+72+80+48:]) + 1
			binary.LittleEndian.PutUint32(bad[p+tc.field:], off)
		}
		if _, err := Compress(bad); err == nil {
			t.Fatalf("accepted malformed metadata command %#x", tc.kind)
		}
	}
}

func TestSmallMachOAndLongSectionNames(t *testing.T) {
	raw := machoFixtureSize(t, 4)
	if out, err := Compress(raw); err != nil || !bytes.Equal(out, raw) {
		t.Fatalf("page padding should avoid growing a small dSYM: %v", err)
	}
	raw = machoFixture(t)
	p := 32 + 72 + 80
	clear(raw[p : p+16])
	copy(raw[p:], "__debug_longname_")
	out, err := Compress(raw)
	if err != nil || string(out[p:p+16]) != "__zdebug_longnam" {
		t.Fatalf("compressed name did not fit the Mach-O header: %v", err)
	}
}

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
	"strings"
	"testing"
)

func TestCompactCarrierValidation(t *testing.T) {
	if _, _, err := compactCarrier(nil, &binaryInfo{entryVMSize: 1}, 2); err == nil {
		t.Fatal("oversized entry was accepted")
	}
	if _, _, err := compactCarrier(nil, &binaryInfo{format: "unknown"}, 0); err == nil {
		t.Fatal("unknown format was accepted")
	}
	if got := alignUp(7, 0); got != 7 {
		t.Fatalf("alignUp(7, 0) = %d", got)
	}
	if got := alignUp(^uint64(0)-1, 8); got != ^uint64(0) {
		t.Fatalf("overflowing alignUp = %#x", got)
	}
	if got := alignDown(7, 1); got != 7 {
		t.Fatalf("alignDown(7, 1) = %d", got)
	}
}

func TestPatchMachOFileOffsetCommands(t *testing.T) {
	tests := []struct {
		cmd       uint32
		size      uint32
		positions []uint64
		wide      bool
	}{
		{0x2, 24, []uint64{8, 16}, false},
		{0xb, 80, []uint64{32, 40, 48, 56, 64, 72}, false},
		{0x22, 48, []uint64{8, 16, 24, 32, 40}, false},
		{0x1d, 16, []uint64{8}, false},
		{0x16, 16, []uint64{8}, false},
		{0x31, 40, []uint64{24}, true},
		{0x21, 20, []uint64{8}, false},
	}
	for _, test := range tests {
		raw := make([]byte, 160)
		binary.LittleEndian.PutUint32(raw, uint32(macho.Magic64))
		binary.LittleEndian.PutUint32(raw[16:], 1)
		binary.LittleEndian.PutUint32(raw[32:], test.cmd)
		binary.LittleEndian.PutUint32(raw[36:], test.size)
		for _, pos := range test.positions {
			if test.wide {
				binary.LittleEndian.PutUint64(raw[32+pos:], 0x200)
			} else {
				binary.LittleEndian.PutUint32(raw[32+pos:], 0x200)
			}
		}
		layout, err := parseMachOLayout(raw)
		if err != nil {
			t.Fatalf("parse command %#x: %v", test.cmd, err)
		}
		if err := patchMachOFileOffsets(raw, layout, 0, 0, 0x100, 0x120); err != nil {
			t.Fatalf("command %#x: %v", test.cmd, err)
		}
		for _, pos := range test.positions {
			var got uint64
			if test.wide {
				got = binary.LittleEndian.Uint64(raw[32+pos:])
			} else {
				got = uint64(binary.LittleEndian.Uint32(raw[32+pos:]))
			}
			if got != 0x1e0 {
				t.Fatalf("command %#x field %#x = %#x", test.cmd, pos, got)
			}
		}
	}
}

func TestPatchMachOFileOffsetCommandErrors(t *testing.T) {
	for _, cmd := range []uint32{0x35, 0xdeadbeef} {
		raw := make([]byte, 64)
		binary.LittleEndian.PutUint32(raw, uint32(macho.Magic64))
		binary.LittleEndian.PutUint32(raw[16:], 1)
		binary.LittleEndian.PutUint32(raw[32:], cmd)
		binary.LittleEndian.PutUint32(raw[36:], 8)
		if _, err := parseMachOLayout(raw); err == nil {
			t.Fatalf("command %#x was accepted", cmd)
		}
	}
	raw := make([]byte, 64)
	binary.LittleEndian.PutUint32(raw, uint32(macho.Magic64))
	binary.LittleEndian.PutUint32(raw[16:], 1)
	binary.LittleEndian.PutUint32(raw[32:], 0x21)
	binary.LittleEndian.PutUint32(raw[36:], 20)
	binary.LittleEndian.PutUint32(raw[48:], 1)
	if _, err := parseMachOLayout(raw); err == nil {
		t.Fatal("encrypted image was accepted")
	}
	for _, test := range []struct {
		name   string
		cmdsz  uint32
		nsects uint32
	}{
		{"command past EOF", 128, 0},
		{"short segment", 64, 0},
		{"truncated sections", 72, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := make([]byte, 104)
			binary.LittleEndian.PutUint32(raw, uint32(macho.Magic64))
			binary.LittleEndian.PutUint32(raw[16:], 1)
			binary.LittleEndian.PutUint32(raw[32:], 0x19)
			binary.LittleEndian.PutUint32(raw[36:], test.cmdsz)
			binary.LittleEndian.PutUint32(raw[32+64:], test.nsects)
			if _, err := parseMachOLayout(raw); err == nil {
				t.Fatal("malformed segment was accepted")
			}
		})
	}
}

func TestMachOLoadCommandClassification(t *testing.T) {
	for _, cmd := range []uint32{0x3, 0x19, 0x80000028} {
		if !machoLoadCommandHasNoFileOffset(cmd) {
			t.Fatalf("command %#x should need no offset rewrite", cmd)
		}
	}
	if machoLoadCommandHasNoFileOffset(0xdeadbeef) {
		t.Fatal("unknown command was classified as offset-free")
	}
}

func compactMachOInput(t *testing.T) ([]byte, *binaryInfo, machoLayout, *machoSegmentLayout, *machoSectionLayout) {
	t.Helper()
	path := machoRewriteFixture(t, 65536)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), info.raw...)
	layout, err := parseMachOLayout(raw)
	if err != nil {
		t.Fatal(err)
	}
	var carrier *machoSegmentLayout
	var entry *machoSectionLayout
	for i := range layout.segments {
		if layout.segments[i].name != "__LLGO" {
			continue
		}
		carrier = &layout.segments[i]
		for j := range carrier.sections {
			if carrier.sections[j].name == "__llgo_fie" {
				entry = &carrier.sections[j]
			}
		}
	}
	if carrier == nil || entry == nil {
		t.Fatal("fixture has no Mach-O carrier")
	}
	return raw, info, layout, carrier, entry
}

func TestCompactMachORejectsMalformedCarrierShapes(t *testing.T) {
	tests := []struct {
		name string
		want string
		edit func([]byte, *binaryInfo, machoLayout, *machoSegmentLayout, *machoSectionLayout) uint64
	}{
		{
			name: "file range",
			want: "file range",
			edit: func(raw []byte, _ *binaryInfo, _ machoLayout, carrier *machoSegmentLayout, _ *machoSectionLayout) uint64 {
				binary.LittleEndian.PutUint64(raw[carrier.header+48:], uint64(len(raw))+1)
				return 64
			},
		},
		{
			name: "entry outside segment",
			want: "outside __LLGO",
			edit: func(raw []byte, _ *binaryInfo, _ machoLayout, carrier *machoSegmentLayout, entry *machoSectionLayout) uint64 {
				binary.LittleEndian.PutUint32(raw[entry.header+48:], uint32(carrier.fileoff-1))
				return 64
			},
		},
		{
			name: "segment follows carrier",
			want: "follows __LLGO",
			edit: func(raw []byte, _ *binaryInfo, layout machoLayout, carrier *machoSegmentLayout, _ *machoSectionLayout) uint64 {
				oldEnd := carrier.fileoff + carrier.filesize
				for _, segment := range layout.segments {
					if segment.name == "__TEXT" {
						binary.LittleEndian.PutUint64(raw[segment.header+40:], oldEnd)
						binary.LittleEndian.PutUint64(raw[segment.header+48:], 1)
					}
				}
				return 64
			},
		},
		{
			name: "payload outside segment",
			want: "outside __LLGO",
			edit: func(_ []byte, _ *binaryInfo, _ machoLayout, carrier *machoSegmentLayout, _ *machoSectionLayout) uint64 {
				return carrier.filesize + 1
			},
		},
		{
			name: "unaligned carrier cannot grow",
			want: "compact __LLGO size",
			edit: func(raw []byte, _ *binaryInfo, _ machoLayout, _ *machoSegmentLayout, entry *machoSectionLayout) uint64 {
				binary.LittleEndian.PutUint64(raw[entry.header-24:], entry.size)
				return entry.size
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, info, layout, carrier, entry := compactMachOInput(t)
			used := test.edit(raw, info, layout, carrier, entry)
			if _, _, err := compactMachO(raw, info, used); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("compactMachO error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCompactMachOPropagatesParserAndOffsetErrors(t *testing.T) {
	if _, _, err := compactMachO(nil, &binaryInfo{}, 0); err == nil {
		t.Fatal("compactMachO accepted an invalid image")
	}
	raw, info, layout, carrier, entry := compactMachOInput(t)
	used := uint64(64)
	cutStart := carrier.fileoff + alignUp(entry.offset+used-carrier.fileoff, machoPageSize(raw))
	if len(layout.offsets) == 0 {
		t.Fatal("fixture has no linkedit offsets")
	}
	field := layout.offsets[0]
	if field.width != 4 {
		t.Fatalf("first offset width = %d", field.width)
	}
	binary.LittleEndian.PutUint32(raw[field.pos:], uint32(cutStart+1))
	if _, _, err := compactMachO(raw, info, used); err == nil || !strings.Contains(err.Error(), "compacted range") {
		t.Fatalf("compactMachO error = %v", err)
	}
}

func TestCompactMachONoRemovablePage(t *testing.T) {
	raw, info, _, carrier, entry := compactMachOInput(t)
	// The synthetic carrier starts at a non-16K file offset. Exercise the
	// no-cut path with the amd64 4K page contract, where its file size is
	// already page aligned.
	binary.LittleEndian.PutUint32(raw[4:], 0x01000007)
	entryUsed := carrier.filesize - (entry.offset - carrier.fileoff)
	out, removed, err := compactMachO(raw, info, entryUsed)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 || len(out) != len(info.raw) {
		t.Fatalf("no-op compaction removed=%d sizes=%d/%d", removed, len(out), len(info.raw))
	}
}

func TestPatchMachOFileOffsetsValidation(t *testing.T) {
	tests := []struct {
		name   string
		layout machoLayout
		setup  func([]byte)
		want   string
	}{
		{
			name:   "truncated 32-bit field",
			layout: machoLayout{offsets: []machoOffsetField{{pos: 127, width: 4, label: "small"}}},
			want:   "truncated",
		},
		{
			name:   "truncated 64-bit field",
			layout: machoLayout{offsets: []machoOffsetField{{pos: 124, width: 8, label: "wide"}}},
			want:   "truncated",
		},
		{
			name:   "32-bit field in cut",
			layout: machoLayout{offsets: []machoOffsetField{{pos: 96, width: 4, label: "small"}}},
			setup:  func(raw []byte) { binary.LittleEndian.PutUint32(raw[96:], 0x110) },
			want:   "compacted range",
		},
		{
			name:   "64-bit field in cut",
			layout: machoLayout{offsets: []machoOffsetField{{pos: 96, width: 8, label: "wide"}}},
			setup:  func(raw []byte) { binary.LittleEndian.PutUint64(raw[96:], 0x110) },
			want:   "compacted range",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := make([]byte, 128)
			if test.setup != nil {
				test.setup(raw)
			}
			if err := patchMachOFileOffsets(raw, test.layout, ^uint64(0), 0, 0x100, 0x120); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("patch error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPatchMachOFileOffsetsMovesSegmentsAndSections(t *testing.T) {
	raw := make([]byte, 192)
	segment := machoSegmentLayout{
		name: "__LINKEDIT", header: 0,
		sections: []machoSectionLayout{{name: "__symbols", segment: "__LINKEDIT", header: 64, nreloc: 1}},
	}
	binary.LittleEndian.PutUint64(raw[40:], 0x200)
	binary.LittleEndian.PutUint32(raw[64+48:], 0x220)
	binary.LittleEndian.PutUint32(raw[64+56:], 0x240)
	layout := machoLayout{segments: []machoSegmentLayout{segment}}
	if err := patchMachOFileOffsets(raw, layout, ^uint64(0), 0, 0x100, 0x120); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[40:]); got != 0x1e0 {
		t.Fatalf("segment fileoff = %#x", got)
	}
	if got := binary.LittleEndian.Uint32(raw[64+48:]); got != 0x200 {
		t.Fatalf("section offset = %#x", got)
	}
	if got := binary.LittleEndian.Uint32(raw[64+56:]); got != 0x220 {
		t.Fatalf("section reloff = %#x", got)
	}
}

func TestPatchMachOFileOffsetsBranchErrors(t *testing.T) {
	tests := []struct {
		name          string
		layout        machoLayout
		carrierHeader uint64
		setup         func([]byte)
	}{
		{
			name:          "segment",
			layout:        machoLayout{segments: []machoSegmentLayout{{name: "__LINKEDIT", header: 0}}},
			carrierHeader: ^uint64(0),
			setup:         func(raw []byte) { binary.LittleEndian.PutUint64(raw[40:], 0x110) },
		},
		{
			name:   "section",
			layout: machoLayout{segments: []machoSegmentLayout{{header: 0, sections: []machoSectionLayout{{name: "__symbols", header: 64}}}}},
			setup:  func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+48:], 0x110) },
		},
		{
			name:   "relocation",
			layout: machoLayout{segments: []machoSegmentLayout{{header: 0, sections: []machoSectionLayout{{name: "__llgo_fie", header: 64, nreloc: 1}}}}},
			setup:  func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+56:], 0x110) },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := make([]byte, 160)
			test.setup(raw)
			if err := patchMachOFileOffsets(raw, test.layout, test.carrierHeader, 0, 0x100, 0x120); err == nil {
				t.Fatal("offset inside cut was accepted")
			}
		})
	}
	raw := make([]byte, 128)
	binary.LittleEndian.PutUint64(raw[96:], 0x80)
	layout := machoLayout{offsets: []machoOffsetField{{pos: 88, width: 4}, {pos: 96, width: 8}}}
	if err := patchMachOFileOffsets(raw, layout, ^uint64(0), 0, 0x100, 0x120); err != nil {
		t.Fatal(err)
	}
}

func compactELFInput(t *testing.T) ([]byte, *binaryInfo, []elfSectionLayout, []elfProgramLayout) {
	t.Helper()
	path := buildELF(t, fixtureFns(), fixtureEntry, 8192)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), info.raw...)
	sections, programs, _, err := parseELFLayout(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, info, sections, programs
}

func findELFSection(t *testing.T, sections []elfSectionLayout, name string) elfSectionLayout {
	t.Helper()
	for _, section := range sections {
		if section.name == name {
			return section
		}
	}
	t.Fatalf("fixture has no section %s", name)
	return elfSectionLayout{}
}

func TestCompactELFRejectsMalformedCarrierShapes(t *testing.T) {
	t.Run("missing entry", func(t *testing.T) {
		raw, info, sections, _ := compactELFInput(t)
		entry := findELFSection(t, sections, "llgo_funcinfo_entry")
		text := findELFSection(t, sections, ".text")
		binary.LittleEndian.PutUint32(raw[entry.header:], binary.LittleEndian.Uint32(raw[text.header:]))
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "missing ELF entry") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("entry not loaded", func(t *testing.T) {
		raw, info, _, programs := compactELFInput(t)
		binary.LittleEndian.PutUint32(raw[programs[1].header:], uint32(elf.PT_NOTE))
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "not in a PT_LOAD") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("payload outside", func(t *testing.T) {
		raw, info, _, programs := compactELFInput(t)
		if _, _, err := compactELF(raw, info, programs[1].filesz+1); err == nil || !strings.Contains(err.Error(), "outside PT_LOAD") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("allocated section follows payload", func(t *testing.T) {
		raw, info, sections, programs := compactELFInput(t)
		data := findELFSection(t, sections, ".data")
		binary.LittleEndian.PutUint64(raw[programs[1].header+32:], data.offset+data.size-programs[1].offset)
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "allocated section") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("virtual tail uncovered", func(t *testing.T) {
		raw, info, _, programs := compactELFInput(t)
		binary.LittleEndian.PutUint64(raw[programs[1].header+40:], 1)
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "virtual tail") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("invalid following alignment", func(t *testing.T) {
		raw, info, sections, _ := compactELFInput(t)
		symtab := findELFSection(t, sections, ".symtab")
		binary.LittleEndian.PutUint64(raw[symtab.header+48:], 24)
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "invalid alignment") {
			t.Fatalf("compact error = %v", err)
		}
	})
	t.Run("program overlaps cut", func(t *testing.T) {
		raw, info, _, programs := compactELFInput(t)
		carrier := programs[1]
		oldEnd := carrier.offset + carrier.filesz
		binary.LittleEndian.PutUint16(raw[56:], 3)
		h := uint64(64 + 2*56)
		binary.LittleEndian.PutUint32(raw[h:], uint32(elf.PT_NOTE))
		binary.LittleEndian.PutUint64(raw[h+8:], oldEnd-16)
		binary.LittleEndian.PutUint64(raw[h+32:], 16)
		binary.LittleEndian.PutUint64(raw[h+40:], 16)
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "overlaps compacted range") {
			t.Fatalf("compact error = %v", err)
		}
	})
}

func TestCompactELFUpdatesNOBITSTail(t *testing.T) {
	raw, info, sections, programs := compactELFInput(t)
	carrier := programs[1]
	oldEnd := carrier.offset + carrier.filesz
	data := findELFSection(t, sections, ".data")
	binary.LittleEndian.PutUint32(raw[data.header+4:], uint32(elf.SHT_NOBITS))
	binary.LittleEndian.PutUint64(raw[data.header+24:], oldEnd-16)
	binary.LittleEndian.PutUint64(raw[data.header+32:], 32)
	out, removed, err := compactELF(raw, info, 64)
	if err != nil {
		t.Fatal(err)
	}
	if removed == 0 {
		t.Fatal("fixture did not compact")
	}
	parsed, _, _, err := parseELFLayout(out)
	if err != nil {
		t.Fatal(err)
	}
	got := findELFSection(t, parsed, ".data")
	if got.offset >= oldEnd {
		t.Fatalf("NOBITS offset was not moved into compact tail: %#x", got.offset)
	}
}

func TestCompactELFRejectsSectionAndHeaderOverlap(t *testing.T) {
	t.Run("section", func(t *testing.T) {
		raw, info, sections, programs := compactELFInput(t)
		carrier := programs[1]
		oldEnd := carrier.offset + carrier.filesz
		symtab := findELFSection(t, sections, ".symtab")
		binary.LittleEndian.PutUint64(raw[symtab.header+8:], 0)
		binary.LittleEndian.PutUint64(raw[symtab.header+24:], oldEnd-32)
		binary.LittleEndian.PutUint64(raw[symtab.header+32:], 16)
		if _, _, err := compactELF(raw, info, 64); err == nil || !strings.Contains(err.Error(), "section .symtab overlaps") {
			t.Fatalf("compactELF error = %v", err)
		}
	})
	t.Run("section headers", func(t *testing.T) {
		raw, info, sections, programs := compactELFInput(t)
		carrier := programs[1]
		shoff := binary.LittleEndian.Uint64(raw[40:])
		for _, section := range sections {
			if section.name != "llgo_funcinfo_entry" {
				binary.LittleEndian.PutUint64(raw[section.header+8:], 0)
			}
		}
		binary.LittleEndian.PutUint64(raw[carrier.header+32:], shoff+64-carrier.offset)
		used := shoff - findELFSection(t, sections, "llgo_funcinfo_entry").offset
		if _, _, err := compactELF(raw, info, used); err == nil || !strings.Contains(err.Error(), "section headers overlap") {
			t.Fatalf("compactELF error = %v", err)
		}
	})
}

func TestParseBinaryLayoutsRejectInvalidHeaders(t *testing.T) {
	for _, raw := range [][]byte{nil, bytes.Repeat([]byte{0}, 64)} {
		if _, err := parseMachOLayout(raw); err == nil {
			t.Fatal("parseMachOLayout accepted a non-Mach-O image")
		}
		if _, _, _, err := parseELFLayout(raw); err == nil {
			t.Fatal("parseELFLayout accepted a non-ELF image")
		}
	}
	raw, _, _, _ := compactELFInput(t)
	for _, edit := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint16(b[56:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[60:], 0) },
	} {
		bad := append([]byte(nil), raw...)
		edit(bad)
		if _, _, _, err := parseELFLayout(bad); err == nil {
			t.Fatal("parseELFLayout accepted an invalid table")
		}
	}
}

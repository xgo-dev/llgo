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
	"debug/macho"
	"encoding/binary"
	"os"
	"testing"
)

func TestCanonicalOwner(t *testing.T) {
	elf := &binaryInfo{format: "elf"}
	macho := &binaryInfo{format: "macho"}
	id := fnv64("example.com/p.F")
	cases := []struct {
		info *binaryInfo
		name string
		want bool
	}{
		// ELF: symbol names are source-level.
		{elf, "example.com/p.F", true},
		{elf, "example.com/p.G", false},
		// Mach-O: one C-mangling underscore, and debug/macho's suffix-shared
		// string table can surface one underscore more or less.
		{macho, "_example.com/p.F", true},
		{macho, "example.com/p.F", true},
		// An LTO inline copy: record id names F but the owner is the host.
		{macho, "_example.com/p.Host", false},
	}
	for _, c := range cases {
		if got := canonicalOwner(c.info, c.name, id); got != c.want {
			t.Errorf("canonicalOwner(%s, %q) = %v, want %v", c.info.format, c.name, got, c.want)
		}
	}
}

func TestBinaryBoundsHelpers(t *testing.T) {
	info := &binaryInfo{
		raw:  []byte{0, 1, 2, 3},
		secs: []secInfo{{vmaddr: 0x1000, size: 4, fileOff: 0}},
	}
	if got := readVM(info, 0x1001, 2); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("readVM in range = %v", got)
	}
	if got := readVM(info, 0x2000, 3); len(got) != 3 || got[0] != 0 || got[1] != 0 || got[2] != 0 {
		t.Fatalf("readVM out of range = %v, want three zero bytes", got)
	}
	if _, err := sectionBytes(info.raw, 3, 2); err == nil {
		t.Fatal("sectionBytes accepted an out-of-file range")
	}

	path := t.TempDir() + "/not-an-executable"
	if err := os.WriteFile(path, []byte("not an executable"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := load(path); err == nil {
		t.Fatal("load accepted a non-Mach-O/non-ELF file")
	}
}

func TestFinishAndBuildFtabBoundaryCases(t *testing.T) {
	info := &binaryInfo{
		textStart: 0x1000,
		textEnd:   0x1040,
		syms: []textSym{
			{addr: 0x1020, name: "second"},
			{addr: 0x1000, name: "first"},
			{addr: 0x1000, name: "first-alias"},
		},
	}
	finish(info)
	if len(info.syms) != 2 || info.syms[0].size != 0x20 || info.syms[1].size != 0x20 {
		t.Fatalf("finish() symbols = %+v", info.syms)
	}
	if ftab, base := buildFtab(info, nil); ftab != nil || base != info.textStart {
		t.Fatalf("empty buildFtab = (%v, %#x)", ftab, base)
	}
	ftab, base := buildFtab(info, []siteRecord{{pc: 0x1020}, {pc: 0x1000}, {pc: 0x1000}})
	if base != 0x1000 || len(ftab) != 3 || ftab[0].EntryOff != 0 || ftab[1].EntryOff != 0x20 {
		t.Fatalf("alias buildFtab = (%+v, %#x)", ftab, base)
	}
}

func TestLoadBindTargetsRejectsMalformedBounds(t *testing.T) {
	mf := new(macho.File)
	for name, raw := range map[string][]byte{
		"short header": make([]byte, 31),
		"no command":   make([]byte, 32),
		"invalid command size": func() []byte {
			raw := make([]byte, 40)
			binary.LittleEndian.PutUint32(raw[16:], 1)
			binary.LittleEndian.PutUint32(raw[36:], 4)
			return raw
		}(),
		"short command": func() []byte {
			raw := make([]byte, 40)
			binary.LittleEndian.PutUint32(raw[16:], 1)
			binary.LittleEndian.PutUint32(raw[32:], lcDyldChainedFixups)
			binary.LittleEndian.PutUint32(raw[36:], 8)
			return raw
		}(),
		"oversized payload":  chainedFixupFixture(64, 128, 1),
		"undersized payload": chainedFixupFixture(64, 16, 1),
	} {
		t.Run(name, func(t *testing.T) {
			info := &binaryInfo{raw: raw}
			loadBindTargets(info, mf)
			if len(info.bindTargets) != 0 {
				t.Fatalf("malformed input produced targets %v", info.bindTargets)
			}
		})
	}
}

func chainedFixupFixture(fixOff, fixSize uint32, format uint32) []byte {
	raw := make([]byte, 128)
	binary.LittleEndian.PutUint32(raw[16:], 1)
	binary.LittleEndian.PutUint32(raw[32:], lcDyldChainedFixups)
	binary.LittleEndian.PutUint32(raw[36:], 16)
	binary.LittleEndian.PutUint32(raw[40:], fixOff)
	binary.LittleEndian.PutUint32(raw[44:], fixSize)
	if int(fixOff)+24 <= len(raw) {
		binary.LittleEndian.PutUint32(raw[fixOff+8:], 28)
		binary.LittleEndian.PutUint32(raw[fixOff+12:], 40)
		binary.LittleEndian.PutUint32(raw[fixOff+16:], 1)
		binary.LittleEndian.PutUint32(raw[fixOff+20:], format)
	}
	return raw
}

func TestLoadBindTargetsFormatsAndInvalidImports(t *testing.T) {
	for _, format := range []uint32{1, 2} {
		t.Run(string(rune('0'+format)), func(t *testing.T) {
			raw := chainedFixupFixture(64, 64, format)
			copy(raw[64+40:], "_target\x00")
			mf := &macho.File{Symtab: &macho.Symtab{Syms: []macho.Symbol{{Name: "target", Value: 0x1234}}}}
			info := &binaryInfo{raw: raw}
			loadBindTargets(info, mf)
			if len(info.bindTargets) != 1 || info.bindTargets[0] != 0x1234 {
				t.Fatalf("format %d targets = %v", format, info.bindTargets)
			}
		})
	}

	for name, mutate := range map[string]func([]byte){
		"no imports":       func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+16:], 0) },
		"too many imports": func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+16:], 1<<24+1) },
		"unknown format":   func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+20:], 99) },
		"truncated record": func(raw []byte) { binary.LittleEndian.PutUint32(raw[64+8:], 63) },
		"invalid name": func(raw []byte) {
			binary.LittleEndian.PutUint32(raw[64+28:], 1<<9)
			binary.LittleEndian.PutUint32(raw[64+12:], 127)
		},
	} {
		t.Run(name, func(t *testing.T) {
			raw := chainedFixupFixture(64, 64, 1)
			mutate(raw)
			info := &binaryInfo{raw: raw}
			loadBindTargets(info, new(macho.File))
			if name != "truncated record" && name != "invalid name" && len(info.bindTargets) != 0 {
				t.Fatalf("invalid import header produced targets %v", info.bindTargets)
			}
		})
	}
}

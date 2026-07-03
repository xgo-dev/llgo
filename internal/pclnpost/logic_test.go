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
	"encoding/binary"
	"testing"
)

func rec(pc, id uint64) []byte {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:], pc)
	binary.LittleEndian.PutUint64(b[8:], id)
	return b[:]
}

func TestParseRecordsELF(t *testing.T) {
	info := &binaryInfo{format: "elf", textStart: 0x1000, textEnd: 0x2000}
	sec := append(rec(0, 0), rec(0x1100, 7)...) // zero keep-alive skipped
	sec = append(sec, rec(0x1200, 0)...)        // id==0 skipped
	sec = append(sec, rec(0x1300, 9)...)
	got := parseRecords(info, sec)
	if len(got) != 2 || got[0].pc != 0x1100 || got[1].symbolID != 9 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseRecordsMachO(t *testing.T) {
	info := &binaryInfo{format: "macho", textStart: 0x100001000, textEnd: 0x100002000,
		bindTargets: []uint64{0, 0x100001500}}
	// Rebase-encoded slot: chain metadata above the low 36 bits.
	rebase := (uint64(3) << 51) | 0x100001100
	// Bind-encoded slot: bit 63, ordinal 1, addend 4.
	bind := (uint64(1) << 63) | (uint64(4) << 24) | 1
	// Bind to an unresolved ordinal is dropped.
	badBind := (uint64(1) << 63) | 0
	sec := append(rec(rebase, 5), rec(bind, 6)...)
	sec = append(sec, rec(badBind, 8)...)
	got := parseRecords(info, sec)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].pc != 0x100001100 || got[1].pc != 0x100001504 {
		t.Fatalf("decoded pcs %#x %#x", got[0].pc, got[1].pc)
	}
}

func TestDedupeCanonicalAndInline(t *testing.T) {
	fn := "example.com/p.F"
	host := "example.com/p.Host"
	info := &binaryInfo{format: "elf", textStart: 0x1000, textEnd: 0x4000, syms: []textSym{
		{addr: 0x1000, size: 0x100, name: fn},
		{addr: 0x1100, size: 0x100, name: host},
		{addr: 0x1200, size: 0x10, name: "__llgo_stub." + fn},
	}}
	id := fnv64(fn)
	recs := []siteRecord{
		{pc: 0x1004, symbolID: id}, // canonical, inside F
		{pc: 0x1104, symbolID: id}, // inline copy inside Host
		{pc: 0x1204, symbolID: id}, // stub wrapper, canonical
		{pc: 0x1008, symbolID: id}, // duplicate owner, collapsed
		{pc: 0x9999, symbolID: id}, // no owner
	}
	kept, inline, nosym := dedupe(info, recs, false)
	if len(kept) != 2 || inline != 1 || nosym != 1 {
		t.Fatalf("kept=%d inline=%d nosym=%d", len(kept), inline, nosym)
	}
	if kept[0].pc != 0x1000 || kept[1].pc != 0x1200 {
		t.Fatalf("normalized pcs %#x %#x", kept[0].pc, kept[1].pc)
	}
}

func TestBuildFtabSortsAndAppendsSentinel(t *testing.T) {
	info := &binaryInfo{textStart: 0x1000, textEnd: 0x3000}
	kept := []siteRecord{{pc: 0x2000, symbolID: 2}, {pc: 0x1100, symbolID: 1}, {pc: 0x2000, symbolID: 3}}
	ftab, base := buildFtab(info, kept)
	if base != 0x1100 || len(ftab) != 3 {
		t.Fatalf("base=%#x len=%d", base, len(ftab))
	}
	if ftab[0].EntryOff != 0 || ftab[1].EntryOff != 0xF00 || ftab[2].EntryOff != 0x3000-0x1100 {
		t.Fatalf("offsets %+v", ftab)
	}
}

func TestOwnerLookup(t *testing.T) {
	info := &binaryInfo{syms: []textSym{{addr: 0x100, size: 0x10, name: "a"}, {addr: 0x110, size: 0x10, name: "b"}}}
	if s, ok := owner(info, 0x105); !ok || s.name != "a" {
		t.Fatalf("got %+v ok=%v", s, ok)
	}
	if _, ok := owner(info, 0x50); ok {
		t.Fatal("below first symbol should miss")
	}
	if _, ok := owner(info, 0x200); ok {
		t.Fatal("past extent should miss")
	}
}

func TestFnv64NonZero(t *testing.T) {
	if fnv64("") == 0 || fnv64("a") == fnv64("b") {
		t.Fatal("fnv sanity")
	}
}

func TestSymbolAddrBothFormats(t *testing.T) {
	elfPath := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 4096, 256)
	if addr, err := symbolAddr(elfPath, "example.com/p.A"); err != nil || addr != 0x10000 {
		t.Fatalf("elf symbolAddr = %#x, %v", addr, err)
	}
	if _, err := symbolAddr(elfPath, "no.such.symbol"); err == nil {
		t.Fatal("expected missing-symbol error on elf")
	}
	machoPath := buildMachO(t, rec(0, 0), rec(0, 0),
		[]elfFn{{name: "example.com/p.M", size: 0x10}})
	if addr, err := symbolAddr(machoPath, "example.com/p.M"); err != nil || addr == 0 {
		t.Fatalf("macho symbolAddr = %#x, %v", addr, err)
	}
	if _, err := symbolAddr(machoPath, "no.such.symbol"); err == nil {
		t.Fatal("expected missing-symbol error on macho")
	}
}

func TestDecodePtrVal(t *testing.T) {
	elf := &binaryInfo{format: "elf"}
	if got := decodePtrVal(elf, 0x1234); got != 0x1234 {
		t.Fatalf("elf passthrough %#x", got)
	}
	macho := &binaryInfo{format: "macho", imageBase: 0x100000000}
	chained := (uint64(5) << 51) | 0x100000abc
	if got := decodePtrVal(macho, chained); got != 0x100000abc {
		t.Fatalf("chained decode %#x", got)
	}
	if got := decodePtrVal(macho, 0x100000abc); got != 0x100000abc {
		t.Fatalf("plain macho %#x", got)
	}
}

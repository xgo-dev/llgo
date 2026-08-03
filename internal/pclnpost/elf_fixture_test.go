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
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildELF fabricates the minimal ELF load() understands: .text, the two
// funcinfo site sections, a data section holding the symbol index, .symtab
// and .strtab. Layout is one flat file segment; vmaddr == file offset + 0x10000.
type elfFn struct {
	name string
	size uint64
}

func buildELF(t *testing.T, fns []elfFn, entryRecs, stubRecs func(addrOf func(string) uint64) []byte, entryPad, stubPad int) string {
	return buildELFExternal(t, fns, entryRecs, stubRecs, entryPad, stubPad, nil, nil)
}

func buildELFExternal(t *testing.T, fns []elfFn, entryRecs, stubRecs func(addrOf func(string) uint64) []byte, entryPad, stubPad int, pcLine, identity []byte) string {
	t.Helper()
	const base = uint64(0x10000)
	var text bytes.Buffer
	addr := map[string]uint64{}
	for _, fn := range fns {
		addr[fn.name] = base + uint64(text.Len())
		text.Write(make([]byte, fn.size))
	}
	addrOf := func(n string) uint64 { return addr[n] }
	entry := entryRecs(addrOf)
	entry = append(entry, make([]byte, entryPad)...)
	stub := stubRecs(addrOf)
	stub = append(stub, make([]byte, stubPad)...)

	// Symbol index: sorted {u64 fnv(name), u32 funcIndex, u32 pad}.
	type sie struct {
		id  uint64
		idx uint32
	}
	var idx []sie
	for i, fn := range fns {
		idx = append(idx, sie{fnv64(fn.name), uint32(i + 1)})
	}
	for i := 0; i < len(idx); i++ {
		for j := i + 1; j < len(idx); j++ {
			if idx[j].id < idx[i].id {
				idx[i], idx[j] = idx[j], idx[i]
			}
		}
	}
	var data bytes.Buffer
	for _, e := range idx {
		binary.Write(&data, binary.LittleEndian, e.id)
		binary.Write(&data, binary.LittleEndian, e.idx)
		binary.Write(&data, binary.LittleEndian, uint32(0))
	}
	idxTableAddr := base + 0x400000
	// pointer global + count global at fixed addrs inside data section
	ptrGlobal := idxTableAddr + uint64(data.Len())
	binary.Write(&data, binary.LittleEndian, idxTableAddr)
	cntGlobal := idxTableAddr + uint64(data.Len())
	binary.Write(&data, binary.LittleEndian, uint64(len(idx)))

	// Entry section gets a meta record up front (pc=0 rows are skipped by
	// parseRecords; the tool locates the index through them).
	meta := append(rec(0, metaRecordMagic), rec(ptrGlobal, 0)...)
	meta = append(meta, rec(cntGlobal, 0)...)
	entry = append(meta, entry...)

	// strtab / symtab
	strtab := []byte{0}
	var symtab bytes.Buffer
	symtab.Write(make([]byte, 24)) // null symbol
	for _, fn := range fns {
		nameOff := len(strtab)
		strtab = append(strtab, fn.name...)
		strtab = append(strtab, 0)
		binary.Write(&symtab, binary.LittleEndian, uint32(nameOff))
		symtab.WriteByte(0x12) // GLOBAL FUNC
		symtab.WriteByte(0)
		binary.Write(&symtab, binary.LittleEndian, uint16(1)) // shndx .text
		binary.Write(&symtab, binary.LittleEndian, addr[fn.name])
		binary.Write(&symtab, binary.LittleEndian, fn.size)
	}

	sectionNames := []string{".text", "llgo_funcinfo_entry", "llgo_funcinfo_stubsite"}
	if pcLine != nil {
		sectionNames = append(sectionNames, "llgo_pcline")
	}
	if identity != nil {
		sectionNames = append(sectionNames, "llgo_pclntab_id")
	}
	sectionNames = append(sectionNames, ".data", ".symtab", ".strtab", ".shstrtab")
	shstr := []byte{0}
	names := map[string]uint32{}
	for _, n := range sectionNames {
		names[n] = uint32(len(shstr))
		shstr = append(shstr, n...)
		shstr = append(shstr, 0)
	}

	type sec struct {
		name  string
		typ   uint32
		addr  uint64
		body  []byte
		link  uint32
		entsz uint64
	}
	secs := []sec{
		{".text", 1, base, text.Bytes(), 0, 0},
		{"llgo_funcinfo_entry", 1, base + 0x4000, entry, 0, 0},
		{"llgo_funcinfo_stubsite", 1, base + 0x6000, stub, 0, 0},
	}
	if pcLine != nil {
		secs = append(secs, sec{"llgo_pcline", 1, base + 0x7000, pcLine, 0, 0})
	}
	if identity != nil {
		secs = append(secs, sec{"llgo_pclntab_id", 1, base + 0x7800, identity, 0, 0})
	}
	secs = append(secs, sec{".data", 1, idxTableAddr, data.Bytes(), 0, 0})
	strtabIndex := uint32(len(secs) + 2)
	secs = append(secs,
		sec{".symtab", 2, 0, symtab.Bytes(), strtabIndex, 24},
		sec{".strtab", 3, 0, strtab, 0, 0},
		sec{".shstrtab", 3, 0, shstr, 0, 0},
	)

	var body bytes.Buffer
	body.Write(make([]byte, 64+2*56)) // ELF header + two program headers
	offs := make([]uint64, len(secs))
	for i := range secs {
		for body.Len()%16 != 0 {
			body.WriteByte(0)
		}
		offs[i] = uint64(body.Len())
		body.Write(secs[i].body)
	}
	for body.Len()%16 != 0 {
		body.WriteByte(0)
	}
	shoff := uint64(body.Len())
	// null section header
	body.Write(make([]byte, 64))
	for i, s := range secs {
		var sh [64]byte
		binary.LittleEndian.PutUint32(sh[0:], names[s.name])
		binary.LittleEndian.PutUint32(sh[4:], s.typ)
		binary.LittleEndian.PutUint64(sh[8:], 2 /*ALLOC*/)
		binary.LittleEndian.PutUint64(sh[16:], s.addr)
		binary.LittleEndian.PutUint64(sh[24:], offs[i])
		binary.LittleEndian.PutUint64(sh[32:], uint64(len(s.body)))
		binary.LittleEndian.PutUint32(sh[40:], s.link)
		binary.LittleEndian.PutUint64(sh[56:], s.entsz)
		body.Write(sh[:])
	}
	raw := body.Bytes()
	copy(raw[0:], []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0})
	binary.LittleEndian.PutUint16(raw[16:], 2)                   // EXEC
	binary.LittleEndian.PutUint16(raw[18:], 0x3E)                // x86-64
	binary.LittleEndian.PutUint32(raw[20:], 1)                   // version
	binary.LittleEndian.PutUint64(raw[32:], 64)                  // phoff
	binary.LittleEndian.PutUint64(raw[40:], shoff)               // shoff
	binary.LittleEndian.PutUint16(raw[52:], 64)                  // ehsize
	binary.LittleEndian.PutUint16(raw[54:], 56)                  // phentsize
	binary.LittleEndian.PutUint16(raw[56:], 2)                   // phnum
	binary.LittleEndian.PutUint16(raw[58:], 64)                  // shentsize
	binary.LittleEndian.PutUint16(raw[60:], uint16(len(secs)+1)) // shnum
	binary.LittleEndian.PutUint16(raw[62:], uint16(len(secs)))   // shstrndx

	// A text PT_LOAD establishes the image base. A second, deliberately
	// isolated PT_LOAD carries only entry/stub bytes; its larger p_memsz keeps
	// the removed suffix mapped as zero-fill after compaction.
	writeLoad := func(h, off, vaddr, filesz, memsz uint64, flags uint32) {
		binary.LittleEndian.PutUint32(raw[h:], uint32(elf.PT_LOAD))
		binary.LittleEndian.PutUint32(raw[h+4:], flags)
		binary.LittleEndian.PutUint64(raw[h+8:], off)
		binary.LittleEndian.PutUint64(raw[h+16:], vaddr)
		binary.LittleEndian.PutUint64(raw[h+24:], vaddr)
		binary.LittleEndian.PutUint64(raw[h+32:], filesz)
		binary.LittleEndian.PutUint64(raw[h+40:], memsz)
		binary.LittleEndian.PutUint64(raw[h+48:], 1)
	}
	writeLoad(64, offs[0], secs[0].addr, uint64(len(secs[0].body)), uint64(len(secs[0].body)), 5)
	entryIndex, stubIndex := 1, 2
	carrierOff := offs[entryIndex]
	carrierEnd := offs[stubIndex] + uint64(len(secs[stubIndex].body))
	writeLoad(64+56, carrierOff, secs[entryIndex].addr, carrierEnd-carrierOff, 0x20000, 6)

	path := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(path, raw, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixtureFns() []elfFn {
	return []elfFn{
		{"example.com/p.A", 64},
		{"example.com/p.B", 64},
		{"__llgo_stub.example.com/p.A", 16},
	}
}

func fixtureEntry(addrOf func(string) uint64) []byte {
	out := rec(addrOf("example.com/p.A")+4, fnv64("example.com/p.A"))
	return append(out, rec(addrOf("example.com/p.B")+4, fnv64("example.com/p.B"))...)
}

func fixtureStub(addrOf func(string) uint64) []byte {
	return rec(addrOf("__llgo_stub.example.com/p.A")+4, fnv64("example.com/p.A"))
}

func TestRewriteELFInPlace(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 4096, 256)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.FtabEntries != 4 { // A, B, stub, sentinel
		t.Fatalf("stats %+v", st)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.BytesRemoved == 0 || before.Size()-after.Size() != int64(st.BytesRemoved) {
		t.Fatalf("physical shrink before=%d after=%d stats=%+v", before.Size(), after.Size(), st)
	}
	// Idempotence guard.
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected already-rewritten error")
	}
	// Adoptable header with a plain runtime base on non-PIE ELF.
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(info.entrySec[0:]); got != prebuiltMagic {
		t.Fatalf("magic %#x", got)
	}
	base := binary.LittleEndian.Uint64(info.entrySec[16:])
	if base != 0x10000 { // first function entry
		t.Fatalf("base %#x", base)
	}
	if info.entryVMSize >= 4096 || info.stubVMSize != 0 {
		t.Fatalf("section sizes entry=%#x stub=%#x", info.entryVMSize, info.stubVMSize)
	}
}

func TestRewriteELFSpillsToStubSection(t *testing.T) {
	// Entry section too small for the blob, stub section large enough.
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 0, 8192)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.FtabEntries != 4 {
		t.Fatalf("stats %+v", st)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.BytesRemoved == 0 || before.Size()-after.Size() != int64(st.BytesRemoved) {
		t.Fatalf("physical spill shrink before=%d after=%d stats=%+v", before.Size(), after.Size(), st)
	}
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(info.entrySec[0:]); got != redirectMagic {
		t.Fatalf("entry magic %#x", got)
	}
	if got := binary.LittleEndian.Uint64(info.stubSec[0:]); got != prebuiltMagic {
		t.Fatalf("stub magic %#x", got)
	}
	if ptr := binary.LittleEndian.Uint64(info.entrySec[16:]); ptr != info.stubVMAddr {
		t.Fatalf("redirect ptr %#x want %#x", ptr, info.stubVMAddr)
	}
}

func TestRewriteELFOverflowFallsBack(t *testing.T) {
	// Neither section fits: Rewrite must fail (no gap-y table).
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 0, 0)
	before, _ := os.ReadFile(path)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected overflow error")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("binary must be untouched on failure")
	}
}

func TestRewriteELFDedupShrinksInlineCopies(t *testing.T) {
	const copies = 512
	entry := func(addrOf func(string) uint64) []byte {
		out := fixtureEntry(addrOf)
		// These records carry B's ID inside A, exactly the shape produced when
		// Full LTO inlines B into A. Only B's canonical record may survive.
		for i := 0; i < copies; i++ {
			out = append(out, rec(addrOf("example.com/p.A")+4, fnv64("example.com/p.B"))...)
		}
		return out
	}
	path := buildELF(t, fixtureFns(), entry, fixtureStub, 65536, 4096)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.InlineCopies != copies || st.FtabEntries != 4 || st.BytesRemoved == 0 {
		t.Fatalf("dedup stats = %+v", st)
	}
	if before.Size()-after.Size() != int64(st.BytesRemoved) {
		t.Fatalf("dedup did not physically shrink: before=%d after=%d stats=%+v", before.Size(), after.Size(), st)
	}
	// Reopening and resolving symbols exercises the shifted symtab/strtab and
	// section-header offsets, not merely the compact table bytes.
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.syms) != len(fixtureFns()) {
		t.Fatalf("reopened symbols = %#v", info.syms)
	}
}

func TestCompactELFWithNoRemovableSuffix(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 64, 64)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, removed, err := compactCarrier(info.raw, info, info.entryVMSize, info.stubVMSize)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 || len(raw) != len(info.raw) {
		t.Fatalf("no-op compaction removed=%d sizes=%d/%d", removed, len(raw), len(info.raw))
	}
}

func TestCompactELFRejectsOutOfFileCarrier(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 4096, 256)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), info.raw...)
	// The second program header is the carrier PT_LOAD.
	binary.LittleEndian.PutUint64(raw[64+56+32:], uint64(len(raw)))
	if _, _, err := compactCarrier(raw, info, 64, 0); err == nil || !strings.Contains(err.Error(), "file range") {
		t.Fatalf("compact error = %v, want invalid file range", err)
	}
}

func TestCompactELFRejectsFollowingProgram(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 8192, 256)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := append([]byte(nil), info.raw...)
	shoff := binary.LittleEndian.Uint64(raw[40:])
	binary.LittleEndian.PutUint16(raw[56:], 3)
	h := uint64(64 + 2*56)
	binary.LittleEndian.PutUint32(raw[h:], uint32(elf.PT_NOTE))
	binary.LittleEndian.PutUint64(raw[h+8:], shoff)
	binary.LittleEndian.PutUint64(raw[h+16:], shoff%0x2000)
	binary.LittleEndian.PutUint64(raw[h+48:], 0x2000)
	if _, _, err := compactCarrier(raw, info, 64, 0); err == nil || !strings.Contains(err.Error(), "follows") {
		t.Fatalf("compact error = %v, want following-program rejection", err)
	}
}

func TestParseELFLayoutRejectsOverflowingTables(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, fixtureStub, 256, 64)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []int{32, 40} {
		bad := append([]byte(nil), raw...)
		binary.LittleEndian.PutUint64(bad[field:], ^uint64(0)-32)
		if _, _, _, err := parseELFLayout(bad); err == nil {
			t.Fatalf("parse accepted overflowing table at header field %d", field)
		}
	}
	bad := append([]byte(nil), raw...)
	shoff := binary.LittleEndian.Uint64(bad[40:])
	shentsz := uint64(binary.LittleEndian.Uint16(bad[58:]))
	shstrndx := uint64(binary.LittleEndian.Uint16(bad[62:]))
	shstr := shoff + shstrndx*shentsz
	binary.LittleEndian.PutUint64(bad[shstr+24:], ^uint64(0)-16)
	if _, _, _, err := parseELFLayout(bad); err == nil || !strings.Contains(err.Error(), "section-name") {
		t.Fatalf("parse section-name error = %v", err)
	}
}

func TestRewriteErrorPaths(t *testing.T) {
	// No entry records at all.
	empty := func(addrOf func(string) uint64) []byte { return nil }
	path := buildELF(t, fixtureFns(), empty, empty, 4096, 256)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected no-entry-records error")
	}
	// Records whose anchors have no owning symbol: dropped, nothing survives.
	orphan := func(addrOf func(string) uint64) []byte {
		return rec(0xdead0000, 42)
	}
	path = buildELF(t, fixtureFns(), orphan, empty, 4096, 256)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected no-survivors error")
	}
}

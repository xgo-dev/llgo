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
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// buildELF fabricates the minimal ELF load() understands: .text, the funcinfo
// entry-site section, a data section holding the symbol index, .symtab
// and .strtab. Layout is one flat file segment; vmaddr == file offset + 0x10000.
type elfFn struct {
	name string
	size uint64
}

func buildELF(t *testing.T, fns []elfFn, entryRecs func(addrOf func(string) uint64) []byte, entryPad int) string {
	return buildELFExternal(t, fns, entryRecs, entryPad, nil, nil)
}

func buildELFExternal(t *testing.T, fns []elfFn, entryRecs func(addrOf func(string) uint64) []byte, entryPad int, pcLine, identity []byte) string {
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
	idxTableAddr := base + 0x8000
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

	sectionNames := []string{".text", "llgo_funcinfo_entry"}
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
	body.Write(make([]byte, 64)) // ELF header placeholder
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
	binary.LittleEndian.PutUint64(raw[40:], shoff)               // shoff
	binary.LittleEndian.PutUint16(raw[52:], 64)                  // ehsize
	binary.LittleEndian.PutUint16(raw[58:], 64)                  // shentsize
	binary.LittleEndian.PutUint16(raw[60:], uint16(len(secs)+1)) // shnum
	binary.LittleEndian.PutUint16(raw[62:], uint16(len(secs)))   // shstrndx

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
	}
}

func fixtureEntry(addrOf func(string) uint64) []byte {
	out := rec(addrOf("example.com/p.A")+4, fnv64("example.com/p.A"))
	return append(out, rec(addrOf("example.com/p.B")+4, fnv64("example.com/p.B"))...)
}

func TestRewriteELFInPlace(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, 4096)
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.FtabEntries != 3 { // A, B, sentinel
		t.Fatalf("stats %+v", st)
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
}

func TestRewriteELFOverflowFallsBack(t *testing.T) {
	path := buildELF(t, fixtureFns(), fixtureEntry, 0)
	before, _ := os.ReadFile(path)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected overflow error")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("binary must be untouched on failure")
	}
}

func TestRewriteErrorPaths(t *testing.T) {
	// No entry records at all.
	empty := func(addrOf func(string) uint64) []byte { return nil }
	path := buildELF(t, fixtureFns(), empty, 4096)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected no-entry-records error")
	}
	// Records whose anchors have no owning symbol: dropped, nothing survives.
	orphan := func(addrOf func(string) uint64) []byte {
		return rec(0xdead0000, 42)
	}
	path = buildELF(t, fixtureFns(), orphan, 4096)
	if _, err := Rewrite(path); err == nil {
		t.Fatal("expected no-survivors error")
	}
}

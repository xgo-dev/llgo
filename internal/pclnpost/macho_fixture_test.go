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

// buildMachO fabricates a minimal 64-bit Mach-O that debug/macho can Open:
// one __TEXT segment (__text) and one __DATA segment carrying __llgo_fie /
// optional pcline data, plus LC_SYMTAB and an LC_DYLD_CHAINED_FIXUPS whose imports
// table binds ordinal 1 to a local symbol.
func buildMachO(t *testing.T, entry []byte, syms []elfFn) string {
	return buildMachOExternal(t, entry, nil, nil, syms)
}

func buildMachOExternal(t *testing.T, entry, pcLine, identity []byte, syms []elfFn) string {
	t.Helper()
	const base = uint64(0x100000000)
	text := make([]byte, 0x40000) // big enough that findfunctab buckets outgrow a tiny entry section
	type lc struct{ b []byte }
	var cmds []lc

	sect := func(name, seg string, addr, off, size uint64) []byte {
		var b [80]byte
		copy(b[0:], name)
		copy(b[16:], seg)
		binary.LittleEndian.PutUint64(b[32:], addr)
		binary.LittleEndian.PutUint64(b[40:], size)
		binary.LittleEndian.PutUint32(b[48:], uint32(off))
		return b[:]
	}
	segment := func(name string, vmaddr, fileoff, filesz uint64, sects [][]byte) []byte {
		var h [72]byte
		binary.LittleEndian.PutUint32(h[0:], 0x19) // LC_SEGMENT_64
		binary.LittleEndian.PutUint32(h[4:], uint32(72+80*len(sects)))
		copy(h[8:], name)
		binary.LittleEndian.PutUint64(h[24:], vmaddr)
		binary.LittleEndian.PutUint64(h[32:], filesz)
		binary.LittleEndian.PutUint64(h[40:], fileoff)
		binary.LittleEndian.PutUint64(h[48:], filesz)
		binary.LittleEndian.PutUint32(h[64:], uint32(len(sects)))
		out := h[:]
		for _, s := range sects {
			out = append(out, s...)
		}
		return out
	}

	// File layout (fixed offsets, one page apart).
	const textOff = uint64(0x1000)
	const entryOff = uint64(0x2000)
	dataEnd := entryOff + uint64(len(entry))
	pcLineOff := dataEnd
	if pcLine != nil {
		dataEnd += uint64(len(pcLine))
	}
	identityOff := dataEnd
	if identity != nil {
		dataEnd += uint64(len(identity))
	}
	symOff := (dataEnd + 0xF) &^ 0xF
	fixOff := symOff + 0x800

	// A fileless low segment mirrors __PAGEZERO in real executables. Image
	// base discovery must ignore it and choose file-backed __TEXT instead.
	cmds = append(cmds, lc{segment("__PAGEZERO", 0, 0, 0, nil)})
	cmds = append(cmds, lc{segment("__TEXT", base, 0, textOff+uint64(len(text)), [][]byte{
		sect("__text", "__TEXT", base+textOff, textOff, uint64(len(text))),
	})})
	dataSections := [][]byte{
		sect("__llgo_fie", "__DATA", base+entryOff, entryOff, uint64(len(entry))),
	}
	if pcLine != nil {
		dataSections = append(dataSections, sect("__llgo_pcl", "__DATA", base+pcLineOff, pcLineOff, uint64(len(pcLine))))
	}
	if identity != nil {
		dataSections = append(dataSections, sect("__llgo_pid", "__DATA", base+identityOff, identityOff, uint64(len(identity))))
	}
	cmds = append(cmds, lc{segment("__DATA", base+entryOff, entryOff, dataEnd-entryOff, dataSections)})

	// Symtab: nlist_64 entries + strtab.
	strtab := []byte{0}
	var nlist bytes.Buffer
	for _, fn := range syms {
		nameOff := len(strtab)
		strtab = append(strtab, "_"+fn.name...)
		strtab = append(strtab, 0)
		binary.Write(&nlist, binary.LittleEndian, uint32(nameOff))
		nlist.WriteByte(0x0F) // N_SECT|N_EXT
		nlist.WriteByte(1)    // __text
		binary.Write(&nlist, binary.LittleEndian, uint16(0))
		binary.Write(&nlist, binary.LittleEndian, base+textOff+fn.size) // addr encoded via size field as offset
	}
	strOff := symOff + uint64(nlist.Len())
	var symtabCmd [24]byte
	binary.LittleEndian.PutUint32(symtabCmd[0:], 0x2) // LC_SYMTAB
	binary.LittleEndian.PutUint32(symtabCmd[4:], 24)
	binary.LittleEndian.PutUint32(symtabCmd[8:], uint32(symOff))
	binary.LittleEndian.PutUint32(symtabCmd[12:], uint32(len(syms)))
	binary.LittleEndian.PutUint32(symtabCmd[16:], uint32(strOff))
	binary.LittleEndian.PutUint32(symtabCmd[20:], uint32(len(strtab)))
	cmds = append(cmds, lc{symtabCmd[:]})

	// Chained fixups: header + starts(no pages) + one import (ordinal 0
	// unused, ordinal 1 -> first symbol) + names.
	var fx bytes.Buffer
	fxHdr := make([]byte, 28)
	binary.LittleEndian.PutUint32(fxHdr[4:], 28) // starts_offset
	fx.Write(fxHdr)
	// starts_in_image: seg_count=3; __PAGEZERO and __TEXT have no chains,
	// __DATA gets a
	// starts_in_segment whose pages are all "no chain yet" so the rewriter
	// can splice its live-relocation inserts.
	binary.Write(&fx, binary.LittleEndian, uint32(3))
	binary.Write(&fx, binary.LittleEndian, uint32(0))
	binary.Write(&fx, binary.LittleEndian, uint32(0))
	segInfoOff := 16 // seg_count + three offsets, then this struct
	binary.Write(&fx, binary.LittleEndian, uint32(segInfoOff))
	// pad to seg_info start (nothing between)
	dataPages := int((dataEnd-entryOff)/0x4000 + 1)
	segInfo := make([]byte, 22+2*dataPages)
	binary.LittleEndian.PutUint32(segInfo[0:], uint32(len(segInfo)))
	binary.LittleEndian.PutUint16(segInfo[4:], 0x4000) // page_size
	binary.LittleEndian.PutUint16(segInfo[6:], 2)      // DYLD_CHAINED_PTR_64
	binary.LittleEndian.PutUint64(segInfo[8:], entryOff)
	binary.LittleEndian.PutUint16(segInfo[20:], uint16(dataPages))
	for i := 0; i < dataPages; i++ {
		binary.LittleEndian.PutUint16(segInfo[22+2*i:], chainedPtrStartNone)
	}
	fx.Write(segInfo)
	importsOff := fx.Len()
	names := []byte{0}
	addImport := func(sym string) {
		no := len(names)
		names = append(names, "_"+sym...)
		names = append(names, 0)
		binary.Write(&fx, binary.LittleEndian, uint32(no)<<9)
		binary.Write(&fx, binary.LittleEndian, int32(0))
	}
	addImport("missing.symbol")
	if len(syms) > 0 {
		addImport(syms[0].name)
	}
	symbolsOff := fx.Len()
	fx.Write(names)
	blob := fx.Bytes()
	binary.LittleEndian.PutUint32(blob[8:], uint32(importsOff))
	binary.LittleEndian.PutUint32(blob[12:], uint32(symbolsOff))
	binary.LittleEndian.PutUint32(blob[16:], 2) // imports_count
	binary.LittleEndian.PutUint32(blob[20:], 2) // DYLD_CHAINED_IMPORT_ADDEND

	var fixCmd [16]byte
	binary.LittleEndian.PutUint32(fixCmd[0:], 0x80000034)
	binary.LittleEndian.PutUint32(fixCmd[4:], 16)
	binary.LittleEndian.PutUint32(fixCmd[8:], uint32(fixOff))
	binary.LittleEndian.PutUint32(fixCmd[12:], uint32(len(blob)))
	cmds = append(cmds, lc{fixCmd[:]})

	var cmdBytes []byte
	for _, c := range cmds {
		cmdBytes = append(cmdBytes, c.b...)
	}
	total := fixOff + uint64(len(blob))
	raw := make([]byte, total)
	binary.LittleEndian.PutUint32(raw[0:], 0xFEEDFACF)
	binary.LittleEndian.PutUint32(raw[4:], 0x0100000C) // CPU_TYPE_ARM64
	binary.LittleEndian.PutUint32(raw[8:], 0)
	binary.LittleEndian.PutUint32(raw[12:], 2) // MH_EXECUTE
	binary.LittleEndian.PutUint32(raw[16:], uint32(len(cmds)))
	binary.LittleEndian.PutUint32(raw[20:], uint32(len(cmdBytes)))
	copy(raw[32:], cmdBytes)
	copy(raw[textOff:], text)
	copy(raw[entryOff:], entry)
	if pcLine != nil {
		copy(raw[pcLineOff:], pcLine)
	}
	if identity != nil {
		copy(raw[identityOff:], identity)
	}
	copy(raw[symOff:], nlist.Bytes())
	copy(raw[strOff:], strtab)
	copy(raw[fixOff:], blob)

	path := filepath.Join(t.TempDir(), "macho")
	if err := os.WriteFile(path, raw, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMachOFixture(t *testing.T) {
	const base = uint64(0x100000000)
	fns := []elfFn{{name: "example.com/p.A", size: 0x10}, {name: "example.com/p.B", size: 0x40}}
	// Entry records: one rebase-encoded (chain bits above bit 36), one bind
	// to import ordinal 1 (= example.com/p.A) with addend 4, one bind to the
	// unresolved ordinal 0 that must be dropped.
	rebase := (uint64(7) << 51) | (base + 0x1000 + 0x44)
	bind := (uint64(1) << 63) | (uint64(4) << 24) | 1
	badBind := uint64(1) << 63
	entry := append(rec(rebase, fnv64("example.com/p.B")), rec(bind, fnv64("example.com/p.A"))...)
	entry = append(entry, rec(badBind, 99)...)
	path := buildMachO(t, entry, fns)
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.format != "macho" {
		t.Fatalf("format %s", info.format)
	}
	if info.textStart != base+0x1000 || info.entryVMAddr != base+0x2000 {
		t.Fatalf("layout %#x %#x", info.textStart, info.entryVMAddr)
	}
	if len(info.bindTargets) != 2 || info.bindTargets[0] != 0 || info.bindTargets[1] != base+0x1000+fns[0].size {
		t.Fatalf("bindTargets %#v", info.bindTargets)
	}
	recs := parseRecords(info, info.entrySec)
	if len(recs) != 2 {
		t.Fatalf("records %+v", recs)
	}
	if recs[0].pc != base+0x1000+0x44 {
		t.Fatalf("rebase pc %#x", recs[0].pc)
	}
	if recs[1].pc != info.bindTargets[1]+4 {
		t.Fatalf("bind pc %#x", recs[1].pc)
	}
}

// machoRewriteFixture: records anchored inside real text symbols so dedupe
// keeps them, plus a meta record advertising the symbol index inside the
// entry section (readVM resolves it through the __DATA section).
func machoRewriteFixture(t *testing.T, entryPad int) string {
	t.Helper()
	const base = uint64(0x100000000)
	fns := []elfFn{{name: "example.com/p.A", size: 0x10}, {name: "example.com/p.B", size: 0x3F000}} // far apart: findfunctab spans many buckets
	aAddr := base + 0x1000 + fns[0].size                                                            // buildMachO uses size as text offset
	bAddr := base + 0x1000 + fns[1].size
	idA, idB := fnv64("example.com/p.A"), fnv64("example.com/p.B")

	// Symbol index table + pointer/count globals live in the entry section
	// tail so readVM can reach them via the __llgo_fie section.
	entryBase := base + 0x2000
	var idx []byte
	type sie struct {
		id uint64
		i  uint32
	}
	ids := []sie{{idA, 1}, {idB, 2}}
	if ids[0].id > ids[1].id {
		ids[0], ids[1] = ids[1], ids[0]
	}
	for _, e := range ids {
		var b [16]byte
		binary.LittleEndian.PutUint64(b[0:], e.id)
		binary.LittleEndian.PutUint32(b[8:], e.i)
		idx = append(idx, b[:]...)
	}
	// entry layout: meta(3 recs) + 2 records + [idx table][ptr][cnt] + pad
	recs := append(rec(0, metaRecordMagic), rec(0, 0)...) // ptr filled below
	recs = append(recs, rec(0, 0)...)
	recs = append(recs, rec(aAddr+4, idA)...)
	recs = append(recs, rec(bAddr+4, idB)...)
	idxAddr := entryBase + uint64(len(recs))
	ptrAddr := idxAddr + uint64(len(idx))
	cntAddr := ptrAddr + 8
	var tail []byte
	tail = append(tail, idx...)
	var p8 [8]byte
	binary.LittleEndian.PutUint64(p8[:], idxAddr)
	tail = append(tail, p8[:]...)
	binary.LittleEndian.PutUint64(p8[:], 2)
	tail = append(tail, p8[:]...)
	entry := append(recs, tail...)
	// backpatch meta rows 2/3 with ptr/cnt addresses
	binary.LittleEndian.PutUint64(entry[16:], ptrAddr)
	binary.LittleEndian.PutUint64(entry[32:], cntAddr)
	entry = append(entry, make([]byte, entryPad)...)
	return buildMachO(t, entry, fns)
}

func TestRewriteMachOInPlace(t *testing.T) {
	path := machoRewriteFixture(t, 4096)
	st, err := Rewrite(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Format != "macho" || st.FtabEntries != 3 {
		t.Fatalf("stats %+v", st)
	}
	info, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(info.entrySec[0:]); got != prebuiltMagic {
		t.Fatalf("magic %#x", got)
	}
}

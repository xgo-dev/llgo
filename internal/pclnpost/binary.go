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

// Package pclnpost implements the P1/P2 prototype of link-phase ftab/findfunctab
// generation (doc/design/pclntab-linkphase.md). It parses a linked LLGo
// binary's funcinfo site section, deduplicates LTO inline copies against the
// symbol table, sorts the entries, builds the Go-layout findfunctab via
// internal/pclntab, and transactionally replaces the linked image. Isolated
// LTO carriers in embedded Mach-O executables are physically compacted; shared
// carriers in other Mach-O build modes retain their file layout.
package pclnpost

import (
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/xgo-dev/llgo/internal/pclntab"
)

type siteRecord struct {
	pc       uint64
	symbolID uint64
}

type textSym struct {
	addr uint64
	size uint64
	name string
}

type secInfo struct {
	vmaddr  uint64
	size    uint64
	fileOff uint64
}

type binaryInfo struct {
	format       string
	raw          []byte
	entrySec     []byte
	pcLineSec    []byte
	textStart    uint64
	textEnd      uint64
	imageBase    uint64
	pointerSize  int
	littleEndian bool
	syms         []textSym // sorted by addr, text symbols only
	secs         []secInfo
	// Mach-O chained-fixup import targets, ordinal -> resolved vmaddr (0 if
	// the import name has no local definition). Exported symbols' pointer
	// slots are emitted as BIND nodes even when they bind to this image, so
	// record decoding needs the imports table, not just rebase decoding.
	bindTargets []uint64

	entryVMAddr, entryVMSize, entryFileOff uint64
	pcLineVMSize, pcLineFileOff            uint64
	identityVMSize, identityFileOff        uint64
	hasCodeSignature                       bool
}

// readVM returns n bytes at a link-time virtual address.
func readVM(info *binaryInfo, addr uint64, n int) []byte {
	for _, s := range info.secs {
		if addr >= s.vmaddr && addr+uint64(n) <= s.vmaddr+s.size {
			off := s.fileOff + (addr - s.vmaddr)
			return info.raw[off : off+uint64(n)]
		}
	}
	return make([]byte, n)
}

func load(path string) (*binaryInfo, error) {
	if mf, err := macho.Open(path); err == nil {
		defer mf.Close()
		pointerSize := 4
		if mf.Magic == macho.Magic64 {
			pointerSize = 8
		}
		info := &binaryInfo{format: "macho", pointerSize: pointerSize, littleEndian: mf.ByteOrder == binary.LittleEndian}
		var err error
		info.raw, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		imageBase := ^uint64(0)
		for _, l := range mf.Loads {
			if seg, ok := l.(*macho.Segment); ok && seg.Filesz != 0 && seg.Addr < imageBase {
				imageBase = seg.Addr
			}
		}
		if imageBase != ^uint64(0) {
			info.imageBase = imageBase
		}
		for _, s := range mf.Sections {
			if sectionInFile(info.raw, uint64(s.Offset), s.Size) {
				info.secs = append(info.secs, secInfo{vmaddr: s.Addr, size: s.Size, fileOff: uint64(s.Offset)})
			}
		}
		if s := mf.Section("__llgo_fie"); s != nil {
			info.entrySec, err = sectionBytes(info.raw, uint64(s.Offset), s.Size)
			if err != nil {
				return nil, fmt.Errorf("Mach-O __llgo_fie: %w", err)
			}
			info.entryVMAddr, info.entryVMSize, info.entryFileOff = s.Addr, s.Size, uint64(s.Offset)
		}
		if s := mf.Section("__llgo_pcl"); s != nil {
			info.pcLineSec, err = sectionBytes(info.raw, uint64(s.Offset), s.Size)
			if err != nil {
				return nil, fmt.Errorf("Mach-O __llgo_pcl: %w", err)
			}
			info.pcLineVMSize, info.pcLineFileOff = s.Size, uint64(s.Offset)
		}
		if s := mf.Section("__llgo_pid"); s != nil {
			if _, err := sectionBytes(info.raw, uint64(s.Offset), s.Size); err != nil {
				return nil, fmt.Errorf("Mach-O __llgo_pid: %w", err)
			}
			info.identityVMSize, info.identityFileOff = s.Size, uint64(s.Offset)
		}
		if s := mf.Section("__text"); s != nil {
			info.textStart, info.textEnd = s.Addr, s.Addr+s.Size
		}
		if mf.Symtab != nil {
			for _, sym := range mf.Symtab.Syms {
				if sym.Value >= info.textStart && sym.Value < info.textEnd && sym.Name != "" {
					info.syms = append(info.syms, textSym{addr: sym.Value, name: sym.Name})
				}
			}
		}
		for _, l := range mf.Loads {
			if b := l.Raw(); len(b) >= 4 && binary.LittleEndian.Uint32(b) == 0x1D { // LC_CODE_SIGNATURE
				info.hasCodeSignature = true
			}
		}
		if pointerSize == 8 && info.littleEndian {
			loadBindTargets(info, mf)
		}
		finish(info)
		return info, nil
	}
	ef, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("not Mach-O and not ELF: %w", err)
	}
	defer ef.Close()
	info := &binaryInfo{format: "elf", littleEndian: ef.ByteOrder == binary.LittleEndian}
	if ef.Class == elf.ELFCLASS64 {
		info.pointerSize = 8
	} else if ef.Class == elf.ELFCLASS32 {
		info.pointerSize = 4
	}
	info.raw, err = os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	imageBase := ^uint64(0)
	for _, p := range ef.Progs {
		if p.Type == elf.PT_LOAD && p.Filesz != 0 && p.Vaddr < imageBase {
			imageBase = p.Vaddr
		}
	}
	if imageBase != ^uint64(0) {
		info.imageBase = imageBase
	}
	for _, s := range ef.Sections {
		if s.Type != elf.SHT_NOBITS && s.Addr != 0 && sectionInFile(info.raw, s.Offset, s.Size) {
			info.secs = append(info.secs, secInfo{vmaddr: s.Addr, size: s.Size, fileOff: s.Offset})
		}
	}
	// Old pclnpost fixtures predate program headers. Retain a conservative
	// fallback for such files; real executables always take the PT_LOAD path
	// above, which deliberately ignores fileless PT_LOAD segments.
	if imageBase == ^uint64(0) {
		for _, s := range info.secs {
			if s.size != 0 && s.vmaddr < imageBase {
				imageBase = s.vmaddr
			}
		}
		if imageBase != ^uint64(0) {
			info.imageBase = imageBase
		}
	}
	if s := ef.Section("llgo_funcinfo_entry"); s != nil {
		info.entrySec, err = sectionBytes(info.raw, s.Offset, s.Size)
		if err != nil {
			return nil, fmt.Errorf("ELF llgo_funcinfo_entry: %w", err)
		}
		info.entryVMAddr, info.entryVMSize, info.entryFileOff = s.Addr, s.Size, s.Offset
	}
	if s := ef.Section("llgo_pcline"); s != nil {
		info.pcLineSec, err = sectionBytes(info.raw, s.Offset, s.Size)
		if err != nil {
			return nil, fmt.Errorf("ELF llgo_pcline: %w", err)
		}
		info.pcLineVMSize, info.pcLineFileOff = s.Size, s.Offset
	}
	if s := ef.Section("llgo_pclntab_id"); s != nil {
		if _, err := sectionBytes(info.raw, s.Offset, s.Size); err != nil {
			return nil, fmt.Errorf("ELF llgo_pclntab_id: %w", err)
		}
		info.identityVMSize, info.identityFileOff = s.Size, s.Offset
	}
	if s := ef.Section(".text"); s != nil {
		info.textStart, info.textEnd = s.Addr, s.Addr+s.Size
	}
	syms, _ := ef.Symbols()
	for _, sym := range syms {
		if elf.ST_TYPE(sym.Info) == elf.STT_FUNC && sym.Value >= info.textStart && sym.Value < info.textEnd {
			info.syms = append(info.syms, textSym{addr: sym.Value, size: sym.Size, name: sym.Name})
		}
	}
	finish(info)
	return info, nil
}

func sectionInFile(raw []byte, off, size uint64) bool {
	return off <= uint64(len(raw)) && size <= uint64(len(raw))-off
}

func sectionBytes(raw []byte, off, size uint64) ([]byte, error) {
	if !sectionInFile(raw, off, size) {
		return nil, fmt.Errorf("section range [%#x,%#x) is outside file size %#x", off, off+size, len(raw))
	}
	return raw[off : off+size], nil
}

func finish(info *binaryInfo) {
	sort.Slice(info.syms, func(i, j int) bool { return info.syms[i].addr < info.syms[j].addr })
	// Collapse same-address aliases, then derive missing extents from the
	// next distinct symbol start (Mach-O nlist carries no sizes; Go's linker
	// uses the same next-start rule for its final ftab).
	dedup := info.syms[:0]
	for _, s := range info.syms {
		if len(dedup) > 0 && dedup[len(dedup)-1].addr == s.addr {
			continue
		}
		dedup = append(dedup, s)
	}
	info.syms = dedup
	for i := range info.syms {
		if info.syms[i].size == 0 {
			if i+1 < len(info.syms) {
				info.syms[i].size = info.syms[i+1].addr - info.syms[i].addr
			} else {
				info.syms[i].size = info.textEnd - info.syms[i].addr
			}
		}
	}
}

func parseRecords(info *binaryInfo, sec []byte) []siteRecord {
	var out []siteRecord
	for off := 0; off+16 <= len(sec); off += 16 {
		pc := binary.LittleEndian.Uint64(sec[off:])
		id := binary.LittleEndian.Uint64(sec[off+8:])
		if pc == 0 || id == 0 { // zero keep-alive record
			continue
		}
		// Mach-O pointer slots in the on-disk file hold dyld chained-fixup
		// encodings; dyld rewrites them at load. Rebase nodes
		// (DYLD_CHAINED_PTR_64) carry the target in the low 36 bits. Anchors
		// naming exported functions are emitted as BIND nodes instead (bit 63
		// set, import ordinal in the low 24 bits, addend above), even though
		// they bind back into this same image, so those resolve through the
		// imports table. The P2 write-back avoids the problem entirely by
		// storing anchor-relative offsets instead of pointers.
		if info.format == "macho" && (pc < info.textStart || pc >= info.textEnd) {
			if pc>>63 != 0 { // DYLD_CHAINED_PTR_64_BIND
				ordinal := pc & (1<<24 - 1)
				addend := (pc >> 24) & 0xFF
				if ordinal >= uint64(len(info.bindTargets)) || info.bindTargets[ordinal] == 0 {
					continue
				}
				pc = info.bindTargets[ordinal] + addend
			} else if t := pc & (1<<36 - 1); t >= info.textStart && t < info.textEnd {
				pc = t
			}
		}
		out = append(out, siteRecord{pc: pc, symbolID: id})
	}
	return out
}

// owner returns the text symbol containing addr.
func owner(info *binaryInfo, addr uint64) (textSym, bool) {
	i := sort.Search(len(info.syms), func(i int) bool { return info.syms[i].addr > addr })
	if i == 0 {
		return textSym{}, false
	}
	s := info.syms[i-1]
	if addr >= s.addr+s.size {
		return textSym{}, false
	}
	return s, true
}

// fnv64 mirrors funcInfoSymbolID in internal/build/funcinfo_table.go.
func fnv64(name string) uint64 {
	const offset = uint64(14695981039346656037)
	const prime = uint64(1099511628211)
	h := offset
	for i := 0; i < len(name); i++ {
		h ^= uint64(name[i])
		h *= prime
	}
	if h == 0 {
		return 1
	}
	return h
}

// canonicalOwner reports whether owner symbol `name` is the function the
// record's symbolID names.
// Mach-O symbol names carry a C-mangling underscore, and debug/macho's
// suffix-shared string table can surface one underscore more or less than
// the source-level name, so try each plausible normalization — matching a
// specific 64-bit FNV makes a false positive practically impossible.
func canonicalOwner(info *binaryInfo, name string, symbolID uint64) bool {
	for {
		if fnv64(name) == symbolID {
			return true
		}
		if info.format == "macho" && len(name) > 1 && name[0] == '_' {
			name = name[1:]
			continue
		}
		return false
	}
}

// dedupe keeps exactly the canonical record per emitting function: a record
// is canonical when the symbol that owns its anchor PC is the function the
// symbolID names (id == fnv64(owner)). Everything else with a known owner is
// an LTO inline copy: inlining duplicated the body-embedded record into a host
// function. A compiler-generated wrapper or adapter is therefore represented
// by its own funcinfo symbol and follows exactly the same path as any other
// function.
// Kept records are normalized to their owner's true entry address. Records
// whose owner cannot be determined are dropped conservatively.
func dedupe(info *binaryInfo, recs []siteRecord, verbose bool) (kept []siteRecord, droppedInline, droppedUnknown int) {
	seenOwner := make(map[uint64]bool, len(recs))
	for _, r := range recs {
		sym, ok := owner(info, r.pc)
		if !ok {
			droppedUnknown++
			continue
		}
		if !canonicalOwner(info, sym.name, r.symbolID) {
			droppedInline++
			if verbose {
				fmt.Printf("  inline copy: id=%#x pc=%#x inside %s\n", r.symbolID, r.pc, sym.name)
			}
			continue
		}
		if seenOwner[sym.addr] {
			continue
		}
		seenOwner[sym.addr] = true
		kept = append(kept, siteRecord{pc: sym.addr, symbolID: r.symbolID})
	}
	return kept, droppedInline, droppedUnknown
}

// buildFtab returns the sorted table plus the base PC (Go's minpc): offsets
// are relative to the first recorded function so ftab[0].EntryOff == 0, as
// internal/pclntab requires.
func buildFtab(info *binaryInfo, kept []siteRecord) ([]pclntab.FuncTabEntry, uint64) {
	sort.Slice(kept, func(i, j int) bool { return kept[i].pc < kept[j].pc })
	if len(kept) == 0 {
		return nil, info.textStart
	}
	base := kept[0].pc
	ftab := make([]pclntab.FuncTabEntry, 0, len(kept)+1)
	prev := uint64(0)
	for i, r := range kept {
		if r.pc == prev {
			continue // two symbolIDs at one entry (aliases); keep first
		}
		prev = r.pc
		ftab = append(ftab, pclntab.FuncTabEntry{EntryOff: uint32(r.pc - base), FuncOff: uint32(i)})
	}
	// Go-style sentinel at end of text.
	ftab = append(ftab, pclntab.FuncTabEntry{EntryOff: uint32(info.textEnd - base), FuncOff: ^uint32(0)})
	return ftab, base
}

// loadBindTargets parses the LC_DYLD_CHAINED_FIXUPS imports table and
// resolves each import ordinal to the address of its local definition (this
// is a main executable: every funcinfo bind target is defined in-image).
func loadBindTargets(info *binaryInfo, mf *macho.File) {
	raw := info.raw
	if len(raw) < 32 {
		return
	}
	ncmds := binary.LittleEndian.Uint32(raw[16:])
	var fixOff, fixSize uint64
	off := uint64(32)
	for i := uint32(0); i < ncmds && off+8 <= uint64(len(raw)); i++ {
		cmd := binary.LittleEndian.Uint32(raw[off:])
		size := binary.LittleEndian.Uint32(raw[off+4:])
		if size < 8 {
			return
		}
		if cmd == lcDyldChainedFixups {
			if off+16 > uint64(len(raw)) {
				return
			}
			fixOff = uint64(binary.LittleEndian.Uint32(raw[off+8:]))
			fixSize = uint64(binary.LittleEndian.Uint32(raw[off+12:]))
		}
		off += uint64(size)
	}
	if fixOff == 0 || fixSize < 28 || fixOff > uint64(len(raw)) || fixSize > uint64(len(raw))-fixOff {
		return
	}
	hdr := raw[fixOff : fixOff+fixSize]
	importsOff := binary.LittleEndian.Uint32(hdr[8:])
	symbolsOff := binary.LittleEndian.Uint32(hdr[12:])
	importsCount := binary.LittleEndian.Uint32(hdr[16:])
	importsFormat := binary.LittleEndian.Uint32(hdr[20:])
	if importsCount == 0 || importsCount > 1<<24 {
		return
	}
	var stride, nameShift uint32
	switch importsFormat {
	case 1: // DYLD_CHAINED_IMPORT: u32 {lib:8, weak:1, name_offset:23}
		stride, nameShift = 4, 9
	case 2: // DYLD_CHAINED_IMPORT_ADDEND: {u32, i32 addend}
		stride, nameShift = 8, 9
	default: // ADDEND64 or unknown: leave unresolved
		return
	}
	byName := make(map[string]uint64, len(info.syms))
	if mf.Symtab != nil {
		for _, sym := range mf.Symtab.Syms {
			if sym.Value != 0 && sym.Name != "" {
				byName[sym.Name] = sym.Value
			}
		}
	}
	cstr := func(b []byte) string {
		for i, c := range b {
			if c == 0 {
				return string(b[:i])
			}
		}
		return string(b)
	}
	targets := make([]uint64, importsCount)
	for i := uint32(0); i < importsCount; i++ {
		rec := uint64(importsOff) + uint64(i*stride)
		if rec+4 > uint64(len(hdr)) {
			break
		}
		v := binary.LittleEndian.Uint32(hdr[rec:])
		nameOff := uint64(symbolsOff) + uint64(v>>nameShift)
		if nameOff >= uint64(len(hdr)) {
			continue
		}
		name := cstr(hdr[nameOff:])
		addr, ok := byName[name]
		if !ok && len(name) > 1 && name[0] == '_' {
			// debug/macho's Symtab names may carry one less mangling
			// underscore than the import strings.
			addr = byName[name[1:]]
		}
		targets[i] = addr
	}
	info.bindTargets = targets
}

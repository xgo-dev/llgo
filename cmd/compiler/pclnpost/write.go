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
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// Prebuilt blob layout — keep in sync with runtime/internal/lib/runtime
// (runtimePrebuiltMagic and adoptPrebuiltFuncPCTable):
//
//	u64 magic "LLGOFTB1"; u64 linkSectAddr; u64 base
//	u32 count (incl sentinel); u32 bucketCount
//	count × {u32 entryOff, u32 funcIndex}
//	bucketCount × {u32 idx; 16 × u16 subbuckets}
//
// The base slot holds the runtime PC of the first table entry: on Mach-O it
// is re-linked into the dyld chained-fixup chain (dyld rebases it at load),
// on non-PIE ELF the link-time value already equals the runtime address.
const prebuiltMagic = uint64(0x314254464F474C4C)

const (
	prebuiltHeaderBytes = 32
	prebuiltEntryBytes  = 8
	bucketSize          = 4096
	subbucketCnt        = 16
	subbucketSize       = bucketSize / subbucketCnt
	bucketBytes         = 4 + 2*subbucketCnt
)

func prebuiltTableSize(entries, buckets uint64) uint64 {
	return prebuiltHeaderBytes + entries*prebuiltEntryBytes + buckets*bucketBytes
}

// SymbolEntryReserveSize bounds the encoded table, including its sentinel and
// buckets spanning foreign text between Go functions. The extra bucket covers
// an unaligned first PC; entry and text offsets use the runtime's uint32 format.
func SymbolEntryReserveSize(analysis ExternalAnalysis) (uint64, error) {
	if analysis.TextEnd <= analysis.TextStart ||
		analysis.TextEnd-analysis.TextStart > uint64(^uint32(0)) ||
		uint64(len(analysis.EntrySites)) >= uint64(^uint32(0)) {
		return 0, fmt.Errorf("invalid symbol entry count or text range")
	}
	buckets := (analysis.TextEnd-analysis.TextStart)/bucketSize + 2
	return prebuiltTableSize(uint64(len(analysis.EntrySites))+1, buckets), nil
}

type symIndexEntry struct {
	id  uint64
	idx uint32
}

// writeBack rewrites the entry-site prefix with the prebuilt table, removes
// the unused physical carrier tail, and publishes the staged image atomically.
func writeBack(path string, info *binaryInfo, kept []siteRecord) (ftabCount, bucketCount int, bytesRemoved uint64, err error) {
	symIdx, err := loadSymbolIndex(path, info)
	if err != nil {
		return 0, 0, 0, err
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].pc < kept[j].pc })
	type row struct {
		pc  uint64
		idx uint32
	}
	rows := make([]row, 0, len(kept))
	prev := uint64(0)
	for _, r := range kept {
		if r.pc == prev {
			continue
		}
		idx, ok := lookupSymIndex(symIdx, r.symbolID)
		if !ok {
			continue
		}
		prev = r.pc
		rows = append(rows, row{pc: r.pc, idx: idx})
	}
	if len(rows) == 0 {
		return 0, 0, 0, fmt.Errorf("no resolvable entries")
	}
	base := rows[0].pc
	count := len(rows) + 1 // + sentinel

	// findfunctab in the runtime's uint16 layout, mirroring
	// buildRuntimeFuncPCIndex (base aligned down to a bucket boundary).
	alignedBase := base &^ (bucketSize - 1)
	last := rows[len(rows)-1].pc
	nbuckets := int((last-alignedBase)/bucketSize + 1)
	pcs := make([]uint64, len(rows))
	for i, r := range rows {
		pcs[i] = r.pc
	}
	lastLE := func(pc uint64) int { // last index with pcs[i] <= pc, clamped like the runtime
		i := sort.Search(len(pcs), func(i int) bool { return pcs[i] > pc }) - 1
		if i < 0 {
			i = 0
		}
		return i
	}
	buckets := make([]byte, 0, nbuckets*bucketBytes)
	for b := 0; b < nbuckets; b++ {
		bucketStart := alignedBase + uint64(b)*bucketSize
		baseIdx := lastLE(bucketStart)
		var tmp [bucketBytes]byte
		binary.LittleEndian.PutUint32(tmp[0:], uint32(baseIdx))
		for s := 0; s < subbucketCnt; s++ {
			subIdx := lastLE(bucketStart + uint64(s)*subbucketSize)
			delta := subIdx - baseIdx
			if delta < 0 || delta > 0xffff {
				return 0, 0, 0, fmt.Errorf("subbucket delta overflow: %d", delta)
			}
			binary.LittleEndian.PutUint16(tmp[4+2*s:], uint16(delta))
		}
		buckets = append(buckets, tmp[:]...)
	}

	need := int(prebuiltTableSize(uint64(count), uint64(nbuckets)))
	if need > int(info.entryVMSize) {
		return 0, 0, 0, fmt.Errorf("prebuilt blob needs %d bytes; entry section has %d", need, info.entryVMSize)
	}
	blob := make([]byte, need)
	binary.LittleEndian.PutUint64(blob[0:], prebuiltMagic)
	binary.LittleEndian.PutUint64(blob[8:], info.entryVMAddr)
	binary.LittleEndian.PutUint64(blob[16:], base)
	binary.LittleEndian.PutUint32(blob[24:], uint32(count))
	binary.LittleEndian.PutUint32(blob[28:], uint32(len(buckets)/bucketBytes))
	off := prebuiltHeaderBytes
	for _, r := range rows {
		binary.LittleEndian.PutUint32(blob[off:], uint32(r.pc-base))
		binary.LittleEndian.PutUint32(blob[off+4:], r.idx)
		off += prebuiltEntryBytes
	}
	// Sentinel: end of text, funcIndex 0.
	binary.LittleEndian.PutUint32(blob[off:], uint32(info.textEnd-base))
	off += prebuiltEntryBytes
	copy(blob[off:], buckets)

	raw := make([]byte, len(info.raw))
	copy(raw, info.raw)
	var pending []pendingWrite
	if info.format == "macho" {
		// Remove the rewritten section's pointer slots from dyld's chained
		// fixup page chains first: otherwise dyld rebases 8-byte slots
		// inside the new table at load time.
		ranges := [][2]uint64{{info.entryFileOff, info.entryFileOff + info.entryVMSize}}
		// Pointer slots are spliced back into the chain as live rebase
		// nodes: dyld writes *slid* addresses at load, so the runtime reads
		// ready runtime pointers with no slide arithmetic.
		inserts := []fixupInsert{{fileOff: info.entryFileOff + 16, targetVM: base}}
		pending, err = unchainRanges(raw, ranges, inserts)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("chained fixups: %w", err)
		}
	}
	copy(raw[info.entryFileOff:info.entryFileOff+uint64(len(blob))], blob)
	for _, pw := range pending {
		binary.LittleEndian.PutUint64(raw[pw.fileOff:], pw.val)
	}
	raw, removed, err := compactCarrier(raw, info, uint64(need))
	if err != nil {
		return 0, 0, 0, err
	}
	verify := func(staged string) error {
		if removed != 0 {
			st, err := os.Stat(staged)
			if err != nil {
				return err
			}
			if st.Size() >= int64(len(info.raw)) {
				return fmt.Errorf("compacted file did not shrink: old=%d staged=%d", len(info.raw), st.Size())
			}
		}
		stagedInfo, err := load(staged)
		if err != nil {
			return fmt.Errorf("reload compact binary: %w", err)
		}
		if len(stagedInfo.entrySec) < 8 {
			return fmt.Errorf("compact entry section is truncated")
		}
		if got := binary.LittleEndian.Uint64(stagedInfo.entrySec); got != prebuiltMagic {
			return fmt.Errorf("compact entry magic %#x, want %#x", got, prebuiltMagic)
		}
		return nil
	}
	if err := replaceBinary(path, raw, info.format == ExternalFormatMachO && info.hasCodeSignature, verify); err != nil {
		return 0, 0, 0, err
	}
	return count, len(buckets) / bucketBytes, removed, nil
}

// metaRecordMagic marks the entry-section meta record ("LLGOMET1" LE); keep
// in sync with internal/build/funcinfo_table.go.
const metaRecordMagic = uint64(0x3154454D4F474C4C)

// metaGlobalAddrs scans the raw entry section for the meta record and
// returns the link-time addresses of the symbol-index pointer global and its
// count global. Works in +LTO binaries where the symbols are internalized
// away, because the addresses come from relocations, not the symbol table.
func metaGlobalAddrs(info *binaryInfo) (idxPtr, cntPtr uint64, ok bool) {
	sec := info.entrySec
	for off := 0; off+48 <= len(sec); off += 16 {
		pc := binary.LittleEndian.Uint64(sec[off:])
		id := binary.LittleEndian.Uint64(sec[off+8:])
		if pc == 0 && id == metaRecordMagic {
			idxPtr = decodePtrVal(info, binary.LittleEndian.Uint64(sec[off+16:]))
			cntPtr = decodePtrVal(info, binary.LittleEndian.Uint64(sec[off+32:]))
			return idxPtr, cntPtr, idxPtr != 0 && cntPtr != 0
		}
	}
	return 0, 0, false
}

// loadSymbolIndex reads the {u64 symbolID, u32 funcIndex} table, locating it
// through the entry-section meta record (LTO-safe) with the symbol table as
// fallback for older binaries.
func loadSymbolIndex(path string, info *binaryInfo) ([]symIndexEntry, error) {
	ptrAddr, cntAddr, ok := metaGlobalAddrs(info)
	if !ok {
		var err error
		ptrAddr, err = symbolAddr(path, "__llgo_funcinfo_symbol_index")
		if err != nil {
			return nil, err
		}
		cntAddr, err = symbolAddr(path, "__llgo_funcinfo_symbol_index_count")
		if err != nil {
			return nil, err
		}
	}
	dataAddr := decodePtr(info, readVM(info, ptrAddr, 8))
	count := binary.LittleEndian.Uint64(readVM(info, cntAddr, 8))
	if count == 0 || count > 1<<20 {
		return nil, fmt.Errorf("bad symbol index count %d", count)
	}
	raw := readVM(info, dataAddr, int(count)*16)
	out := make([]symIndexEntry, count)
	for i := range out {
		out[i] = symIndexEntry{
			id:  binary.LittleEndian.Uint64(raw[i*16:]),
			idx: binary.LittleEndian.Uint32(raw[i*16+8:]),
		}
	}
	return out, nil
}

func lookupSymIndex(idx []symIndexEntry, id uint64) (uint32, bool) {
	i := sort.Search(len(idx), func(i int) bool { return idx[i].id >= id })
	if i < len(idx) && idx[i].id == id {
		return idx[i].idx, true
	}
	return 0, false
}

// decodePtr resolves an on-disk pointer slot (Mach-O chained fixup or plain).
func decodePtr(info *binaryInfo, b []byte) uint64 {
	return decodePtrVal(info, binary.LittleEndian.Uint64(b))
}

func decodePtrVal(info *binaryInfo, v uint64) uint64 {
	if info.format == "macho" {
		if t := v & (1<<36 - 1); t != v && t >= info.imageBase {
			return t
		}
	}
	return v
}

func symbolAddr(path, name string) (uint64, error) {
	if mf, err := macho.Open(path); err == nil {
		defer mf.Close()
		for _, s := range mf.Symtab.Syms {
			if s.Name == "_"+name || s.Name == name {
				return s.Value, nil
			}
		}
		return 0, fmt.Errorf("symbol %s not found", name)
	}
	ef, err := elf.Open(path)
	if err != nil {
		return 0, err
	}
	defer ef.Close()
	syms, _ := ef.Symbols()
	for _, s := range syms {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return 0, fmt.Errorf("symbol %s not found", name)
}

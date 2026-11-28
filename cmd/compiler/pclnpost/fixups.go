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
	"fmt"
	"sort"
)

// Mach-O dyld chained fixups surgery.
//
// The on-disk pointer slots of the rewritten sections participate in dyld's
// chained-fixup page chains. If the chains are left untouched, dyld walks
// them at load time and rebases 8-byte slots inside our freshly written
// table, corrupting it — and terminating a chain early inside a zeroed
// section would also skip later fixups in the same page, corrupting
// unrelated data. unchainRanges removes every chain node that falls inside
// the given file-offset ranges: predecessors' next links (or the page_start
// table) are repointed to the first surviving successor.

const (
	lcDyldChainedFixups = 0x80000034
	lcSegment64         = 0x19

	chainedPtrStartNone  = 0xFFFF
	chainedPtrStartMulti = 0x8000

	// pointer_format values with a 12-bit next field at bit 51, stride 4.
	chainedPtr64       = 2
	chainedPtr64Offset = 6
)

type segRange struct {
	fileOff uint64
	fileSz  uint64
	vmaddr  uint64
}

// fixupInsert asks for a rebase fixup node at fileOff whose loaded value will
// be targetVM + slide. Slot bytes are returned as pending writes so the
// caller can apply them after overwriting the section contents.
type fixupInsert struct {
	fileOff  uint64
	targetVM uint64
}

type pendingWrite struct {
	fileOff uint64
	val     uint64
}

// unchainRanges edits chain metadata in raw in place: every chain node inside
// `ranges` (file-offset [start,end) pairs) is unlinked, and the requested
// `inserts` are spliced into the page chains as rebase nodes. Because insert
// slots usually lie inside a section the caller is about to overwrite, their
// encoded slot values are returned as pending writes to apply afterwards.
func unchainRanges(raw []byte, ranges [][2]uint64, inserts []fixupInsert) ([]pendingWrite, error) {
	inRange := func(off uint64) bool {
		for _, r := range ranges {
			if off >= r[0] && off < r[1] {
				return true
			}
		}
		return false
	}

	// Locate LC_DYLD_CHAINED_FIXUPS and the segment table.
	if len(raw) < 32 || binary.LittleEndian.Uint32(raw) != 0xFEEDFACF {
		return nil, fmt.Errorf("not a 64-bit little-endian Mach-O")
	}
	ncmds := binary.LittleEndian.Uint32(raw[16:])
	off := uint64(32)
	var fixOff, fixSize uint64
	var segs []segRange
	for i := uint32(0); i < ncmds; i++ {
		cmd := binary.LittleEndian.Uint32(raw[off:])
		size := binary.LittleEndian.Uint32(raw[off+4:])
		switch cmd {
		case lcDyldChainedFixups:
			fixOff = uint64(binary.LittleEndian.Uint32(raw[off+8:]))
			fixSize = uint64(binary.LittleEndian.Uint32(raw[off+12:]))
		case lcSegment64:
			segs = append(segs, segRange{
				vmaddr:  binary.LittleEndian.Uint64(raw[off+24:]),
				fileOff: binary.LittleEndian.Uint64(raw[off+40:]),
				fileSz:  binary.LittleEndian.Uint64(raw[off+48:]),
			})
		}
		off += uint64(size)
	}
	if fixOff == 0 {
		if len(inserts) > 0 {
			return nil, fmt.Errorf("no chained fixups to splice inserts into")
		}
		return nil, nil // no chained fixups (classic dyld info); nothing to do
	}
	_ = fixSize
	imageBase := ^uint64(0)
	for _, sg := range segs {
		if sg.vmaddr != 0 && sg.vmaddr < imageBase {
			imageBase = sg.vmaddr
		}
	}
	var pending []pendingWrite
	consumed := make(map[uint64]bool, len(inserts))

	hdr := raw[fixOff:]
	startsOff := fixOff + uint64(binary.LittleEndian.Uint32(hdr[4:]))
	segCount := binary.LittleEndian.Uint32(raw[startsOff:])
	if int(segCount) != len(segs) {
		// seg_count counts all segments incl. ones without fixups; trust it.
	}
	for si := uint32(0); si < segCount; si++ {
		segInfoOff := binary.LittleEndian.Uint32(raw[startsOff+4+uint64(si)*4:])
		if segInfoOff == 0 {
			continue
		}
		sOff := startsOff + uint64(segInfoOff)
		pageSize := uint64(binary.LittleEndian.Uint16(raw[sOff+4:]))
		ptrFormat := binary.LittleEndian.Uint16(raw[sOff+6:])
		pageCount := uint64(binary.LittleEndian.Uint16(raw[sOff+20:]))
		if ptrFormat != chainedPtr64 && ptrFormat != chainedPtr64Offset {
			// Only reject if this segment's pages intersect our ranges.
			segFile := segs[si].fileOff
			touches := false
			for _, r := range ranges {
				if r[0] < segFile+pageCount*pageSize && r[1] > segFile {
					touches = true
				}
			}
			if !touches {
				continue
			}
			return nil, fmt.Errorf("unsupported pointer_format %d", ptrFormat)
		}
		encode := func(targetVM, next uint64) uint64 {
			t := targetVM
			if ptrFormat == chainedPtr64Offset {
				t = targetVM - imageBase
			}
			return (t & (1<<36 - 1)) | (next << 51)
		}
		segFileOff := segs[si].fileOff
		for pi := uint64(0); pi < pageCount; pi++ {
			psOff := sOff + 22 + pi*2
			pageFile := segFileOff + pi*pageSize
			pageEnd := pageFile + pageSize
			// Inserts requested for this page.
			var ins []fixupInsert
			for _, in := range inserts {
				if in.fileOff >= pageFile && in.fileOff < pageEnd {
					ins = append(ins, in)
					consumed[in.fileOff] = true
				}
			}
			pStart := binary.LittleEndian.Uint16(raw[psOff:])
			if pStart == chainedPtrStartNone && len(ins) == 0 {
				continue
			}
			if pStart != chainedPtrStartNone && pStart&chainedPtrStartMulti != 0 {
				return nil, fmt.Errorf("multi-start pages not supported")
			}
			// Collect the existing chain.
			var nodes []uint64
			if pStart != chainedPtrStartNone {
				node := pageFile + uint64(pStart)
				for {
					nodes = append(nodes, node)
					val := binary.LittleEndian.Uint64(raw[node:])
					next := (val >> 51) & 0xFFF
					if next == 0 {
						break
					}
					node += next * 4
				}
			}
			// Rebuild: out-of-range survivors plus requested inserts.
			type finalNode struct {
				off      uint64
				insert   bool
				targetVM uint64
			}
			var final []finalNode
			removed := 0
			for _, n := range nodes {
				if inRange(n) {
					removed++
				} else {
					final = append(final, finalNode{off: n})
				}
			}
			for _, in := range ins {
				final = append(final, finalNode{off: in.fileOff, insert: true, targetVM: in.targetVM})
			}
			if removed == 0 && len(ins) == 0 {
				continue
			}
			sort.Slice(final, func(i, j int) bool { return final[i].off < final[j].off })
			if len(final) == 0 {
				binary.LittleEndian.PutUint16(raw[psOff:], chainedPtrStartNone)
				continue
			}
			binary.LittleEndian.PutUint16(raw[psOff:], uint16(final[0].off-pageFile))
			for i, n := range final {
				var next uint64
				if i+1 < len(final) {
					delta := final[i+1].off - n.off
					if delta%4 != 0 || delta/4 > 0xFFF {
						return nil, fmt.Errorf("chain gap %d not encodable", delta)
					}
					next = delta / 4
				}
				if n.insert {
					pending = append(pending, pendingWrite{fileOff: n.off, val: encode(n.targetVM, next)})
				} else {
					val := binary.LittleEndian.Uint64(raw[n.off:])
					val = (val &^ (uint64(0xFFF) << 51)) | (next << 51)
					binary.LittleEndian.PutUint64(raw[n.off:], val)
				}
			}
		}
	}
	// An unconsumed insert means its slot never joined a fixup chain; on a
	// PIE binary the runtime would then read an unslid value. Fail loudly so
	// the caller falls back to first-use construction instead.
	for _, in := range inserts {
		if !consumed[in.fileOff] {
			return nil, fmt.Errorf("fixup insert at %#x not within any chain page", in.fileOff)
		}
	}
	return pending, nil
}

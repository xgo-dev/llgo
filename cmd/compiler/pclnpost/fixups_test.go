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

// buildMachOForFixups fabricates the minimal Mach-O that unchainRanges
// parses: header, one LC_SEGMENT_64 covering one 16K page of payload, and an
// LC_DYLD_CHAINED_FIXUPS blob whose page chain holds `nodes` (file offsets,
// ascending, stride-4 aligned) encoded as DYLD_CHAINED_PTR_64 rebases.
func buildMachOForFixups(t *testing.T, nodes []uint64, targets []uint64) ([]byte, uint64) {
	t.Helper()
	const pageSize = 0x4000
	const segFileOff = uint64(0x8000)
	raw := make([]byte, segFileOff+pageSize)
	binary.LittleEndian.PutUint32(raw[0:], 0xFEEDFACF)
	binary.LittleEndian.PutUint32(raw[16:], 2) // ncmds
	// LC_SEGMENT_64 at 32
	off := 32
	binary.LittleEndian.PutUint32(raw[off:], 0x19)
	binary.LittleEndian.PutUint32(raw[off+4:], 72)
	binary.LittleEndian.PutUint64(raw[off+24:], 0x100000000) // vmaddr
	binary.LittleEndian.PutUint64(raw[off+40:], segFileOff)  // fileoff
	binary.LittleEndian.PutUint64(raw[off+48:], pageSize)    // filesize
	// LC_DYLD_CHAINED_FIXUPS
	off += 72
	fixOff := uint64(0x200)
	binary.LittleEndian.PutUint32(raw[off:], 0x80000034)
	binary.LittleEndian.PutUint32(raw[off+4:], 16)
	binary.LittleEndian.PutUint32(raw[off+8:], uint32(fixOff))
	binary.LittleEndian.PutUint32(raw[off+12:], 0x100)
	// dyld_chained_fixups_header
	h := fixOff
	binary.LittleEndian.PutUint32(raw[h+4:], 32) // starts_offset
	// starts_in_image: seg_count=1, seg_info_offset[0]=8
	s := h + 32
	binary.LittleEndian.PutUint32(raw[s:], 1)
	binary.LittleEndian.PutUint32(raw[s+4:], 8)
	// starts_in_segment
	g := s + 8
	binary.LittleEndian.PutUint16(raw[g+4:], 0x4000) // page_size
	binary.LittleEndian.PutUint16(raw[g+6:], 2)      // DYLD_CHAINED_PTR_64
	binary.LittleEndian.PutUint64(raw[g+8:], segFileOff)
	binary.LittleEndian.PutUint16(raw[g+20:], 1) // page_count
	if len(nodes) == 0 {
		binary.LittleEndian.PutUint16(raw[g+22:], chainedPtrStartNone)
	} else {
		binary.LittleEndian.PutUint16(raw[g+22:], uint16(nodes[0]-segFileOff))
	}
	for i, n := range nodes {
		next := uint64(0)
		if i+1 < len(nodes) {
			next = (nodes[i+1] - n) / 4
		}
		val := (next << 51) | (targets[i] & (1<<36 - 1))
		binary.LittleEndian.PutUint64(raw[n:], val)
	}
	return raw, segFileOff
}

func chainNodes(t *testing.T, raw []byte, segFileOff uint64) []uint64 {
	t.Helper()
	g := uint64(0x200) + 32 + 8
	start := binary.LittleEndian.Uint16(raw[g+22:])
	if start == chainedPtrStartNone {
		return nil
	}
	var out []uint64
	n := segFileOff + uint64(start)
	for {
		out = append(out, n)
		next := (binary.LittleEndian.Uint64(raw[n:]) >> 51) & 0xFFF
		if next == 0 {
			return out
		}
		n += next * 4
	}
}

func TestUnchainRangesRemovesAndSplices(t *testing.T) {
	nodes := []uint64{0x8000, 0x8010, 0x8020, 0x8030}
	targets := []uint64{0x100000000, 0x100000008, 0x100000010, 0x100000018}
	raw, seg := buildMachOForFixups(t, nodes, targets)
	// Remove the middle two, splice one insert into the removed range.
	pend, err := unchainRanges(raw, [][2]uint64{{0x8010, 0x8030}},
		[]fixupInsert{{fileOff: 0x8018, targetVM: 0x100000abc}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].fileOff != 0x8018 {
		t.Fatalf("pending %+v", pend)
	}
	binary.LittleEndian.PutUint64(raw[pend[0].fileOff:], pend[0].val)
	got := chainNodes(t, raw, seg)
	want := []uint64{0x8000, 0x8018, 0x8030}
	if len(got) != len(want) {
		t.Fatalf("chain %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chain %#v want %#v", got, want)
		}
	}
	if v := binary.LittleEndian.Uint64(raw[0x8018:]) & (1<<36 - 1); v != 0x100000abc&(1<<36-1) {
		t.Fatalf("insert target %#x", v)
	}
}

func TestUnchainRangesUnconsumedInsertFails(t *testing.T) {
	nodes := []uint64{0x8000}
	raw, _ := buildMachOForFixups(t, nodes, []uint64{0x100000000})
	if _, err := unchainRanges(raw, nil, []fixupInsert{{fileOff: 0x4, targetVM: 1}}); err == nil {
		t.Fatal("expected unconsumed-insert error")
	}
}

func TestUnchainRangesEmptyPageGainsInsert(t *testing.T) {
	raw, seg := buildMachOForFixups(t, nil, nil)
	pend, err := unchainRanges(raw, nil, []fixupInsert{{fileOff: 0x8040, targetVM: 0x100000042}})
	if err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(raw[pend[0].fileOff:], pend[0].val)
	got := chainNodes(t, raw, seg)
	if len(got) != 1 || got[0] != 0x8040 {
		t.Fatalf("chain %#v", got)
	}
	_ = seg
}

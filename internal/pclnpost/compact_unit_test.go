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
	"testing"
)

func TestCompactCarrierValidation(t *testing.T) {
	if _, _, err := compactCarrier(nil, &binaryInfo{entryVMSize: 1}, 2, 0); err == nil {
		t.Fatal("oversized entry was accepted")
	}
	if _, _, err := compactCarrier(nil, &binaryInfo{format: "unknown"}, 0, 0); err == nil {
		t.Fatal("unknown format was accepted")
	}
	if got := alignUp(7, 0); got != 7 {
		t.Fatalf("alignUp(7, 0) = %d", got)
	}
	if got := alignUp(^uint64(0)-1, 8); got != ^uint64(0) {
		t.Fatalf("overflowing alignUp = %#x", got)
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

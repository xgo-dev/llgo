/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package gotest

import (
	"encoding/binary"
	"testing"
)

// Bug: nested shift with untyped constant
//
// When a shift expression `(1 << byteVar)` is used as the right operand
// of another shift expression, SSA generates `1:untyped int << byteVar`
// with type `untyped int`, which violates SSA sanity checks.
//
// SSA output for nested shift expressions:
//   t2 = 1:untyped int << t1                                    untyped int  ← BUG
//   t3 = convert uint <- untyped int (t2)                              uint
//   t4 = 1:uint32 << t3                                              uint32
//
// The bug causes SSA sanity check to fail when compiling:
//   value << (1 << bytes[3])
// but NOT when an explicit conversion is used:
//   uint(1 << bytes[3])
// because the conversion forces type resolution before SSA construction.

// nestedShift triggers the bug: value << (1 << bytes[3])
func nestedShift(bytes []byte) uint32 {
	value := uint32(1)
	return value << (1 << bytes[3])
}

// nestedShiftWithBinary is the original gohex pattern
func nestedShiftWithBinary(bytes []byte) uint32 {
	return uint32(binary.BigEndian.Uint16(bytes[4:6])) << (1 << bytes[3])
}

// TestNestedShift tests the nested shift bug.
func TestNestedShift(t *testing.T) {
	bytes := []byte{0, 0, 0, 4, 0, 1} // bytes[3] = 4

	// 1 << (1 << 4) = 1 << 16 = 65536
	if r := nestedShift(bytes); r != 65536 {
		t.Errorf("nestedShift: got %d, want 65536", r)
	}
}

// TestNestedShiftWithBinary tests the original gohex pattern.
func TestNestedShiftWithBinary(t *testing.T) {
	bytes := []byte{0, 0, 0, 4, 0, 1} // bytes[3] = 4, bytes[4:6] = 0x0001

	// binary.BigEndian.Uint16(bytes[4:6]) = 1
	// (1 << bytes[3]) = (1 << 4) = 16
	// 1 << 16 = 65536
	if r := nestedShiftWithBinary(bytes); r != 65536 {
		t.Errorf("nestedShiftWithBinary: got %d, want 65536", r)
	}
}

// TestNestedShiftEdgeCases tests edge cases.
func TestNestedShiftEdgeCases(t *testing.T) {
	bytes := []byte{0, 0, 0, 0, 0, 1} // bytes[3] = 0

	// 1 << (1 << 0) = 1 << 1 = 2
	if r := nestedShift(bytes); r != 2 {
		t.Errorf("nestedShift (shift by 1): got %d, want 2", r)
	}

	bytes[3] = 3 // 1 << (1 << 3) = 1 << 8 = 256
	if r := nestedShift(bytes); r != 256 {
		t.Errorf("nestedShift (shift by 8): got %d, want 256", r)
	}
}

// issue12133F1 covers a non-constant shift whose left operand is an
// untyped constant. go/types can leave parent expressions untyped, which
// must be defaulted before building SSA.
func issue12133F1(v1 uint) uint {
	return v1 >> ((1 >> v1) + (1 >> v1))
}

func TestGorootIssue12133UnsignedShiftCount(t *testing.T) {
	if got := issue12133F1(48); got != 48 {
		t.Fatalf("issue12133F1(48) = %d, want 48", got)
	}
}

func issue15175Bool(v bool) bool {
	return v
}

func issue15175F1(a1 uint, a2 int8, a3 int8, a4 int8, a5 uint8, a6 int, a7 bool) uint8 {
	a5--
	a4 += (a2 << a1 << 2) | (a4 ^ a4<<(a1&a1)) - a3
	a6 -= a6 >> (2 + uint32(a2)>>3)
	a1 += a1
	a3 *= a4 << (a1 | a1) << (uint16(3) >> 2 & (1 - 0) & (uint16(1) << a5 << 3))
	_ = issue15175Bool(a7) || (a2 == a4) || (a5 == 0)
	return a5 >> a1
}

func issue15175F2(a1 uint8) uint8 {
	a1--
	a1--
	a1 -= a1 + (a1 << 1) - (a1*a1*a1)<<(2-0+(3|3)-1)
	v1 := 0 * ((2 * 1) ^ 1) & ((uint(0) >> a1) + (2+0)*(uint(2)+0))
	_ = v1
	return a1 >> (((2 ^ 2) >> (v1 | 2)) + 0)
}

func issue15175F3(a1 bool, a2 uint, a3 int64) uint8 {
	a3--
	v1 := 1 & (2 & 1 * (1 ^ 2) & (uint8(3*1) >> 0))
	_ = v1
	v1 += v1 - (v1 >> a2) + (v1 << (a2 ^ a2) & v1)
	v1 *= v1
	a3--
	v1 += v1 & v1
	v1--
	v1 = ((v1 << 0) | v1>>0) + v1
	return v1 >> 0
}

func TestGorootIssue15175UnsignedNarrowShiftResults(t *testing.T) {
	a6 := uint8(253)
	if got := a6 >> 0; got != 253 {
		t.Fatalf("uint8(253) >> 0 = %d, want 253", got)
	}
	if got := issue15175F1(0, 2, 1, 0, 0, 1, true); got != 255 {
		t.Fatalf("issue15175F1(...) = %d, want 255", got)
	}
	if got := issue15175F2(1); got != 242 {
		t.Fatalf("issue15175F2(1) = %d, want 242", got)
	}
	if got := issue15175F3(false, 0, 0); got != 254 {
		t.Fatalf("issue15175F3(false, 0, 0) = %d, want 254", got)
	}
}

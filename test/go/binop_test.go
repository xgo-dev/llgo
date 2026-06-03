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
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestIntegerOverflow tests integer overflow behavior
func TestIntegerOverflow(t *testing.T) {
	// Test int8 overflow
	var i8max int8 = 127
	result := i8max + 1
	expected := int8(-128)
	if result != expected {
		t.Errorf("int8 overflow: got %d, want %d", result, expected)
	}

	// Test uint8 overflow
	var u8max uint8 = 255
	resultU8 := u8max + 1
	expectedU8 := uint8(0)
	if resultU8 != expectedU8 {
		t.Errorf("uint8 overflow: got %d, want %d", resultU8, expectedU8)
	}

	// Test int8 underflow
	var i8min int8 = -128
	resultMin := i8min - 1
	expectedMin := int8(127)
	if resultMin != expectedMin {
		t.Errorf("int8 underflow: got %d, want %d", resultMin, expectedMin)
	}
}

// TestUint32ToInt64Conversion tests uint32 to int64 conversion
func TestUint32ToInt64Conversion(t *testing.T) {
	var u32 uint32 = 4000000000
	result := int64(u32)
	expected := int64(4000000000)
	if result != expected {
		t.Errorf("uint32 to int64 conversion: got %d, want %d", result, expected)
	}

	// Test with max uint32
	var maxU32 uint32 = 0xFFFFFFFF
	resultMax := int64(maxU32)
	expectedMax := int64(4294967295)
	if resultMax != expectedMax {
		t.Errorf("max uint32 to int64 conversion: got %d, want %d", resultMax, expectedMax)
	}
}

// TestMixedSignedUnsignedAddition tests mixed signed/unsigned integer addition
func TestMixedSignedUnsignedAddition(t *testing.T) {
	var i32 int32 = 70000
	var u32 uint32 = 4000000000
	result := int64(i32) + int64(u32)
	expected := int64(4000070000)
	if result != expected {
		t.Errorf("int32 + uint32 as int64: got %d, want %d", result, expected)
	}
}

// TestUntypedConstantsWithTypedVariables tests untyped constant arithmetic
func TestUntypedConstantsWithTypedVariables(t *testing.T) {
	const untypedInt = 42
	var i32 int32 = 70000
	result := untypedInt + i32
	expected := int32(70042)
	if result != expected {
		t.Errorf("untypedInt + i32: got %d, want %d", result, expected)
	}

	const untypedFloat = 3.14
	var f32 float32 = 3.14159
	resultFloat := untypedFloat + f32
	expectedFloat := float32(6.28159)
	// Use approximate comparison for floats
	if resultFloat < expectedFloat-0.001 || resultFloat > expectedFloat+0.001 {
		t.Errorf("untypedFloat + f32: got %f, want %f", resultFloat, expectedFloat)
	}

	const untypedComplex = 1 + 2i
	var c64 complex64 = 1 + 2i
	resultComplex := untypedComplex + c64
	expectedComplex := complex64(2 + 4i)
	if resultComplex != expectedComplex {
		t.Errorf("untypedComplex + c64: got %v, want %v", resultComplex, expectedComplex)
	}
}

// TestSignedIntegerConversions tests conversions between signed integers
func TestSignedIntegerConversions(t *testing.T) {
	var i8 int8 = 64
	var i16 int16 = 1000

	result16 := int16(i8) + i16
	expected16 := int16(1064)
	if result16 != expected16 {
		t.Errorf("i8 + i16: got %d, want %d", result16, expected16)
	}

	var i32 int32 = 70000
	result32 := int32(i16) + i32
	expected32 := int32(71000)
	if result32 != expected32 {
		t.Errorf("i16 + i32: got %d, want %d", result32, expected32)
	}

	var i64 int64 = 1<<32 - 1
	result64 := int64(i32) + i64
	expected64 := int64(4295037295)
	if result64 != expected64 {
		t.Errorf("i32 + i64: got %d, want %d", result64, expected64)
	}
}

// TestUnsignedIntegerConversions tests conversions between unsigned integers
func TestUnsignedIntegerConversions(t *testing.T) {
	var u8 uint8 = 200
	var u16 uint16 = 50000

	result16 := uint16(u8) + u16
	expected16 := uint16(50200)
	if result16 != expected16 {
		t.Errorf("u8 + u16: got %d, want %d", result16, expected16)
	}

	var u32 uint32 = 4000000000
	result32 := uint32(u16) + u32
	expected32 := uint32(4000050000)
	if result32 != expected32 {
		t.Errorf("u16 + u32: got %d, want %d", result32, expected32)
	}

	var u64 uint64 = 1<<64 - 1
	result64 := uint64(u32) + u64
	expected64 := uint64(3999999999) // Note: wraps around due to u64 being max
	if result64 != expected64 {
		t.Errorf("u32 + u64: got %d, want %d", result64, expected64)
	}
}

// TestMixedFloatOperations tests mixed float operations
func TestMixedFloatOperations(t *testing.T) {
	var f32 float32 = 3.14159
	var f64 float64 = 3.14159265359

	result := float64(f32) + f64
	expected := 6.283183
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("f32 + f64: got %f, want %f", result, expected)
	}

	var i32 int32 = 70000
	resultI := float64(i32) + f64
	expectedI := 70003.141593
	if resultI < expectedI-0.001 || resultI > expectedI+0.001 {
		t.Errorf("i32 + f64: got %f, want %f", resultI, expectedI)
	}

	var u32 uint32 = 4000000000
	resultU := float32(u32) + f32
	expectedU := float32(4000000000.0)
	if resultU < expectedU-1.0 || resultU > expectedU+1.0 {
		t.Errorf("u32 + f32: got %f, want %f", resultU, expectedU)
	}
}

// TestComplexNumberOperations tests complex number operations
func TestComplexNumberOperations(t *testing.T) {
	var c64 complex64 = 1 + 2i
	var c128 complex128 = 1 + 2i

	result := complex128(c64) + c128
	expected := complex128(2 + 4i)
	if result != expected {
		t.Errorf("c64 + c128: got %v, want %v", result, expected)
	}

	var f64 float64 = 3.14159265359
	resultF := complex(f64, 0) + c128
	expectedF := complex128(4.14159265359 + 2i)
	// Approximate comparison for complex
	if real(resultF) < real(expectedF)-0.001 || real(resultF) > real(expectedF)+0.001 {
		t.Errorf("f64 + c128: got %v, want %v", resultF, expectedF)
	}

	var i32 int32 = 70000
	resultI := complex(float32(i32), 0) + c64
	expectedI := complex64(70001 + 2i)
	if resultI != expectedI {
		t.Errorf("i32 + c64: got %v, want %v", resultI, expectedI)
	}
}

// TestComplexDivisionAndMultiplication tests complex division and multiplication
func TestComplexDivisionAndMultiplication(t *testing.T) {
	var c64 complex64 = 1 + 2i
	var c128 complex128 = 1 + 2i

	// Test division
	resultDiv := complex128(c64) / c128
	expectedDiv := complex128(1 + 0i)
	if resultDiv != expectedDiv {
		t.Errorf("complex division: got %v, want %v", resultDiv, expectedDiv)
	}

	// Test multiplication
	resultMul := c64 * complex64(c128)
	expectedMul := complex64(-3 + 4i)
	if resultMul != expectedMul {
		t.Errorf("complex multiplication: got %v, want %v", resultMul, expectedMul)
	}
}

func TestComplexDivisionSpecialValues(t *testing.T) {
	cases := []struct {
		name string
		num  complex128
		den  complex128
	}{
		{
			name: "finite over infinite denominator",
			num:  complex(1, 1),
			den:  complex(math.Inf(1), math.Inf(1)),
		},
		{
			name: "finite over max finite denominator",
			num:  complex(1, 1),
			den:  complex(math.MaxFloat64, math.MaxFloat64),
		},
	}
	for _, tc := range cases {
		got := tc.num / tc.den
		if real(got) != 0 || imag(got) != 0 {
			t.Fatalf("%s: got %v, want 0+0i", tc.name, got)
		}
	}
}

// TestInt16ToUint32Conversion tests int16 to uint32 conversion
func TestInt16ToUint32Conversion(t *testing.T) {
	var i16 int16 = -1
	result := uint32(i16)
	expected := uint32(0xFFFFFFFF)
	if result != expected {
		t.Errorf("int16(-1) to uint32: got %d, want %d", result, expected)
	}
}

// TestUint16ToInt32Conversion tests uint16 to int32 conversion
func TestUint16ToInt32Conversion(t *testing.T) {
	var u16 uint16 = 0xFFFF
	result := int32(u16)
	expected := int32(65535)
	if result != expected {
		t.Errorf("uint16(0xFFFF) to int32: got %d, want %d", result, expected)
	}
}

// TestFmtSprintfIntegerOverflow tests integer overflow with fmt.Sprintf
// This catches issues where conversion to any in fmt.Printf may hide bugs
func TestFmtSprintfIntegerOverflow(t *testing.T) {
	var i8max int8 = 127
	result := i8max + 1

	// Test direct value
	if result != -128 {
		t.Errorf("int8 overflow: got %d, want -128", result)
	}

	// Test fmt.Sprintf output
	str := fmt.Sprintf("%d", result)
	expectedStr := "-128"
	if str != expectedStr {
		t.Errorf("int8 overflow fmt.Sprintf: got %s, want %s", str, expectedStr)
	}

	// Test type preservation through any conversion
	var anyVal any = result
	if v, ok := anyVal.(int8); !ok || v != -128 {
		t.Errorf("int8 overflow any conversion: got %v (type %T), want -128 (type int8)", anyVal, anyVal)
	}
}

// TestFmtSprintfUint32ToInt64 tests uint32 to int64 conversion with fmt.Sprintf
func TestFmtSprintfUint32ToInt64(t *testing.T) {
	var u32 uint32 = 4000000000
	result := int64(u32)

	// Test direct value
	if result != 4000000000 {
		t.Errorf("uint32 to int64: got %d, want 4000000000", result)
	}

	// Test fmt.Sprintf output
	str := fmt.Sprintf("%d", result)
	expectedStr := "4000000000"
	if str != expectedStr {
		t.Errorf("uint32 to int64 fmt.Sprintf: got %s, want %s", str, expectedStr)
	}

	// Test with type in format
	strTyped := fmt.Sprintf("%d (type %T)", result, result)
	expectedTyped := "4000000000 (type int64)"
	if strTyped != expectedTyped {
		t.Errorf("uint32 to int64 with type: got %s, want %s", strTyped, expectedTyped)
	}
}

// TestFmtSprintfMixedSignedUnsigned tests mixed operations with fmt.Sprintf
func TestFmtSprintfMixedSignedUnsigned(t *testing.T) {
	var i32 int32 = 70000
	var u32 uint32 = 4000000000
	result := int64(i32) + int64(u32)

	// Test direct value
	if result != 4000070000 {
		t.Errorf("int32 + uint32 as int64: got %d, want 4000070000", result)
	}

	// Test fmt.Sprintf output
	str := fmt.Sprintf("%d", result)
	expectedStr := "4000070000"
	if str != expectedStr {
		t.Errorf("mixed addition fmt.Sprintf: got %s, want %s", str, expectedStr)
	}
}

// TestFmtSprintfUntypedConstants tests untyped constant operations with fmt.Sprintf
func TestFmtSprintfUntypedConstants(t *testing.T) {
	const untypedInt = 42
	var i32 int32 = 70000
	result := untypedInt + i32

	// Test value and type
	str := fmt.Sprintf("%d (type %T)", result, result)
	expectedStr := "70042 (type int32)"
	if str != expectedStr {
		t.Errorf("untypedInt + i32: got %s, want %s", str, expectedStr)
	}

	// Test float
	const untypedFloat = 3.14
	var f32 float32 = 3.14159
	resultFloat := untypedFloat + f32
	strFloat := fmt.Sprintf("%f (type %T)", resultFloat, resultFloat)
	// Note: using Contains check due to float precision
	if !contains(strFloat, "6.28") || !contains(strFloat, "float32") {
		t.Errorf("untypedFloat + f32: got %s, want ~6.28... (type float32)", strFloat)
	}

	// Test complex
	const untypedComplex = 1 + 2i
	var c64 complex64 = 1 + 2i
	resultComplex := untypedComplex + c64
	strComplex := fmt.Sprintf("%v (type %T)", resultComplex, resultComplex)
	if !contains(strComplex, "(2+4i)") || !contains(strComplex, "complex64") {
		t.Errorf("untypedComplex + c64: got %s, want (2+4i) (type complex64)", strComplex)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestFmtSprintfComplexOperations tests complex number operations with fmt.Sprintf
func TestFmtSprintfComplexOperations(t *testing.T) {
	var c64 complex64 = 1 + 2i
	var c128 complex128 = 1 + 2i

	// Test addition
	resultAdd := complex128(c64) + c128
	strAdd := fmt.Sprintf("%v", resultAdd)
	if strAdd != "(2+4i)" {
		t.Errorf("c64 + c128: got %s, want (2+4i)", strAdd)
	}

	// Test division
	resultDiv := complex128(c64) / c128
	strDiv := fmt.Sprintf("%v", resultDiv)
	if strDiv != "(1+0i)" {
		t.Errorf("complex division: got %s, want (1+0i)", strDiv)
	}

	// Test multiplication
	resultMul := c64 * complex64(c128)
	strMul := fmt.Sprintf("%v", resultMul)
	if strMul != "(-3+4i)" {
		t.Errorf("complex multiplication: got %s, want (-3+4i)", strMul)
	}
}

// =============================================================================
// Comprehensive BinOp Tests - All Type Combinations
// =============================================================================
// These tests cover all combinations of:
// - typed OP typed
// - typed OP untyped
// - untyped OP typed
// - untyped OP untyped
// For all integer types (signed, unsigned) and float types

// TestBinOpTypedIntegerAddition tests typed integer + typed integer
func TestBinOpTypedIntegerAddition(t *testing.T) {
	// Signed integers
	var i8a, i8b int8 = 10, 20
	if i8a+i8b != 30 {
		t.Errorf("int8 + int8: got %d, want 30", i8a+i8b)
	}

	var i16a, i16b int16 = 100, 200
	if i16a+i16b != 300 {
		t.Errorf("int16 + int16: got %d, want 300", i16a+i16b)
	}

	var i32a, i32b int32 = 1000, 2000
	if i32a+i32b != 3000 {
		t.Errorf("int32 + int32: got %d, want 3000", i32a+i32b)
	}

	var i64a, i64b int64 = 10000, 20000
	if i64a+i64b != 30000 {
		t.Errorf("int64 + int64: got %d, want 30000", i64a+i64b)
	}

	// Unsigned integers
	var u8a, u8b uint8 = 10, 20
	if u8a+u8b != 30 {
		t.Errorf("uint8 + uint8: got %d, want 30", u8a+u8b)
	}

	var u16a, u16b uint16 = 100, 200
	if u16a+u16b != 300 {
		t.Errorf("uint16 + uint16: got %d, want 300", u16a+u16b)
	}

	var u32a, u32b uint32 = 1000, 2000
	if u32a+u32b != 3000 {
		t.Errorf("uint32 + uint32: got %d, want 3000", u32a+u32b)
	}

	var u64a, u64b uint64 = 10000, 20000
	if u64a+u64b != 30000 {
		t.Errorf("uint64 + uint64: got %d, want 30000", u64a+u64b)
	}
}

// TestBinOpTypedFloatAddition tests typed float + typed float
func TestBinOpTypedFloatAddition(t *testing.T) {
	var f32a, f32b float32 = 1.5, 2.5
	if f32a+f32b != 4.0 {
		t.Errorf("float32 + float32: got %f, want 4.0", f32a+f32b)
	}

	var f64a, f64b float64 = 1.5, 2.5
	if f64a+f64b != 4.0 {
		t.Errorf("float64 + float64: got %f, want 4.0", f64a+f64b)
	}
}

// TestBinOpTypedIntegerUntypedAddition tests typed integer + untyped constant
func TestBinOpTypedIntegerUntypedAddition(t *testing.T) {
	const untyped = 10

	var i8 int8 = 20
	if i8+untyped != 30 {
		t.Errorf("int8 + untyped: got %d, want 30", i8+untyped)
	}

	var i16 int16 = 20
	if i16+untyped != 30 {
		t.Errorf("int16 + untyped: got %d, want 30", i16+untyped)
	}

	var i32 int32 = 20
	if i32+untyped != 30 {
		t.Errorf("int32 + untyped: got %d, want 30", i32+untyped)
	}

	var i64 int64 = 20
	if i64+untyped != 30 {
		t.Errorf("int64 + untyped: got %d, want 30", i64+untyped)
	}

	var u8 uint8 = 20
	if u8+untyped != 30 {
		t.Errorf("uint8 + untyped: got %d, want 30", u8+untyped)
	}

	var u16 uint16 = 20
	if u16+untyped != 30 {
		t.Errorf("uint16 + untyped: got %d, want 30", u16+untyped)
	}

	var u32 uint32 = 20
	if u32+untyped != 30 {
		t.Errorf("uint32 + untyped: got %d, want 30", u32+untyped)
	}

	var u64 uint64 = 20
	if u64+untyped != 30 {
		t.Errorf("uint64 + untyped: got %d, want 30", u64+untyped)
	}
}

// TestBinOpUntypedTypedIntegerAddition tests untyped constant + typed integer
func TestBinOpUntypedTypedIntegerAddition(t *testing.T) {
	const untyped = 10

	var i8 int8 = 20
	if untyped+i8 != 30 {
		t.Errorf("untyped + int8: got %d, want 30", untyped+i8)
	}

	var i16 int16 = 20
	if untyped+i16 != 30 {
		t.Errorf("untyped + int16: got %d, want 30", untyped+i16)
	}

	var i32 int32 = 20
	if untyped+i32 != 30 {
		t.Errorf("untyped + int32: got %d, want 30", untyped+i32)
	}

	var i64 int64 = 20
	if untyped+i64 != 30 {
		t.Errorf("untyped + int64: got %d, want 30", untyped+i64)
	}

	var u8 uint8 = 20
	if untyped+u8 != 30 {
		t.Errorf("untyped + uint8: got %d, want 30", untyped+u8)
	}

	var u16 uint16 = 20
	if untyped+u16 != 30 {
		t.Errorf("untyped + uint16: got %d, want 30", untyped+u16)
	}

	var u32 uint32 = 20
	if untyped+u32 != 30 {
		t.Errorf("untyped + uint32: got %d, want 30", untyped+u32)
	}

	var u64 uint64 = 20
	if untyped+u64 != 30 {
		t.Errorf("untyped + uint64: got %d, want 30", untyped+u64)
	}
}

// TestBinOpUntypedUntypedAddition tests untyped constant + untyped constant
func TestBinOpUntypedUntypedAddition(t *testing.T) {
	const a = 10
	const b = 20
	const result = a + b
	if result != 30 {
		t.Errorf("untyped + untyped: got %d, want 30", result)
	}
}

// TestBinOpTypedFloatUntypedAddition tests typed float + untyped constant
func TestBinOpTypedFloatUntypedAddition(t *testing.T) {
	const untyped = 1.5

	var f32 float32 = 2.5
	if f32+untyped != 4.0 {
		t.Errorf("float32 + untyped: got %f, want 4.0", f32+untyped)
	}

	var f64 float64 = 2.5
	if f64+untyped != 4.0 {
		t.Errorf("float64 + untyped: got %f, want 4.0", f64+untyped)
	}
}

// TestBinOpUntypedTypedFloatAddition tests untyped constant + typed float
func TestBinOpUntypedTypedFloatAddition(t *testing.T) {
	const untyped = 1.5

	var f32 float32 = 2.5
	if untyped+f32 != 4.0 {
		t.Errorf("untyped + float32: got %f, want 4.0", untyped+f32)
	}

	var f64 float64 = 2.5
	if untyped+f64 != 4.0 {
		t.Errorf("untyped + float64: got %f, want 4.0", untyped+f64)
	}
}

// TestBinOpIntegerMultiplication tests multiplication for all integer types
func TestBinOpIntegerMultiplication(t *testing.T) {
	// Typed * Typed
	var i8a, i8b int8 = 5, 6
	if i8a*i8b != 30 {
		t.Errorf("int8 * int8: got %d, want 30", i8a*i8b)
	}

	var u32a, u32b uint32 = 5, 6
	if u32a*u32b != 30 {
		t.Errorf("uint32 * uint32: got %d, want 30", u32a*u32b)
	}

	// Typed * Untyped
	const untyped = 6
	var i16 int16 = 5
	if i16*untyped != 30 {
		t.Errorf("int16 * untyped: got %d, want 30", i16*untyped)
	}

	// Untyped * Typed
	var u64 uint64 = 5
	if untyped*u64 != 30 {
		t.Errorf("untyped * uint64: got %d, want 30", untyped*u64)
	}
}

// TestBinOpIntegerSubtraction tests subtraction for all integer types
func TestBinOpIntegerSubtraction(t *testing.T) {
	// Typed - Typed
	var i32a, i32b int32 = 50, 20
	if i32a-i32b != 30 {
		t.Errorf("int32 - int32: got %d, want 30", i32a-i32b)
	}

	var u16a, u16b uint16 = 50, 20
	if u16a-u16b != 30 {
		t.Errorf("uint16 - uint16: got %d, want 30", u16a-u16b)
	}

	// Typed - Untyped
	const untyped = 20
	var i64 int64 = 50
	if i64-untyped != 30 {
		t.Errorf("int64 - untyped: got %d, want 30", i64-untyped)
	}

	// Untyped - Typed
	const untypedA = 50
	var u8 uint8 = 20
	if untypedA-u8 != 30 {
		t.Errorf("untyped - uint8: got %d, want 30", untypedA-u8)
	}
}

// TestBinOpIntegerDivision tests division for all integer types
func TestBinOpIntegerDivision(t *testing.T) {
	// Typed / Typed
	var i8a, i8b int8 = 60, 2
	if i8a/i8b != 30 {
		t.Errorf("int8 / int8: got %d, want 30", i8a/i8b)
	}

	var u32a, u32b uint32 = 60, 2
	if u32a/u32b != 30 {
		t.Errorf("uint32 / uint32: got %d, want 30", u32a/u32b)
	}

	// Typed / Untyped
	const untyped = 2
	var i16 int16 = 60
	if i16/untyped != 30 {
		t.Errorf("int16 / untyped: got %d, want 30", i16/untyped)
	}

	// Untyped / Typed
	const untypedA = 60
	var u64 uint64 = 2
	if untypedA/u64 != 30 {
		t.Errorf("untyped / uint64: got %d, want 30", untypedA/u64)
	}
}

func TestBinOpSignedMinIntDivisionOverflow(t *testing.T) {
	minInt := -int(^uint(0)>>1) - 1
	negOne := -1
	if got := minInt / negOne; got != minInt {
		t.Fatalf("minInt / -1 = %d, want %d", got, minInt)
	}
	if got := minInt % negOne; got != 0 {
		t.Fatalf("minInt %% -1 = %d, want 0", got)
	}

	var minInt8 int8 = -128
	var negOne8 int8 = -1
	if got := minInt8 / negOne8; got != minInt8 {
		t.Fatalf("int8 min / -1 = %d, want %d", got, minInt8)
	}
	if got := minInt8 % negOne8; got != 0 {
		t.Fatalf("int8 min %% -1 = %d, want 0", got)
	}

	var minInt16 int16 = -32768
	var negOne16 int16 = -1
	if got := minInt16 / negOne16; got != minInt16 {
		t.Fatalf("int16 min / -1 = %d, want %d", got, minInt16)
	}
	if got := minInt16 % negOne16; got != 0 {
		t.Fatalf("int16 min %% -1 = %d, want 0", got)
	}

	var minInt32 int32 = -2147483648
	var negOne32 int32 = -1
	if got := minInt32 / negOne32; got != minInt32 {
		t.Fatalf("int32 min / -1 = %d, want %d", got, minInt32)
	}
	if got := minInt32 % negOne32; got != 0 {
		t.Fatalf("int32 min %% -1 = %d, want 0", got)
	}

	var minInt64 int64 = -9223372036854775808
	var negOne64 int64 = -1
	if got := minInt64 / negOne64; got != minInt64 {
		t.Fatalf("int64 min / -1 = %d, want %d", got, minInt64)
	}
	if got := minInt64 % negOne64; got != 0 {
		t.Fatalf("int64 min %% -1 = %d, want 0", got)
	}
}

func TestBinOpIntegerDivideByZeroPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("division by zero did not panic")
		}
	}()
	zero := 0
	_ = 1 / zero
}

func TestBinOpIntegerRemainderByZeroPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("remainder by zero did not panic")
		}
	}()
	zero := 0
	_ = 1 % zero
}

// TestBinOpIntegerModulo tests modulo for all integer types
func TestBinOpIntegerModulo(t *testing.T) {
	// Typed % Typed
	var i32a, i32b int32 = 100, 7
	if i32a%i32b != 2 {
		t.Errorf("int32 %% int32: got %d, want 2", i32a%i32b)
	}

	var u8a, u8b uint8 = 100, 7
	if u8a%u8b != 2 {
		t.Errorf("uint8 %% uint8: got %d, want 2", u8a%u8b)
	}

	// Typed % Untyped
	const untyped = 7
	var i64 int64 = 100
	if i64%untyped != 2 {
		t.Errorf("int64 %% untyped: got %d, want 2", i64%untyped)
	}

	// Untyped % Typed
	const untypedA = 100
	var u16 uint16 = 7
	if untypedA%u16 != 2 {
		t.Errorf("untyped %% uint16: got %d, want 2", untypedA%u16)
	}
}

// TestBinOpIntegerBitAnd tests bitwise AND for all integer types
func TestBinOpIntegerBitAnd(t *testing.T) {
	// Typed & Typed
	var i8a, i8b int8 = 0x3F, 0x0F
	if i8a&i8b != 0x0F {
		t.Errorf("int8 & int8: got %d, want %d", i8a&i8b, 0x0F)
	}

	var u32a, u32b uint32 = 0xFF, 0x0F
	if u32a&u32b != 0x0F {
		t.Errorf("uint32 & uint32: got %d, want %d", u32a&u32b, 0x0F)
	}

	// Typed & Untyped
	const untyped = 0x0F
	var i16 int16 = 0xFF
	if i16&untyped != 0x0F {
		t.Errorf("int16 & untyped: got %d, want %d", i16&untyped, 0x0F)
	}

	// Untyped & Typed
	const untypedA = 0xFF
	var u64 uint64 = 0x0F
	if untypedA&u64 != 0x0F {
		t.Errorf("untyped & uint64: got %d, want %d", untypedA&u64, 0x0F)
	}
}

// TestBinOpIntegerBitOr tests bitwise OR for all integer types
func TestBinOpIntegerBitOr(t *testing.T) {
	// Typed | Typed
	var i32a, i32b int32 = 0x30, 0x0F
	if i32a|i32b != 0x3F {
		t.Errorf("int32 | int32: got %d, want %d", i32a|i32b, 0x3F)
	}

	var u16a, u16b uint16 = 0x30, 0x0F
	if u16a|u16b != 0x3F {
		t.Errorf("uint16 | uint16: got %d, want %d", u16a|u16b, 0x3F)
	}

	// Typed | Untyped
	const untyped = 0x0F
	var i64 int64 = 0x30
	if i64|untyped != 0x3F {
		t.Errorf("int64 | untyped: got %d, want %d", i64|untyped, 0x3F)
	}

	// Untyped | Typed
	const untypedA = 0x30
	var u8 uint8 = 0x0F
	if untypedA|u8 != 0x3F {
		t.Errorf("untyped | uint8: got %d, want %d", untypedA|u8, 0x3F)
	}
}

// TestBinOpIntegerBitXor tests bitwise XOR for all integer types
func TestBinOpIntegerBitXor(t *testing.T) {
	// Typed ^ Typed
	var i8a, i8b int8 = 0x3F, 0x0F
	if i8a^i8b != 0x30 {
		t.Errorf("int8 ^ int8: got %d, want %d", i8a^i8b, 0x30)
	}

	var u64a, u64b uint64 = 0x3F, 0x0F
	if u64a^u64b != 0x30 {
		t.Errorf("uint64 ^ uint64: got %d, want %d", u64a^u64b, 0x30)
	}

	// Typed ^ Untyped
	const untyped = 0x0F
	var i32 int32 = 0x3F
	if i32^untyped != 0x30 {
		t.Errorf("int32 ^ untyped: got %d, want %d", i32^untyped, 0x30)
	}

	// Untyped ^ Typed
	const untypedA = 0x3F
	var u32 uint32 = 0x0F
	if untypedA^u32 != 0x30 {
		t.Errorf("untyped ^ uint32: got %d, want %d", untypedA^u32, 0x30)
	}
}

// TestBinOpIntegerBitAndNot tests bitwise AND NOT for all integer types
func TestBinOpIntegerBitAndNot(t *testing.T) {
	// Typed &^ Typed
	var i16a, i16b int16 = 0x3F, 0x0F
	if i16a&^i16b != 0x30 {
		t.Errorf("int16 &^ int16: got %d, want %d", i16a&^i16b, 0x30)
	}

	var u8a, u8b uint8 = 0x3F, 0x0F
	if u8a&^u8b != 0x30 {
		t.Errorf("uint8 &^ uint8: got %d, want %d", u8a&^u8b, 0x30)
	}

	// Typed &^ Untyped
	const untyped = 0x0F
	var i64 int64 = 0x3F
	if i64&^untyped != 0x30 {
		t.Errorf("int64 &^ untyped: got %d, want %d", i64&^untyped, 0x30)
	}

	// Untyped &^ Typed
	const untypedA = 0x3F
	var u16 uint16 = 0x0F
	if untypedA&^u16 != 0x30 {
		t.Errorf("untyped &^ uint16: got %d, want %d", untypedA&^u16, 0x30)
	}
}

// TestBinOpIntegerShiftLeft tests left shift for all integer types
func TestBinOpIntegerShiftLeft(t *testing.T) {
	// Typed << Typed
	var i8 int8 = 1
	var shift1 uint = 3
	if i8<<shift1 != 8 {
		t.Errorf("int8 << uint: got %d, want 8", i8<<shift1)
	}

	var u32 uint32 = 1
	var shift2 uint = 5
	if u32<<shift2 != 32 {
		t.Errorf("uint32 << uint: got %d, want 32", u32<<shift2)
	}

	// Typed << Untyped
	const untypedShift = 3
	var i16 int16 = 1
	if i16<<untypedShift != 8 {
		t.Errorf("int16 << untyped: got %d, want 8", i16<<untypedShift)
	}

	var u64 uint64 = 1
	if u64<<untypedShift != 8 {
		t.Errorf("uint64 << untyped: got %d, want 8", u64<<untypedShift)
	}
}

// TestBinOpIntegerShiftRight tests right shift for all integer types
func TestBinOpIntegerShiftRight(t *testing.T) {
	// Typed >> Typed
	var i32 int32 = 32
	var shift1 uint = 2
	if i32>>shift1 != 8 {
		t.Errorf("int32 >> uint: got %d, want 8", i32>>shift1)
	}

	var u16 uint16 = 32
	var shift2 uint = 2
	if u16>>shift2 != 8 {
		t.Errorf("uint16 >> uint: got %d, want 8", u16>>shift2)
	}

	// Typed >> Untyped
	const untypedShift = 2
	var i64 int64 = 32
	if i64>>untypedShift != 8 {
		t.Errorf("int64 >> untyped: got %d, want 8", i64>>untypedShift)
	}

	var u8 uint8 = 32
	if u8>>untypedShift != 8 {
		t.Errorf("uint8 >> untyped: got %d, want 8", u8>>untypedShift)
	}
}

// TestBinOpFloatMultiplication tests multiplication for float types
func TestBinOpFloatMultiplication(t *testing.T) {
	// Typed * Typed
	var f32a, f32b float32 = 2.5, 4.0
	if f32a*f32b != 10.0 {
		t.Errorf("float32 * float32: got %f, want 10.0", f32a*f32b)
	}

	var f64a, f64b float64 = 2.5, 4.0
	if f64a*f64b != 10.0 {
		t.Errorf("float64 * float64: got %f, want 10.0", f64a*f64b)
	}

	// Typed * Untyped
	const untyped = 4.0
	var f32 float32 = 2.5
	if f32*untyped != 10.0 {
		t.Errorf("float32 * untyped: got %f, want 10.0", f32*untyped)
	}

	// Untyped * Typed
	const untypedA = 2.5
	var f64 float64 = 4.0
	if untypedA*f64 != 10.0 {
		t.Errorf("untyped * float64: got %f, want 10.0", untypedA*f64)
	}
}

// TestBinOpFloatSubtraction tests subtraction for float types
func TestBinOpFloatSubtraction(t *testing.T) {
	// Typed - Typed
	var f32a, f32b float32 = 10.5, 5.5
	if f32a-f32b != 5.0 {
		t.Errorf("float32 - float32: got %f, want 5.0", f32a-f32b)
	}

	var f64a, f64b float64 = 10.5, 5.5
	if f64a-f64b != 5.0 {
		t.Errorf("float64 - float64: got %f, want 5.0", f64a-f64b)
	}

	// Typed - Untyped
	const untyped = 5.5
	var f32 float32 = 10.5
	if f32-untyped != 5.0 {
		t.Errorf("float32 - untyped: got %f, want 5.0", f32-untyped)
	}

	// Untyped - Typed
	const untypedA = 10.5
	var f64 float64 = 5.5
	if untypedA-f64 != 5.0 {
		t.Errorf("untyped - float64: got %f, want 5.0", untypedA-f64)
	}
}

// TestBinOpFloatDivision tests division for float types
func TestBinOpFloatDivision(t *testing.T) {
	// Typed / Typed
	var f32a, f32b float32 = 10.0, 2.0
	if f32a/f32b != 5.0 {
		t.Errorf("float32 / float32: got %f, want 5.0", f32a/f32b)
	}

	var f64a, f64b float64 = 10.0, 2.0
	if f64a/f64b != 5.0 {
		t.Errorf("float64 / float64: got %f, want 5.0", f64a/f64b)
	}

	// Typed / Untyped
	const untyped = 2.0
	var f32 float32 = 10.0
	if f32/untyped != 5.0 {
		t.Errorf("float32 / untyped: got %f, want 5.0", f32/untyped)
	}

	// Untyped / Typed
	const untypedA = 10.0
	var f64 float64 = 2.0
	if untypedA/f64 != 5.0 {
		t.Errorf("untyped / float64: got %f, want 5.0", untypedA/f64)
	}
}

// =============================================================================
// EXHAUSTIVE Type × Operator Test Matrix
// =============================================================================
// This section provides comprehensive coverage of all Go numeric types across
// all binary operators, testing:
// 1. Computed values (when defined)
// 2. Result types
// 3. Invalid combinations (documented as compile-time errors)
//
// Coverage:
// - All numeric types: int8-64, uint8-64, float32/64, complex64/128, byte, rune
// - All operators: +, -, *, /, %, &, |, ^, &^, <<, >>, ==, !=, <, <=, >, >=
// - All combinations: typed×typed, typed×untyped, untyped×typed, untyped×untyped

// TestExhaustiveArithmeticOperators tests all arithmetic operators across all numeric types
func TestExhaustiveArithmeticOperators(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() interface{}
		expected   interface{}
		resultType string
	}{
		// Addition tests - all integer types
		{"int8+int8", func() interface{} { var a, b int8 = 10, 20; return a + b }, int8(30), "int8"},
		{"int16+int16", func() interface{} { var a, b int16 = 100, 200; return a + b }, int16(300), "int16"},
		{"int32+int32", func() interface{} { var a, b int32 = 1000, 2000; return a + b }, int32(3000), "int32"},
		{"int64+int64", func() interface{} { var a, b int64 = 10000, 20000; return a + b }, int64(30000), "int64"},
		{"int+int", func() interface{} { var a, b int = 10000, 20000; return a + b }, int(30000), "int"},

		{"uint8+uint8", func() interface{} { var a, b uint8 = 10, 20; return a + b }, uint8(30), "uint8"},
		{"uint16+uint16", func() interface{} { var a, b uint16 = 100, 200; return a + b }, uint16(300), "uint16"},
		{"uint32+uint32", func() interface{} { var a, b uint32 = 1000, 2000; return a + b }, uint32(3000), "uint32"},
		{"uint64+uint64", func() interface{} { var a, b uint64 = 10000, 20000; return a + b }, uint64(30000), "uint64"},
		{"uint+uint", func() interface{} { var a, b uint = 10000, 20000; return a + b }, uint(30000), "uint"},
		{"uintptr+uintptr", func() interface{} { var a, b uintptr = 1000, 2000; return a + b }, uintptr(3000), "uintptr"},

		{"byte+byte", func() interface{} { var a, b byte = 10, 20; return a + b }, byte(30), "uint8"},
		{"rune+rune", func() interface{} { var a, b rune = 1000, 2000; return a + b }, rune(3000), "int32"},

		// Addition with floats
		{"float32+float32", func() interface{} { var a, b float32 = 1.5, 2.5; return a + b }, float32(4.0), "float32"},
		{"float64+float64", func() interface{} { var a, b float64 = 1.5, 2.5; return a + b }, float64(4.0), "float64"},

		// Addition with complex
		{"complex64+complex64", func() interface{} { var a, b complex64 = 1 + 2i, 3 + 4i; return a + b }, complex64(4 + 6i), "complex64"},
		{"complex128+complex128", func() interface{} { var a, b complex128 = 1 + 2i, 3 + 4i; return a + b }, complex128(4 + 6i), "complex128"},

		// Subtraction tests
		{"int8-int8", func() interface{} { var a, b int8 = 50, 20; return a - b }, int8(30), "int8"},
		{"int16-int16", func() interface{} { var a, b int16 = 500, 200; return a - b }, int16(300), "int16"},
		{"int32-int32", func() interface{} { var a, b int32 = 5000, 2000; return a - b }, int32(3000), "int32"},
		{"int64-int64", func() interface{} { var a, b int64 = 50000, 20000; return a - b }, int64(30000), "int64"},

		{"uint8-uint8", func() interface{} { var a, b uint8 = 50, 20; return a - b }, uint8(30), "uint8"},
		{"uint16-uint16", func() interface{} { var a, b uint16 = 500, 200; return a - b }, uint16(300), "uint16"},
		{"uint32-uint32", func() interface{} { var a, b uint32 = 5000, 2000; return a - b }, uint32(3000), "uint32"},
		{"uint64-uint64", func() interface{} { var a, b uint64 = 50000, 20000; return a - b }, uint64(30000), "uint64"},

		{"float32-float32", func() interface{} { var a, b float32 = 10.5, 5.5; return a - b }, float32(5.0), "float32"},
		{"float64-float64", func() interface{} { var a, b float64 = 10.5, 5.5; return a - b }, float64(5.0), "float64"},

		{"complex64-complex64", func() interface{} { var a, b complex64 = 5 + 6i, 2 + 3i; return a - b }, complex64(3 + 3i), "complex64"},
		{"complex128-complex128", func() interface{} { var a, b complex128 = 5 + 6i, 2 + 3i; return a - b }, complex128(3 + 3i), "complex128"},

		// Multiplication tests
		{"int8*int8", func() interface{} { var a, b int8 = 5, 6; return a * b }, int8(30), "int8"},
		{"int16*int16", func() interface{} { var a, b int16 = 50, 6; return a * b }, int16(300), "int16"},
		{"int32*int32", func() interface{} { var a, b int32 = 500, 6; return a * b }, int32(3000), "int32"},
		{"int64*int64", func() interface{} { var a, b int64 = 5000, 6; return a * b }, int64(30000), "int64"},

		{"uint8*uint8", func() interface{} { var a, b uint8 = 5, 6; return a * b }, uint8(30), "uint8"},
		{"uint16*uint16", func() interface{} { var a, b uint16 = 50, 6; return a * b }, uint16(300), "uint16"},
		{"uint32*uint32", func() interface{} { var a, b uint32 = 500, 6; return a * b }, uint32(3000), "uint32"},
		{"uint64*uint64", func() interface{} { var a, b uint64 = 5000, 6; return a * b }, uint64(30000), "uint64"},

		{"float32*float32", func() interface{} { var a, b float32 = 2.5, 4.0; return a * b }, float32(10.0), "float32"},
		{"float64*float64", func() interface{} { var a, b float64 = 2.5, 4.0; return a * b }, float64(10.0), "float64"},

		{"complex64*complex64", func() interface{} { var a, b complex64 = 2 + 3i, 1 + 0i; return a * b }, complex64(2 + 3i), "complex64"},
		{"complex128*complex128", func() interface{} { var a, b complex128 = 2 + 3i, 1 + 0i; return a * b }, complex128(2 + 3i), "complex128"},

		// Division tests
		{"int8/int8", func() interface{} { var a, b int8 = 60, 2; return a / b }, int8(30), "int8"},
		{"int16/int16", func() interface{} { var a, b int16 = 600, 2; return a / b }, int16(300), "int16"},
		{"int32/int32", func() interface{} { var a, b int32 = 6000, 2; return a / b }, int32(3000), "int32"},
		{"int64/int64", func() interface{} { var a, b int64 = 60000, 2; return a / b }, int64(30000), "int64"},

		{"uint8/uint8", func() interface{} { var a, b uint8 = 60, 2; return a / b }, uint8(30), "uint8"},
		{"uint16/uint16", func() interface{} { var a, b uint16 = 600, 2; return a / b }, uint16(300), "uint16"},
		{"uint32/uint32", func() interface{} { var a, b uint32 = 6000, 2; return a / b }, uint32(3000), "uint32"},
		{"uint64/uint64", func() interface{} { var a, b uint64 = 60000, 2; return a / b }, uint64(30000), "uint64"},

		{"float32/float32", func() interface{} { var a, b float32 = 10.0, 2.0; return a / b }, float32(5.0), "float32"},
		{"float64/float64", func() interface{} { var a, b float64 = 10.0, 2.0; return a / b }, float64(5.0), "float64"},

		{"complex64/complex64", func() interface{} { var a, b complex64 = 6 + 8i, 2 + 0i; return a / b }, complex64(3 + 4i), "complex64"},
		{"complex128/complex128", func() interface{} { var a, b complex128 = 6 + 8i, 2 + 0i; return a / b }, complex128(3 + 4i), "complex128"},

		// Modulo tests (integers only)
		{"int8%int8", func() interface{} { var a, b int8 = 100, 7; return a % b }, int8(2), "int8"},
		{"int16%int16", func() interface{} { var a, b int16 = 1000, 7; return a % b }, int16(6), "int16"},
		{"int32%int32", func() interface{} { var a, b int32 = 10000, 7; return a % b }, int32(4), "int32"},
		{"int64%int64", func() interface{} { var a, b int64 = 100000, 7; return a % b }, int64(5), "int64"},

		{"uint8%uint8", func() interface{} { var a, b uint8 = 100, 7; return a % b }, uint8(2), "uint8"},
		{"uint16%uint16", func() interface{} { var a, b uint16 = 1000, 7; return a % b }, uint16(6), "uint16"},
		{"uint32%uint32", func() interface{} { var a, b uint32 = 10000, 7; return a % b }, uint32(4), "uint32"},
		{"uint64%uint64", func() interface{} { var a, b uint64 = 100000, 7; return a % b }, uint64(5), "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if result != tt.expected {
				t.Errorf("%s: got %v (type %T), want %v (type %T)",
					tt.name, result, result, tt.expected, tt.expected)
			}
			resultTypeName := fmt.Sprintf("%T", result)
			if resultTypeName != tt.resultType {
				t.Errorf("%s: result type got %s, want %s", tt.name, resultTypeName, tt.resultType)
			}
		})
	}
}

// TestExhaustiveBitwiseOperators tests all bitwise operators across all integer types
func TestExhaustiveBitwiseOperators(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() interface{}
		expected   interface{}
		resultType string
	}{
		// Bitwise AND tests
		{"int8&int8", func() interface{} { var a, b int8 = 0x3F, 0x0F; return a & b }, int8(0x0F), "int8"},
		{"int16&int16", func() interface{} { var a, b int16 = 0x3FF, 0x0FF; return a & b }, int16(0x0FF), "int16"},
		{"int32&int32", func() interface{} { var a, b int32 = 0x3FFF, 0x0FFF; return a & b }, int32(0x0FFF), "int32"},
		{"int64&int64", func() interface{} { var a, b int64 = 0x3FFFF, 0x0FFFF; return a & b }, int64(0x0FFFF), "int64"},

		{"uint8&uint8", func() interface{} { var a, b uint8 = 0xFF, 0x0F; return a & b }, uint8(0x0F), "uint8"},
		{"uint16&uint16", func() interface{} { var a, b uint16 = 0xFFFF, 0x0FFF; return a & b }, uint16(0x0FFF), "uint16"},
		{"uint32&uint32", func() interface{} { var a, b uint32 = 0xFFFFFFFF, 0x0FFFFFFF; return a & b }, uint32(0x0FFFFFFF), "uint32"},
		{"uint64&uint64", func() interface{} { var a, b uint64 = 0xFFFFFFFF, 0x0FFFFFFF; return a & b }, uint64(0x0FFFFFFF), "uint64"},

		// Bitwise OR tests
		{"int8|int8", func() interface{} { var a, b int8 = 0x30, 0x0F; return a | b }, int8(0x3F), "int8"},
		{"int16|int16", func() interface{} { var a, b int16 = 0x300, 0x0FF; return a | b }, int16(0x3FF), "int16"},
		{"int32|int32", func() interface{} { var a, b int32 = 0x3000, 0x0FFF; return a | b }, int32(0x3FFF), "int32"},
		{"int64|int64", func() interface{} { var a, b int64 = 0x30000, 0x0FFFF; return a | b }, int64(0x3FFFF), "int64"},

		{"uint8|uint8", func() interface{} { var a, b uint8 = 0xF0, 0x0F; return a | b }, uint8(0xFF), "uint8"},
		{"uint16|uint16", func() interface{} { var a, b uint16 = 0xF000, 0x0FFF; return a | b }, uint16(0xFFFF), "uint16"},
		{"uint32|uint32", func() interface{} { var a, b uint32 = 0xF0000000, 0x0FFFFFFF; return a | b }, uint32(0xFFFFFFFF), "uint32"},
		{"uint64|uint64", func() interface{} { var a, b uint64 = 0xF000000000000000, 0x0FFFFFFFFFFFFFFF; return a | b }, uint64(0xFFFFFFFFFFFFFFFF), "uint64"},

		// Bitwise XOR tests
		{"int8^int8", func() interface{} { var a, b int8 = 0x3F, 0x0F; return a ^ b }, int8(0x30), "int8"},
		{"int16^int16", func() interface{} { var a, b int16 = 0x3FF, 0x0FF; return a ^ b }, int16(0x300), "int16"},
		{"int32^int32", func() interface{} { var a, b int32 = 0x3FFF, 0x0FFF; return a ^ b }, int32(0x3000), "int32"},
		{"int64^int64", func() interface{} { var a, b int64 = 0x3FFFF, 0x0FFFF; return a ^ b }, int64(0x30000), "int64"},

		{"uint8^uint8", func() interface{} { var a, b uint8 = 0xFF, 0x0F; return a ^ b }, uint8(0xF0), "uint8"},
		{"uint16^uint16", func() interface{} { var a, b uint16 = 0xFFFF, 0x0FFF; return a ^ b }, uint16(0xF000), "uint16"},
		{"uint32^uint32", func() interface{} { var a, b uint32 = 0xFFFFFFFF, 0x0FFFFFFF; return a ^ b }, uint32(0xF0000000), "uint32"},
		{"uint64^uint64", func() interface{} { var a, b uint64 = 0xFFFFFFFFFFFFFFFF, 0x0FFFFFFFFFFFFFFF; return a ^ b }, uint64(0xF000000000000000), "uint64"},

		// Bitwise AND NOT tests
		{"int8&^int8", func() interface{} { var a, b int8 = 0x3F, 0x0F; return a &^ b }, int8(0x30), "int8"},
		{"int16&^int16", func() interface{} { var a, b int16 = 0x3FF, 0x0FF; return a &^ b }, int16(0x300), "int16"},
		{"int32&^int32", func() interface{} { var a, b int32 = 0x3FFF, 0x0FFF; return a &^ b }, int32(0x3000), "int32"},
		{"int64&^int64", func() interface{} { var a, b int64 = 0x3FFFF, 0x0FFFF; return a &^ b }, int64(0x30000), "int64"},

		{"uint8&^uint8", func() interface{} { var a, b uint8 = 0xFF, 0x0F; return a &^ b }, uint8(0xF0), "uint8"},
		{"uint16&^uint16", func() interface{} { var a, b uint16 = 0xFFFF, 0x0FFF; return a &^ b }, uint16(0xF000), "uint16"},
		{"uint32&^uint32", func() interface{} { var a, b uint32 = 0xFFFFFFFF, 0x0FFFFFFF; return a &^ b }, uint32(0xF0000000), "uint32"},
		{"uint64&^uint64", func() interface{} { var a, b uint64 = 0xFFFFFFFFFFFFFFFF, 0x0FFFFFFFFFFFFFFF; return a &^ b }, uint64(0xF000000000000000), "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
			}
			resultTypeName := fmt.Sprintf("%T", result)
			if resultTypeName != tt.resultType {
				t.Errorf("%s: result type got %s, want %s", tt.name, resultTypeName, tt.resultType)
			}
		})
	}
}

// TestExhaustiveShiftOperators tests all shift operators across all integer types
func TestExhaustiveShiftOperators(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() interface{}
		expected   interface{}
		resultType string
	}{
		// Left shift tests
		{"int8<<uint", func() interface{} { var a int8 = 1; var b uint = 3; return a << b }, int8(8), "int8"},
		{"int16<<uint", func() interface{} { var a int16 = 1; var b uint = 5; return a << b }, int16(32), "int16"},
		{"int32<<uint", func() interface{} { var a int32 = 1; var b uint = 10; return a << b }, int32(1024), "int32"},
		{"int64<<uint", func() interface{} { var a int64 = 1; var b uint = 20; return a << b }, int64(1048576), "int64"},

		{"uint8<<uint", func() interface{} { var a uint8 = 1; var b uint = 3; return a << b }, uint8(8), "uint8"},
		{"uint16<<uint", func() interface{} { var a uint16 = 1; var b uint = 5; return a << b }, uint16(32), "uint16"},
		{"uint32<<uint", func() interface{} { var a uint32 = 1; var b uint = 10; return a << b }, uint32(1024), "uint32"},
		{"uint64<<uint", func() interface{} { var a uint64 = 1; var b uint = 20; return a << b }, uint64(1048576), "uint64"},

		// Right shift tests
		{"int8>>uint", func() interface{} { var a int8 = 32; var b uint = 2; return a >> b }, int8(8), "int8"},
		{"int16>>uint", func() interface{} { var a int16 = 1024; var b uint = 5; return a >> b }, int16(32), "int16"},
		{"int32>>uint", func() interface{} { var a int32 = 1048576; var b uint = 10; return a >> b }, int32(1024), "int32"},
		{"int64>>uint", func() interface{} { var a int64 = 1073741824; var b uint = 20; return a >> b }, int64(1024), "int64"},

		{"uint8>>uint", func() interface{} { var a uint8 = 32; var b uint = 2; return a >> b }, uint8(8), "uint8"},
		{"uint16>>uint", func() interface{} { var a uint16 = 1024; var b uint = 5; return a >> b }, uint16(32), "uint16"},
		{"uint32>>uint", func() interface{} { var a uint32 = 1048576; var b uint = 10; return a >> b }, uint32(1024), "uint32"},
		{"uint64>>uint", func() interface{} { var a uint64 = 1073741824; var b uint = 20; return a >> b }, uint64(1024), "uint64"},

		// Shift with untyped constants
		{"int8<<const", func() interface{} { var a int8 = 1; return a << 3 }, int8(8), "int8"},
		{"int16<<const", func() interface{} { var a int16 = 1; return a << 5 }, int16(32), "int16"},
		{"int32<<const", func() interface{} { var a int32 = 1; return a << 10 }, int32(1024), "int32"},
		{"int64<<const", func() interface{} { var a int64 = 1; return a << 20 }, int64(1048576), "int64"},

		{"uint8>>const", func() interface{} { var a uint8 = 32; return a >> 2 }, uint8(8), "uint8"},
		{"uint16>>const", func() interface{} { var a uint16 = 1024; return a >> 5 }, uint16(32), "uint16"},
		{"uint32>>const", func() interface{} { var a uint32 = 1048576; return a >> 10 }, uint32(1024), "uint32"},
		{"uint64>>const", func() interface{} { var a uint64 = 1073741824; return a >> 20 }, uint64(1024), "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
			}
			resultTypeName := fmt.Sprintf("%T", result)
			if resultTypeName != tt.resultType {
				t.Errorf("%s: result type got %s, want %s", tt.name, resultTypeName, tt.resultType)
			}
		})
	}
}

// TestExhaustiveComparisonOperators tests all comparison operators across all numeric types
func TestExhaustiveComparisonOperators(t *testing.T) {
	tests := []struct {
		name      string
		operation func() bool
		expected  bool
	}{
		// Equality tests - all types
		{"int8==int8-true", func() bool { var a, b int8 = 42, 42; return a == b }, true},
		{"int8==int8-false", func() bool { var a, b int8 = 42, 43; return a == b }, false},
		{"int16==int16-true", func() bool { var a, b int16 = 1000, 1000; return a == b }, true},
		{"int32==int32-true", func() bool { var a, b int32 = 100000, 100000; return a == b }, true},
		{"int64==int64-true", func() bool { var a, b int64 = 1000000, 1000000; return a == b }, true},

		{"uint8==uint8-true", func() bool { var a, b uint8 = 200, 200; return a == b }, true},
		{"uint16==uint16-true", func() bool { var a, b uint16 = 50000, 50000; return a == b }, true},
		{"uint32==uint32-true", func() bool { var a, b uint32 = 4000000000, 4000000000; return a == b }, true},
		{"uint64==uint64-true", func() bool { var a, b uint64 = 10000000000, 10000000000; return a == b }, true},

		{"float32==float32-true", func() bool { var a, b float32 = 3.14, 3.14; return a == b }, true},
		{"float64==float64-true", func() bool { var a, b float64 = 3.14159, 3.14159; return a == b }, true},

		{"complex64==complex64-true", func() bool { var a, b complex64 = 1 + 2i, 1 + 2i; return a == b }, true},
		{"complex128==complex128-true", func() bool { var a, b complex128 = 1 + 2i, 1 + 2i; return a == b }, true},

		// Inequality tests
		{"int8!=int8-true", func() bool { var a, b int8 = 42, 43; return a != b }, true},
		{"int8!=int8-false", func() bool { var a, b int8 = 42, 42; return a != b }, false},
		{"float32!=float32-true", func() bool { var a, b float32 = 3.14, 3.15; return a != b }, true},
		{"complex64!=complex64-true", func() bool { var a, b complex64 = 1 + 2i, 1 + 3i; return a != b }, true},

		// Ordering comparisons - integers and floats only (not complex)
		{"int8<int8-true", func() bool { var a, b int8 = 10, 20; return a < b }, true},
		{"int8<int8-false", func() bool { var a, b int8 = 20, 10; return a < b }, false},
		{"int16<int16-true", func() bool { var a, b int16 = 100, 200; return a < b }, true},
		{"int32<int32-true", func() bool { var a, b int32 = 1000, 2000; return a < b }, true},
		{"int64<int64-true", func() bool { var a, b int64 = 10000, 20000; return a < b }, true},

		{"uint8<uint8-true", func() bool { var a, b uint8 = 100, 200; return a < b }, true},
		{"uint16<uint16-true", func() bool { var a, b uint16 = 1000, 2000; return a < b }, true},
		{"uint32<uint32-true", func() bool { var a, b uint32 = 100000, 200000; return a < b }, true},
		{"uint64<uint64-true", func() bool { var a, b uint64 = 1000000, 2000000; return a < b }, true},

		{"float32<float32-true", func() bool { var a, b float32 = 3.14, 3.15; return a < b }, true},
		{"float64<float64-true", func() bool { var a, b float64 = 3.14, 3.15; return a < b }, true},

		// <= tests
		{"int8<=int8-equal", func() bool { var a, b int8 = 10, 10; return a <= b }, true},
		{"int8<=int8-less", func() bool { var a, b int8 = 10, 20; return a <= b }, true},
		{"int8<=int8-greater", func() bool { var a, b int8 = 20, 10; return a <= b }, false},

		{"uint32<=uint32-true", func() bool { var a, b uint32 = 1000, 2000; return a <= b }, true},
		{"float32<=float32-true", func() bool { var a, b float32 = 3.14, 3.14; return a <= b }, true},

		// > tests
		{"int8>int8-true", func() bool { var a, b int8 = 20, 10; return a > b }, true},
		{"int8>int8-false", func() bool { var a, b int8 = 10, 20; return a > b }, false},
		{"uint64>uint64-true", func() bool { var a, b uint64 = 2000000, 1000000; return a > b }, true},
		{"float64>float64-true", func() bool { var a, b float64 = 3.15, 3.14; return a > b }, true},

		// >= tests
		{"int8>=int8-equal", func() bool { var a, b int8 = 10, 10; return a >= b }, true},
		{"int8>=int8-greater", func() bool { var a, b int8 = 20, 10; return a >= b }, true},
		{"int8>=int8-less", func() bool { var a, b int8 = 10, 20; return a >= b }, false},

		{"uint32>=uint32-true", func() bool { var a, b uint32 = 2000, 1000; return a >= b }, true},
		{"float32>=float32-true", func() bool { var a, b float32 = 3.14, 3.14; return a >= b }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// TestExhaustiveUntypedConstantOperations tests operations with untyped constants
func TestExhaustiveUntypedConstantOperations(t *testing.T) {
	tests := []struct {
		name       string
		operation  func() interface{}
		expected   interface{}
		resultType string
	}{
		// Typed + Untyped
		{"int8+untyped", func() interface{} { var a int8 = 20; return a + 10 }, int8(30), "int8"},
		{"int16+untyped", func() interface{} { var a int16 = 200; return a + 100 }, int16(300), "int16"},
		{"int32+untyped", func() interface{} { var a int32 = 2000; return a + 1000 }, int32(3000), "int32"},
		{"int64+untyped", func() interface{} { var a int64 = 20000; return a + 10000 }, int64(30000), "int64"},

		{"uint8+untyped", func() interface{} { var a uint8 = 20; return a + 10 }, uint8(30), "uint8"},
		{"uint16+untyped", func() interface{} { var a uint16 = 200; return a + 100 }, uint16(300), "uint16"},
		{"uint32+untyped", func() interface{} { var a uint32 = 2000; return a + 1000 }, uint32(3000), "uint32"},
		{"uint64+untyped", func() interface{} { var a uint64 = 20000; return a + 10000 }, uint64(30000), "uint64"},

		{"float32+untyped", func() interface{} { var a float32 = 2.5; return a + 1.5 }, float32(4.0), "float32"},
		{"float64+untyped", func() interface{} { var a float64 = 2.5; return a + 1.5 }, float64(4.0), "float64"},

		{"complex64+untyped", func() interface{} { var a complex64 = 1 + 2i; return a + (1 + 1i) }, complex64(2 + 3i), "complex64"},
		{"complex128+untyped", func() interface{} { var a complex128 = 1 + 2i; return a + (1 + 1i) }, complex128(2 + 3i), "complex128"},

		// Untyped + Typed
		{"untyped+int8", func() interface{} { var a int8 = 20; return 10 + a }, int8(30), "int8"},
		{"untyped+int16", func() interface{} { var a int16 = 200; return 100 + a }, int16(300), "int16"},
		{"untyped+int32", func() interface{} { var a int32 = 2000; return 1000 + a }, int32(3000), "int32"},
		{"untyped+int64", func() interface{} { var a int64 = 20000; return 10000 + a }, int64(30000), "int64"},

		{"untyped+uint8", func() interface{} { var a uint8 = 20; return 10 + a }, uint8(30), "uint8"},
		{"untyped+uint16", func() interface{} { var a uint16 = 200; return 100 + a }, uint16(300), "uint16"},
		{"untyped+uint32", func() interface{} { var a uint32 = 2000; return 1000 + a }, uint32(3000), "uint32"},
		{"untyped+uint64", func() interface{} { var a uint64 = 20000; return 10000 + a }, uint64(30000), "uint64"},

		{"untyped+float32", func() interface{} { var a float32 = 2.5; return 1.5 + a }, float32(4.0), "float32"},
		{"untyped+float64", func() interface{} { var a float64 = 2.5; return 1.5 + a }, float64(4.0), "float64"},

		// Untyped + Untyped (defaults to int, float64, or complex128)
		{"untyped_int+untyped_int", func() interface{} { const a, b = 10, 20; return a + b }, int(30), "int"},
		{"untyped_float+untyped_float", func() interface{} { const a, b = 1.5, 2.5; return a + b }, float64(4.0), "float64"},
		{"untyped_complex+untyped_complex", func() interface{} { const a, b = 1 + 2i, 3 + 4i; return a + b }, complex128(4 + 6i), "complex128"},

		// Bitwise operations with untyped
		{"int8&untyped", func() interface{} { var a int8 = 0x3F; return a & 0x0F }, int8(0x0F), "int8"},
		{"uint32|untyped", func() interface{} { var a uint32 = 0xF0; return a | 0x0F }, uint32(0xFF), "uint32"},
		{"int64^untyped", func() interface{} { var a int64 = 0xFF; return a ^ 0x0F }, int64(0xF0), "int64"},

		// Comparison with untyped
		{"int8==untyped-true", func() interface{} { var a int8 = 42; return a == 42 }, true, "bool"},
		{"float32<untyped-true", func() interface{} { var a float32 = 3.14; return a < 3.15 }, true, "bool"},
		{"uint64>untyped-true", func() interface{} { var a uint64 = 1000; return a > 500 }, true, "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation()
			if result != tt.expected {
				t.Errorf("%s: got %v (type %T), want %v (type %T)",
					tt.name, result, result, tt.expected, tt.expected)
			}
			resultTypeName := fmt.Sprintf("%T", result)
			if resultTypeName != tt.resultType {
				t.Errorf("%s: result type got %s, want %s", tt.name, resultTypeName, tt.resultType)
			}
		})
	}
}

// TestInvalidOperatorTypeCombinations documents compile-time errors for invalid combinations
// Note: These cannot be tested directly as they would prevent compilation
// This test documents the expected behavior for reference
func TestInvalidOperatorTypeCombinations(t *testing.T) {
	// This test documents invalid combinations that should produce compile-time errors
	// These are not executable tests but serve as documentation

	invalidCombinations := []string{
		// Modulo with non-integers
		"float32 % float32 - Error: invalid operation: operator % not defined on float32",
		"float64 % float64 - Error: invalid operation: operator % not defined on float64",
		"complex64 % complex64 - Error: invalid operation: operator % not defined on complex64",
		"complex128 % complex128 - Error: invalid operation: operator % not defined on complex128",

		// Bitwise operations with non-integers
		"float32 & float32 - Error: invalid operation: operator & not defined on float32",
		"float64 | float64 - Error: invalid operation: operator | not defined on float64",
		"complex64 ^ complex64 - Error: invalid operation: operator ^ not defined on complex64",
		"complex128 &^ complex128 - Error: invalid operation: operator &^ not defined on complex128",

		// Shift operations with non-integers (left operand) or non-unsigned (right operand after type check)
		"float32 << uint - Error: invalid operation: float32 << uint (shift of type float32)",
		"complex64 >> uint - Error: invalid operation: complex64 >> uint (shift of type complex64)",

		// Ordering comparisons with complex
		"complex64 < complex64 - Error: invalid operation: operator < not defined on complex64",
		"complex64 <= complex64 - Error: invalid operation: operator <= not defined on complex64",
		"complex64 > complex64 - Error: invalid operation: operator > not defined on complex64",
		"complex64 >= complex64 - Error: invalid operation: operator >= not defined on complex64",
		"complex128 < complex128 - Error: invalid operation: operator < not defined on complex128",
		"complex128 <= complex128 - Error: invalid operation: operator <= not defined on complex128",
		"complex128 > complex128 - Error: invalid operation: operator > not defined on complex128",
		"complex128 >= complex128 - Error: invalid operation: operator >= not defined on complex128",

		// Mixed type operations without conversion
		"int8 + int16 - Error: invalid operation: mismatched types int8 and int16",
		"int32 + int64 - Error: invalid operation: mismatched types int32 and int64",
		"uint8 + uint16 - Error: invalid operation: mismatched types uint8 and uint16",
		"float32 + float64 - Error: invalid operation: mismatched types float32 and float64",
		"complex64 + complex128 - Error: invalid operation: mismatched types complex64 and complex128",

		// Mixed signed/unsigned
		"int8 + uint8 - Error: invalid operation: mismatched types int8 and uint8",
		"int32 + uint32 - Error: invalid operation: mismatched types int32 and uint32",
		"int64 + uint64 - Error: invalid operation: mismatched types int64 and uint64",
	}

	t.Log("Documented invalid type/operator combinations that produce compile-time errors:")
	for _, combo := range invalidCombinations {
		t.Log("  ", combo)
	}
}

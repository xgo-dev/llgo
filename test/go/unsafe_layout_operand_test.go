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

package gotest

import (
	"testing"
	"unsafe"
)

func layoutOperandValue[T any]() T    { panic("layout operand evaluated") }
func layoutOperandSlice[T any]() []T  { panic("layout slice evaluated") }
func layoutOperandIndex() int         { panic("layout index evaluated") }
func layoutOperandPointer[T any]() *T { panic("layout pointer evaluated") }

func layoutOperandResults[T any]() [8]uintptr {
	return [8]uintptr{
		unsafe.Sizeof(layoutOperandValue[T]()), unsafe.Alignof(layoutOperandValue[T]()),
		unsafe.Sizeof(layoutOperandSlice[T]()[layoutOperandIndex()]), unsafe.Alignof(layoutOperandSlice[T]()[layoutOperandIndex()]),
		unsafe.Sizeof(*layoutOperandPointer[T]()), unsafe.Alignof(*layoutOperandPointer[T]()),
		unsafe.Sizeof(struct{ Value T }{layoutOperandValue[T]()}), unsafe.Alignof(struct{ Value T }{layoutOperandValue[T]()}),
	}
}

func layoutOperandExpected[T any]() (uintptr, uintptr) {
	// Compare operand evaluation independently of existing target layout differences.
	var value T
	return unsafe.Sizeof(value), unsafe.Alignof(value)
}

func checkLayoutOperands[T any](t *testing.T) {
	t.Helper()
	size, alignment := layoutOperandExpected[T]()
	structSize, structAlignment := layoutOperandExpected[struct{ Value T }]()
	for i, got := range layoutOperandResults[T]() {
		wantSize, wantAlignment := size, alignment
		if i >= 6 {
			wantSize, wantAlignment = structSize, structAlignment
		}
		want := wantSize
		if i%2 == 1 {
			want = wantAlignment
		}
		if got != want {
			t.Fatalf("result %d = %d, want %d", i, got, want)
		}
	}
}

func TestGenericSizeofAlignofOperandsAreNotEvaluated(t *testing.T) {
	checkLayoutOperands[byte](t)
	checkLayoutOperands[int64](t)
	checkLayoutOperands[[3]int64](t)
	checkLayoutOperands[[]int64](t)
	checkLayoutOperands[*int64](t)
	checkLayoutOperands[func()](t)
	checkLayoutOperands[string](t)
	checkLayoutOperands[map[string]int](t)
	checkLayoutOperands[chan int](t)
	checkLayoutOperands[any](t)
	checkLayoutOperands[struct {
		Pad   byte
		Value int64
	}](t)
}

func layoutOperandEarlier[T any](calls *int) (uintptr, uintptr) {
	value := func() T { *calls++; var zero T; return zero }()
	return unsafe.Sizeof(value), unsafe.Alignof(value)
}

func TestGenericSizeofAlignofPreserveEarlierEvaluation(t *testing.T) {
	var calls int
	size, alignment := layoutOperandEarlier[int64](&calls)
	wantSize, wantAlignment := layoutOperandExpected[int64]()
	if calls != 1 || size != wantSize || alignment != wantAlignment {
		t.Fatalf("calls=%d, size=%d, alignment=%d, want %d/%d", calls, size, alignment, wantSize, wantAlignment)
	}
}

func layoutOperandBool() bool { panic("layout condition evaluated") }

func layoutOperandShortCircuit[T any]() (uintptr, uintptr) {
	return unsafe.Sizeof(struct {
			Value T
			Flag  bool
		}{Flag: layoutOperandBool() && layoutOperandBool()}),
		unsafe.Alignof(struct {
			Value T
			Flag  bool
		}{Flag: layoutOperandBool() || layoutOperandBool()})
}

func layoutOperandNil[T any]() (uintptr, uintptr) {
	var value *T
	return unsafe.Sizeof(*value), unsafe.Alignof(*value)
}

func TestGenericSizeofAlignofShortCircuitAndNil(t *testing.T) {
	size, alignment := layoutOperandShortCircuit[int64]()
	wantSize, wantAlignment := layoutOperandExpected[struct {
		Value int64
		Flag  bool
	}]()
	if size != wantSize || alignment != wantAlignment {
		t.Fatalf("short circuit size=%d, alignment=%d, want %d/%d", size, alignment, wantSize, wantAlignment)
	}
	size, alignment = layoutOperandNil[int64]()
	wantSize, wantAlignment = layoutOperandExpected[int64]()
	if size != wantSize || alignment != wantAlignment {
		t.Fatalf("nil size=%d, alignment=%d, want %d/%d", size, alignment, wantSize, wantAlignment)
	}
}

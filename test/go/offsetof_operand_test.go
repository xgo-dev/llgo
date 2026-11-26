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

type offsetOperandBox[T any] struct {
	Pad   byte
	Value T
}

func offsetOperandPointer[T any]() *offsetOperandBox[T] {
	panic("Offsetof operand evaluated")
}

func offsetOperandIndex() int {
	panic("Offsetof index evaluated")
}

func offsetOperandCall[T any]() uintptr {
	return unsafe.Offsetof(offsetOperandPointer[T]().Value)
}

func offsetOperandIndexed[T any]() uintptr {
	var boxes []offsetOperandBox[T]
	return unsafe.Offsetof(boxes[offsetOperandIndex()].Value)
}

func offsetOperandComposite[T any]() uintptr {
	return unsafe.Offsetof(struct {
		Pad   byte
		Value T
	}{Value: offsetOperandPointer[T]().Value}.Value)
}

func offsetOperandNil[T any]() uintptr {
	var box *offsetOperandBox[T]
	return unsafe.Offsetof(box.Value)
}

func TestGenericOffsetofOperandIsNotEvaluated(t *testing.T) {
	var boxes [2]offsetOperandBox[int64]
	want := uintptr(unsafe.Pointer(&boxes[0].Value)) - uintptr(unsafe.Pointer(&boxes[0]))
	for _, got := range []uintptr{
		offsetOperandCall[int64](),
		offsetOperandIndexed[int64](),
		offsetOperandNil[int64](),
	} {
		if got != want {
			t.Fatalf("Offsetof = %d, want %d", got, want)
		}
	}
	if got := offsetOperandComposite[int64](); got != want {
		t.Fatalf("Offsetof composite = %d, want %d", got, want)
	}
	if got := offsetOperandCall[byte](); got != 1 {
		t.Fatalf("Offsetof byte = %d, want 1", got)
	}
}

func offsetOperandEarlier[T any](calls *int) uintptr {
	box := func() *offsetOperandBox[T] {
		*calls++
		return nil
	}()
	return unsafe.Offsetof(box.Value)
}

func offsetOperandShortCircuit[T any]() uintptr {
	return unsafe.Offsetof(struct {
		Pad   byte
		Value T
		Flag  bool
	}{Flag: offsetOperandIndex() == 0 && offsetOperandIndex() == 1}.Value)
}

func offsetOperandEarlierLoad[T any]() uintptr {
	var holder *struct{ Box *offsetOperandBox[T] }
	box := holder.Box
	return unsafe.Offsetof(box.Value)
}

func TestGenericOffsetofPreservesEarlierLoad(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("earlier nil field read did not panic")
		}
	}()
	offsetOperandEarlierLoad[int64]()
}

func TestGenericOffsetofShortCircuitIsNotEvaluated(t *testing.T) {
	if got, want := offsetOperandShortCircuit[int64](), unsafe.Offsetof(offsetOperandBox[int64]{}.Value); got != want {
		t.Fatalf("Offsetof = %d, want %d", got, want)
	}
}

func TestGenericOffsetofPreservesEarlierEvaluation(t *testing.T) {
	var calls int
	got := offsetOperandEarlier[int64](&calls)
	if calls != 1 || got != unsafe.Offsetof(offsetOperandBox[int64]{}.Value) {
		t.Fatalf("calls = %d, Offsetof = %d", calls, got)
	}
}

type offsetOperandEmbedded[T any] struct {
	Pad byte
	offsetOperandBox[T]
}

func offsetOperandEmbeddedPointer[T any]() *offsetOperandEmbedded[T] {
	panic("Offsetof embedded operand evaluated")
}

func offsetOperandPromoted[T any]() uintptr {
	return unsafe.Offsetof(offsetOperandEmbeddedPointer[T]().Value)
}

func offsetOperandExplicit[T any]() uintptr {
	return unsafe.Offsetof(offsetOperandEmbeddedPointer[T]().offsetOperandBox.Value)
}

func TestGenericOffsetofFieldPaths(t *testing.T) {
	var embedded offsetOperandEmbedded[int64]
	want := uintptr(unsafe.Pointer(&embedded.Value)) - uintptr(unsafe.Pointer(&embedded))
	if got := offsetOperandPromoted[int64](); got != want {
		t.Fatalf("promoted Offsetof = %d, want %d", got, want)
	}
	want = unsafe.Offsetof(offsetOperandBox[int64]{}.Value)
	if got := offsetOperandExplicit[int64](); got != want {
		t.Fatalf("explicit Offsetof = %d, want %d", got, want)
	}
}

func offsetOperandPair[T any](ok bool, calls *int) (*offsetOperandBox[T], bool) {
	*calls++
	return nil, ok
}

func offsetOperandGet[T any](ok bool, box *offsetOperandBox[T]) *offsetOperandBox[T] {
	panic("Offsetof tuple operand evaluated")
}

func offsetOperandTupleBranch[T any](input bool, calls *int) (uintptr, int) {
	box, ok := offsetOperandPair[T](input, calls)
	value := 10
	if ok {
		value = 20
	}
	return unsafe.Offsetof(offsetOperandGet[T](ok, box).Value), value
}

func TestGenericOffsetofPreservesTupleBranch(t *testing.T) {
	var calls int
	for _, input := range []bool{false, true} {
		got, value := offsetOperandTupleBranch[int64](input, &calls)
		want := 10
		if input {
			want = 20
		}
		if got != unsafe.Offsetof(offsetOperandBox[int64]{}.Value) || value != want {
			t.Fatalf("input %v: Offsetof = %d, branch value = %d, want %d", input, got, value, want)
		}
	}
	if calls != 2 {
		t.Fatalf("pair calls = %d, want 2", calls)
	}
}

type offsetOperandNamedPointer[T any] *offsetOperandBox[T]

func offsetOperandNamed[T any]() uintptr {
	return unsafe.Offsetof(offsetOperandNamedPointer[T](offsetOperandPointer[T]()).Value)
}

func TestGenericOffsetofNamedPointer(t *testing.T) {
	if got, want := offsetOperandNamed[int64](), unsafe.Offsetof(offsetOperandBox[int64]{}.Value); got != want {
		t.Fatalf("named pointer Offsetof = %d, want %d", got, want)
	}
}

type offsetOperandFunctionBox[T any] struct {
	Pad      byte
	Function T
	Tail     byte
}

func offsetOperandAfterFunction[T any]() uintptr {
	var box *offsetOperandFunctionBox[T]
	return unsafe.Offsetof(box.Tail)
}

func TestGenericOffsetofAfterFunction(t *testing.T) {
	if got, want := offsetOperandAfterFunction[func()](), unsafe.Offsetof(offsetOperandFunctionBox[func()]{}.Tail); got != want {
		t.Fatalf("Offsetof after function = %d, want %d", got, want)
	}
}

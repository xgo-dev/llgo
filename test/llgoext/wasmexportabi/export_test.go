//go:build llgo && wasm

package wasmexportabi

import (
	"testing"
	"unsafe"
)

func TestWasmGoExports(t *testing.T) {
	if got := callGoExports(); got != 0 {
		t.Fatalf("C caller failed check %d (1: int, 2: uint, 3: slice)", got)
	}

	data := [5]byte{7, 8, 9, 10, 11}
	got := cSlice(data[:3])
	if len(got) != 3 || cap(got) != 4 || !slicePointerSlotOK(&got) || unsafe.SliceData(got) != &data[1] {
		t.Fatalf("slice returned by C: len=%d cap=%d, want 3 and 4 with data[1:]", len(got), cap(got))
	}
}

func TestWasmAggregateExports(t *testing.T) {
	if callGoMixed() != 1 {
		t.Fatal("Go export did not normalize the nested slice")
	}
	data := [5]byte{7, 8, 9, 10, 11}
	got := cMixed(Mixed{Text: "wasm", Data: data[:3]})
	if got.Text != "wasm" || len(got.Data) != 3 || cap(got.Data) != 4 ||
		!slicePointerSlotOK(&got.Data) || &got.Data[0] != &data[1] {
		t.Fatalf("unexpected mixed value returned by C: %+v", got)
	}
}

func TestWasmOverlappingHeaderCopies(t *testing.T) {
	got := []string{"a", "a", "b"}
	copy(got, got[1:])
	if got[0] != "a" || got[1] != "b" {
		t.Fatalf("overlapping copy produced %v", got)
	}
}

func TestWasmSliceCallbacks(t *testing.T) {
	for _, second := range []bool{false, true} {
		if !callCallbacks(second) {
			t.Fatalf("C callback failed with second=%v", second)
		}
	}
	data := [5]byte{7, 8, 9, 10, 11}
	got := callSliceFunc(cSlice, data[:3])
	if len(got) != 3 || cap(got) != 4 || !slicePointerSlotOK(&got) || &got[0] != &data[1] {
		t.Fatalf("unexpected slice from native function value: %v", got)
	}
}

//go:build llgo && wasm

package wasmexportabi

import "testing"

func TestWasmGoExports(t *testing.T) {
	if got := callGoExports(); got != 0 {
		t.Fatalf("C caller failed check %d (1: int, 2: uint, 3: slice)", got)
	}

	data := [5]byte{7, 8, 9, 10, 11}
	got := cSlice(data[:3])
	if len(got) != 3 || cap(got) != 4 || !slicePointerSlotOK(got) || &got[0] != &data[1] {
		t.Fatalf("slice returned by C: len=%d cap=%d, want 3 and 4 with data[1:]", len(got), cap(got))
	}
}

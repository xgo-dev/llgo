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
	if len(got) != 3 || cap(got) != 4 || !slicePointerSlotOK(&got) || &got[0] != &data[1] {
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

func TestWasmHostGlobalFields(t *testing.T) {
	dirtyHostHeaders()
	check := func(name string, text *string, data *[]byte, word uintptr) {
		t.Helper()
		if !stringPointerSlotOK(text) || *text != "wasm" {
			t.Errorf("%s lost the string pointer or payload", name)
		}
		if !slicePointerSlotOK(data) || len(*data) != 3 || cap(*data) != 5 || (*data)[0] != 7 {
			t.Errorf("%s lost the slice pointer, bounds, or payload", name)
		}
		if word != 0x123456789abcdef0 {
			t.Errorf("%s read uintptr %#x", name, word)
		}
	}
	text, data := hostGlobal.Text, hostGlobal.Data
	check("field read", &text, &data, hostGlobal.Word)
	whole := hostGlobal
	check("record copy", &whole.Text, &whole.Data, whole.Word)
	if got := plainGlobal.Word; got != whole.Word {
		t.Errorf("ordinary record read uintptr %#x, want %#x", got, whole.Word)
	}
	plainGlobal.Word = 7
	plain := plainGlobal
	if plain.Word != 7 {
		t.Errorf("ordinary record store left uintptr %#x, want 7", plain.Word)
	}
}

//go:noinline
func readHeader(p *string) string { return *p }

//go:noinline
func copyHeaders(dst, src []Mixed) int { return copy(dst, src) }

func TestWasmHostHeaderAliases(t *testing.T) {
	textPtr := dirtyHostHeaders()
	items := hostGlobal.Rows[0][:]
	raw := *(*uintptr)(unsafe.Pointer(&items[0].Text))
	if copyHeaders(items, items) != len(items) {
		t.Fatal("self-copy returned the wrong count")
	}
	check := func(m *Mixed) {
		t.Helper()
		if !stringPointerSlotOK(&m.Text) || !slicePointerSlotOK(&m.Data) ||
			m.Text != "wasm" || len(m.Data) != 3 || cap(m.Data) != 5 || m.Data[0] != 7 {
			t.Fatalf("incorrect copy of C string or slice: %+v", m)
		}
	}
	value := items[0]
	check(&value)
	header := readHeader(textPtr)
	if !stringPointerSlotOK(&header) || header != "wasm" {
		t.Fatal("string load through a pointer argument retained C padding")
	}
	var dst [3]Mixed
	dst[2].Text = "end"
	if n := copyHeaders(dst[:], items); n != 2 || dst[2].Text != "end" {
		t.Fatal("copy changed the destination outside the copied range")
	}
	check(&dst[0])
	check(&dst[1])
	var rows [1][2]Mixed
	copy(rows[:], hostGlobal.Rows[:])
	check(&rows[0][1])
	grown := append(items[:1:1], items[1:]...)
	check(&grown[0])
	check(&grown[1])
	reused := append(dst[:1], items...)
	check(&reused[1])
	check(&reused[2])
	inPlace := append(items[:0], items...)
	if len(inPlace) != len(items) || &inPlace[0] != &items[0] {
		t.Fatal("in-place append changed the backing store")
	}
	if after := *(*uintptr)(unsafe.Pointer(&items[0].Text)); after != raw {
		t.Fatal("reading or copying the C global rewrote the original prefix")
	}
}

func TestWasmOverlappingHeaderCopies(t *testing.T) {
	words := []string{"a", "b", "c"}
	got := append(words[:1], words[:2]...)
	if got[0] != "a" || got[1] != "a" || got[2] != "b" {
		t.Fatalf("overlapping append produced %v", got)
	}
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

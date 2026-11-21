//go:build llgo && wasm

package wasmexportabi

import (
	"reflect"
	"testing"
	"unsafe"
)

//llgo:cdecl
type reflectSlot struct {
	Data *byte
	Tag  uint32
}

type reflectTextRecord struct {
	Slot reflectSlot
	Len  int64
}

func TestWasmReflectHeaderCopies(t *testing.T) {
	dirtyHostHeaders()
	items := hostGlobal.Rows[0][:]
	src := reflect.ValueOf(items)
	raw := *(*uintptr)(unsafe.Pointer(&items[0].Text))
	reflect.Copy(src, src)
	src.Index(0).Set(src.Index(0))

	var copied [3]Mixed
	copied[2].Text = "end"
	if n := reflect.Copy(reflect.ValueOf(copied[:]), src); n != 2 || copied[2].Text != "end" {
		t.Fatal("reflect.Copy changed the destination outside the copied range")
	}
	var record hostRecord
	reflect.ValueOf(&record).Elem().Set(reflect.ValueOf(&hostGlobal).Elem())
	grown := reflect.AppendSlice(src.Slice3(0, 1, 1), src.Slice(1, 2)).Interface().([]Mixed)
	for _, value := range []*Mixed{&copied[0], &copied[1], &record.Mixed, &record.Rows[0][1], &grown[0], &grown[1]} {
		if !stringPointerSlotOK(&value.Text) || value.Text != "wasm" ||
			!slicePointerSlotOK(&value.Data) || len(value.Data) != 3 || cap(value.Data) != 5 {
			t.Fatalf("reflect copied incorrect string or slice data: %+v", value)
		}
	}
	if record.Word != 0x123456789abcdef0 || *(*uintptr)(unsafe.Pointer(&items[0].Text)) != raw {
		t.Fatal("reflect changed an adjacent word or the source padding")
	}

	// A C record with the string header's physical shape has a live Tag.
	input := reflectTextRecord{reflectSlot{&items[0].Data[0], 0x12345678}, 3}
	var output reflectTextRecord
	reflect.ValueOf(&output).Elem().Set(reflect.ValueOf(&input).Elem())
	if output != input {
		t.Fatal("reflect changed a live field in a header lookalike")
	}
}

func TestWasmReflectStridedArrayCopy(t *testing.T) {
	type row struct {
		Pair [2]Mixed
		Tag  uintptr
	}
	dirtyHostHeaders()
	// Copy the C record as bytes to preserve its padding in every source row.
	type rawPair [unsafe.Sizeof(row{}.Pair)]byte
	raw := *(*rawPair)(unsafe.Pointer(&hostGlobal.Rows[0]))
	var src, dst [2]row
	for i := range src {
		*(*rawPair)(unsafe.Pointer(&src[i].Pair)) = raw
		src[i].Tag = uintptr(i + 1)
	}
	reflect.Copy(reflect.ValueOf(dst[:]), reflect.ValueOf(src[:]))
	for i := range dst {
		if *(*rawPair)(unsafe.Pointer(&src[i].Pair)) != raw || dst[i].Tag != src[i].Tag {
			t.Fatal("reflect.Copy changed source bytes or an adjacent field")
		}
		for j := range dst[i].Pair {
			m := &dst[i].Pair[j]
			if !stringPointerSlotOK(&m.Text) || !slicePointerSlotOK(&m.Data) || m.Text != "wasm" || len(m.Data) != 3 {
				t.Fatalf("reflect.Copy stored incorrect headers in row %d element %d", i, j)
			}
		}
	}
}

func TestWasmReflectGrowPreservesCapacity(t *testing.T) {
	dirtyHostHeaders()
	items := hostGlobal.Rows[0][:]
	tail := *(*uintptr)(unsafe.Pointer(&items[1].Text))
	s := items[:1]
	v := reflect.ValueOf(&s).Elem()
	v.Grow(1)
	if len(s) != 1 || &s[0] != &items[0] {
		t.Fatal("Grow with spare capacity changed the slice")
	}
	v.Grow(2)
	if len(s) != 1 || cap(s) < 3 || &s[0] == &items[0] {
		t.Fatal("Grow returned an incorrect length, capacity or backing array")
	}
	for i := range items {
		m := &s[:2][i]
		if m.Text != "wasm" || len(m.Data) != 3 || !stringPointerSlotOK(&m.Text) || !slicePointerSlotOK(&m.Data) {
			t.Fatalf("Grow failed to preserve element %d within the old capacity", i)
		}
	}
	if *(*uintptr)(unsafe.Pointer(&items[1].Text)) != tail {
		t.Fatal("Grow changed the source tail padding")
	}
}

//go:build llgo && !wasm && !baremetal

package gotest

import (
	"encoding/json"
	"reflect"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

//llgo:cdecl
type metadataNativeTail struct {
	X float64
	Z struct{}
}

type metadataNativeOuter struct {
	N metadataNativeTail
	Y float64
}

//llgo:cdecl
type metadataNative128 struct {
	Data [16]uint64
	Z    struct{}
}

//llgo:cdecl
type metadataNativePadded struct {
	I int32
	U uint64
}

func TestNativeAggregateMetadataConsumers(t *testing.T) {
	items := [4]metadataNativeTail{{X: 1.25}, {X: 2.5}, {X: 3.75}, {X: 5}}
	objects := [2]metadataNativeOuter{{N: items[0], Y: 2.5}, {N: items[2], Y: 5}}
	nt, ot := reflect.TypeOf(items[0]), reflect.TypeOf(objects[0])
	stride := uintptr(unsafe.Pointer(&items[1])) - uintptr(unsafe.Pointer(&items[0]))
	outerStride := uintptr(unsafe.Pointer(&objects[1])) - uintptr(unsafe.Pointer(&objects[0]))
	yOffset := uintptr(unsafe.Pointer(&objects[0].Y)) - uintptr(unsafe.Pointer(&objects[0]))
	if nt.Size() != 8 || nt.Size() != stride || ot.Size() != 16 || ot.Size() != outerStride || ot.Field(1).Offset != yOffset || yOffset != 8 {
		t.Fatalf("native size/stride=%d/%d, outer size/stride=%d/%d, Y offset=%d/%d", nt.Size(), stride, ot.Size(), outerStride, ot.Field(1).Offset, yOffset)
	}
	if got := reflect.ArrayOf(2, nt).Size(); got != 16 {
		t.Fatalf("reflect ArrayOf size=%d, want 16", got)
	}
	dst := [4]metadataNativeTail{{}, {}, {X: 77}, {X: 88}}
	reflect.Copy(reflect.ValueOf(dst[:2]), reflect.ValueOf(items[:2]))
	if dst[0].X != 1.25 || dst[1].X != 2.5 || dst[2].X != 77 || dst[3].X != 88 {
		t.Fatalf("native copy values/guards=%v", dst)
	}
	if got := reflect.ValueOf(items).Index(1).Field(0).Float(); got != 2.5 {
		t.Fatalf("native reflect Index=%v, want 2.5", got)
	}
	slice := reflect.MakeSlice(reflect.TypeOf([]metadataNativeTail{}), 2, 2)
	slice.Index(1).Field(0).SetFloat(9)
	if got := slice.Interface().([]metadataNativeTail)[1].X; got != 9 {
		t.Fatalf("typed MakeSlice element=%v, want 9", got)
	}
	swap := items
	reflect.Swapper(swap[:2])(0, 1)
	if swap[0].X != 2.5 || swap[1].X != 1.25 || swap[2].X != 3.75 {
		t.Fatalf("native Swapper changed adjacent elements: %v", swap)
	}
	left, right := objects, objects
	right[0].Y = 9
	if reflect.DeepEqual(&left[0], &right[0]) {
		t.Fatal("DeepEqual missed the differing Y field")
	}
	encoded, err := json.Marshal(&objects[0])
	if err != nil || string(encoded) != `{"N":{"X":1.25,"Z":{}},"Y":2.5}` {
		t.Fatalf("native JSON=%s, err=%v", encoded, err)
	}

	f := func(v metadataNativeOuter, marker uint64) float64 { return v.N.X + v.Y + float64(marker) }
	got := reflect.ValueOf(f).Call([]reflect.Value{reflect.ValueOf(objects[0]), reflect.ValueOf(uint64(37))})[0].Float()
	if got != 40.75 {
		t.Fatalf("native reflect Call=%v, want 40.75", got)
	}
	made := reflect.MakeFunc(reflect.TypeOf(f), func(in []reflect.Value) []reflect.Value {
		v := in[0]
		if v.Field(0).Field(0).Float() != 1.25 || v.Field(1).Float() != 2.5 || in[1].Uint() != 37 {
			t.Fatal("MakeFunc decoded native aggregate arguments incorrectly")
		}
		return []reflect.Value{reflect.ValueOf(float64(40.75))}
	}).Interface().(func(metadataNativeOuter, uint64) float64)
	if got := made(objects[0], 37); got != 40.75 {
		t.Fatalf("native MakeFunc=%v, want 40.75", got)
	}
}

func TestNativeAggregateMapBoundary(t *testing.T) {
	var value any = (map[uint64]metadataNative128)(nil)
	mt := (*abi.MapType)((*[2]unsafe.Pointer)(unsafe.Pointer(&value))[0])
	if mt.Elem.Size_ != 128 || mt.IndirectElem() || mt.ValueSize != 128 {
		t.Fatalf("native map size=%d, indirect=%v, slot=%d", mt.Elem.Size_, mt.IndirectElem(), mt.ValueSize)
	}
	values := map[uint64]metadataNative128{}
	for i := uint64(0); i < 64; i++ {
		var v metadataNative128
		v.Data[0], v.Data[15] = i+11, i+99
		values[i] = v
	}
	for i := uint64(0); i < 64; i++ {
		v := values[i]
		if v.Data[0] != i+11 || v.Data[15] != i+99 {
			t.Fatalf("native map growth corrupted entry %d: %v", i, v.Data)
		}
	}

	var padded any = metadataNativePadded{}
	typ := (*abi.Type)((*[2]unsafe.Pointer)(unsafe.Pointer(&padded))[0])
	items := [2]metadataNativePadded{}
	stride := uintptr(unsafe.Pointer(&items[1])) - uintptr(unsafe.Pointer(&items[0]))
	if typ.Size_ != stride || uintptr(typ.Align_) != unsafe.Alignof(items[0]) || typ.FieldAlign_ != typ.Align_ {
		t.Fatalf("native alignment metadata disagrees with storage: size=%d stride=%d align=%d/%d", typ.Size_, stride, typ.Align_, unsafe.Alignof(items[0]))
	}
	if stride > unsafe.Sizeof(items[0].I)+unsafe.Sizeof(items[0].U) && typ.TFlag&abi.TFlagRegularMemory != 0 {
		t.Fatal("padded native storage is incorrectly marked RegularMemory")
	}
}

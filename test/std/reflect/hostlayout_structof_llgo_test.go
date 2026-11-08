//go:build llgo

package reflect_test

import (
	"reflect"
	"structs"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

type structOfHostMarker structs.HostLayout
type structOfHostMarkerAlias = structs.HostLayout

func TestHostLayoutStructOf(t *testing.T) {
	for _, value := range []any{
		struct {
			_    structs.HostLayout
			Tag  byte
			I16  int16
			I32  int32
			Int  int
			I    int64
			U    uint64
			F32  float32
			F    float64
			C64  complex64
			C    complex128
			A    [2]float64
			Tail byte
		}{},
		struct {
			_    structOfHostMarker
			Tag  byte
			F    float64
			Tail struct{}
		}{},
		struct {
			Tag    byte
			Marker structOfHostMarkerAlias
			F      float64
		}{},
		struct {
			Tag byte
			F   float64
			_   structs.HostLayout
		}{},
		struct {
			_     structs.HostLayout
			P     *byte
			U     uintptr
			Words [2]uintptr
			Tail  uint32
		}{},
		struct {
			_     structs.HostLayout
			Inner struct {
				B byte
				I int32
			}
			F float64
		}{},
	} {
		want := reflect.TypeOf(value)
		fields := make([]reflect.StructField, want.NumField())
		for i := range fields {
			fields[i] = want.Field(i)
		}
		got := reflect.StructOf(fields)
		if got.Size() != want.Size() || got.Align() != want.Align() {
			t.Fatalf("%s: dynamic size/align=%d/%d, compiled=%d/%d", want, got.Size(), got.Align(), want.Size(), want.Align())
		}
		for i := range fields {
			if got.Field(i).Offset != want.Field(i).Offset {
				t.Fatalf("%s field %d: dynamic offset=%d, compiled=%d", want, i, got.Field(i).Offset, want.Field(i).Offset)
			}
		}
	}
}

// Keep copied descriptors alive for the dynamically constructed types' lifetime.
var structOfHostScalarTypes []*abi.Type

func TestHostLayoutStructOfScalarAlignment(t *testing.T) {
	// ARM and Windows/386 report Go scalar alignment four while HostLayout
	// uses eight. Copy the descriptor to exercise that boundary on every host,
	// without changing any global scalar descriptor or requiring CPU emulation.
	for _, tc := range []struct {
		value any
		align uintptr
	}{
		{int64(0), unsafe.Alignof(struct {
			_ structs.HostLayout
			V int64
		}{})},
		{uint64(0), unsafe.Alignof(struct {
			_ structs.HostLayout
			V uint64
		}{})},
		{float64(0), unsafe.Alignof(struct {
			_ structs.HostLayout
			V float64
		}{})},
		{complex128(0), unsafe.Alignof(struct {
			_ structs.HostLayout
			V complex128
		}{})},
	} {
		scalar := reflect.TypeOf(tc.value)
		type typeInterface struct{ tab, data unsafe.Pointer }
		bits := (*typeInterface)(unsafe.Pointer(&scalar))
		copied := new(abi.Type)
		*copied = *(*abi.Type)(bits.data)
		copied.Align_, copied.FieldAlign_ = 4, 4
		structOfHostScalarTypes = append(structOfHostScalarTypes, copied)
		bits.data = unsafe.Pointer(copied)
		for _, fieldType := range []reflect.Type{scalar, reflect.ArrayOf(2, scalar)} {
			for _, marker := range []struct {
				typ  reflect.Type
				host bool
			}{
				{reflect.TypeOf(structs.HostLayout{}), true},
				{reflect.TypeOf(structOfHostMarker{}), true},
				{reflect.TypeOf(struct{ _ structs.HostLayout }{}), false},
				{reflect.TypeOf([0]structs.HostLayout{}), false},
			} {
				fields := []reflect.StructField{
					{Name: "Marker", Type: marker.typ},
					{Name: "StructOfHostTag", Type: reflect.TypeOf(byte(0))},
					{Name: "StructOfHostValue", Type: fieldType},
				}
				got := reflect.StructOf(fields)
				wantAlign := uintptr(4)
				if marker.host {
					wantAlign = tc.align
				}
				wantSize := (wantAlign + fieldType.Size() + wantAlign - 1) &^ (wantAlign - 1)
				if uintptr(got.Align()) != wantAlign || got.Field(2).Offset != wantAlign || got.Size() != wantSize {
					t.Fatalf("marker=%s, value=%s: size/align/offset=%d/%d/%d, want %d/%d/%d", marker.typ, fieldType, got.Size(), got.Align(), got.Field(2).Offset, wantSize, wantAlign, wantAlign)
				}
				if got.Field(2).Type.Size() != fieldType.Size() {
					t.Fatal("HostLayout changed field storage width")
				}
			}
		}
	}
}

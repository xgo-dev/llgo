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
type hostLayout struct{}
type structOfFakeHostMarker struct{ _ hostLayout }

type structOfHostWideInner struct {
	_ structs.HostLayout
	B byte
	F float64
	P *byte
}

//go:linkname structOfIsHostLayoutMarker reflect.isHostLayoutMarker
func structOfIsHostLayoutMarker(reflect.Type) bool

func TestHostLayoutStructOfMarkerIdentity(t *testing.T) {
	for _, tc := range []struct {
		typ  reflect.Type
		want bool
	}{
		{reflect.TypeOf(structs.HostLayout{}), true},
		{reflect.TypeOf(structOfHostMarker{}), true},
		{reflect.TypeOf(structOfHostMarkerAlias{}), true},
		{reflect.TypeOf(structOfFakeHostMarker{}), false},
		{reflect.TypeOf(struct{ _ structs.HostLayout }{}), false},
		{reflect.TypeOf([0]structs.HostLayout{}), false},
		{reflect.TypeOf(struct{ Value int }{}), false},
	} {
		if got := structOfIsHostLayoutMarker(tc.typ); got != tc.want {
			t.Fatalf("marker %s = %v, want %v", tc.typ, got, tc.want)
		}
		if n := testing.AllocsPerRun(100, func() { structOfIsHostLayoutMarker(tc.typ) }); n != 0 {
			t.Fatalf("marker %s allocated %v times, want zero", tc.typ, n)
		}
	}
}

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
		// Mark nested host records independently. Their descriptors already
		// describe their own host layout, including on ARM and Windows/386.
		struct {
			_     structs.HostLayout
			Inner structOfHostWideInner
			Tail  byte
		}{},
		struct {
			_     structs.HostLayout
			Items [2]structOfHostWideInner
			Tail  byte
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

func TestHostLayoutStructOfNestedLayout(t *testing.T) {
	// Model a valid nested Go descriptor with four-byte float64 alignment.
	// A parent marker must not rewrite that type or resize its array elements.
	scalar := reflect.TypeOf(float64(0))
	type typeInterface struct{ tab, data unsafe.Pointer }
	bits := (*typeInterface)(unsafe.Pointer(&scalar))
	copied := new(abi.Type)
	*copied = *(*abi.Type)(bits.data)
	copied.Align_, copied.FieldAlign_ = 4, 4
	structOfHostScalarTypes = append(structOfHostScalarTypes, copied)
	bits.data = unsafe.Pointer(copied)
	byteType := reflect.TypeOf(byte(0))
	inner := reflect.StructOf([]reflect.StructField{
		{Name: "NestedLayoutTag", Type: byteType},
		{Name: "NestedLayoutFloat", Type: scalar},
	})
	if inner.Size() != 12 || inner.Align() != 4 || inner.Field(1).Offset != 4 {
		t.Fatal("invalid nested Go layout control")
	}
	for _, nested := range []reflect.Type{inner, reflect.ArrayOf(2, inner)} {
		outer := reflect.StructOf([]reflect.StructField{
			{Name: "Marker", Type: reflect.TypeOf(structs.HostLayout{})},
			{Name: "NestedLayoutValue", Type: nested},
			{Name: "NestedLayoutTail", Type: byteType},
		})
		if outer.Field(1).Type != nested || outer.Field(2).Offset != nested.Size() || outer.Align() != 4 || outer.Size() != nested.Size()+4 {
			t.Fatalf("parent changed %s: size/align/tail=%d/%d/%d", nested, outer.Size(), outer.Align(), outer.Field(2).Offset)
		}
	}

	// Conversely, a nested marked record must not mark an ordinary container.
	marked := reflect.TypeOf(structOfHostWideInner{})
	outer := reflect.StructOf([]reflect.StructField{
		{Name: "MarkedLayoutValue", Type: marked},
		{Name: "MarkedLayoutTag", Type: byteType},
		{Name: "MarkedLayoutFloat", Type: scalar},
	})
	wantOffset := (marked.Size() + 1 + 3) &^ uintptr(3)
	if outer.Field(2).Offset != wantOffset || outer.Field(0).Type != marked {
		t.Fatal("nested marker propagated to its ordinary container")
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

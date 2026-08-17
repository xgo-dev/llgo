// LITTEST
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// Converting the two named values to interface{} must select their distinct
// runtime type descriptors; the test body then exercises pointer-to-this and
// element links from those descriptors.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: insertvalue %"{{.*}}eface" { ptr @_llgo_main.T,
// CHECK: insertvalue %"{{.*}}eface" { ptr @"_llgo_{{.*}}/runtime/abi.Type",
// CHECK: call ptr @"{{.*}}/runtime/abi.(*Type).StructType"
// CHECK: call ptr @"{{.*}}/runtime/abi.(*Type).Elem"

type T struct {
	p *T
	t *abi.Type
	n uintptr
	a []T
}

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

func main() {
	e := toEface(T{})
	e2 := toEface(abi.Type{})

	println(e.typ)
	println(e.typ.PtrToThis_)
	println(e2.typ)
	println(e2.typ.PtrToThis_)

	f0 := e.typ.StructType().Fields[0]
	if f0.Typ != e.typ.PtrToThis_ {
		panic("error field 0")
	}
	if f0.Typ.Elem() != e.typ {
		panic("error field 0 elem")
	}
	f1 := e.typ.StructType().Fields[1]
	if f1.Typ != e2.typ.PtrToThis_ {
		panic("error field 1")
	}
	if f1.Typ.Elem() != e2.typ {
		panic("error field 1 elem")
	}
	f2 := e.typ.StructType().Fields[2]
	if f2.Typ != e2.typ.StructType().Fields[0].Typ {
		panic("error field 2")
	}
	f3 := e.typ.StructType().Fields[3]
	if f3.Typ.Elem() != e.typ {
		panic("error field 3")
	}
}

func toEface(i any) *eface {
	// CHECK-LABEL: define ptr @main.toEface(%"{{.*}}eface" %{{[0-9]+}}){{.*}} {
	// CHECK: [[TO_EFACE_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
	// CHECK-NEXT: store %"{{.*}}eface" [[TO_EFACE_VALUE:%[0-9]+]], ptr [[TO_EFACE_ADDR]]
	// CHECK-NEXT: ret ptr [[TO_EFACE_ADDR]]
	return (*eface)(unsafe.Pointer(&i))
}

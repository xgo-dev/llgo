// LITTEST
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[EFACE_STORAGE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: [[RUNE_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.eface" [[RUNE_EFACE]], ptr [[EFACE_STORAGE]]
// CHECK: [[RUNE_TYPE_SLOT:%[0-9]+]] = getelementptr inbounds %main.eface, ptr [[EFACE_STORAGE]], i32 0, i32 0
// CHECK-NEXT: [[RUNE_TYPE:%[0-9]+]] = load ptr, ptr [[RUNE_TYPE_SLOT]]
// CHECK-NEXT: [[RUNE_NAME:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr [[RUNE_TYPE]])
// CHECK-NEXT: [[RUNE_MATCH:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" [[RUNE_NAME]], %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[RUNE_BAD:%[0-9]+]] = xor i1 [[RUNE_MATCH]], true
// CHECK-NEXT: br i1 [[RUNE_BAD]], label %{{.*}}, label %{{.*}}
// CHECK: [[BYTE_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.eface" [[BYTE_EFACE]], ptr [[EFACE_STORAGE]]
// CHECK: [[BYTE_TYPE_SLOT:%[0-9]+]] = getelementptr inbounds %main.eface, ptr [[EFACE_STORAGE]], i32 0, i32 0
// CHECK-NEXT: [[BYTE_TYPE:%[0-9]+]] = load ptr, ptr [[BYTE_TYPE_SLOT]]
// CHECK-NEXT: [[BYTE_NAME:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr [[BYTE_TYPE]])
// CHECK-NEXT: [[BYTE_MATCH:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" [[BYTE_NAME]], %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[BYTE_BAD:%[0-9]+]] = xor i1 [[BYTE_MATCH]], true
// CHECK-NEXT: br i1 [[BYTE_BAD]], label %{{.*}}, label %{{.*}}

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

func main() {
	var v any = rune(0)
	t := (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "int32" {
		panic("abi rune error")
	}
	v = byte(0)
	t = (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "uint8" {
		panic("abi byte error")
	}
}

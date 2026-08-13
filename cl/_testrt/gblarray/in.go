// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/llgo/runtime/abi"
)

// CHECK: @main.sizeBasicTypes = global [25 x i64] [i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 0, i64 16]

// CHECK-LABEL: define ptr @main.Basic(i64 %0){{.*}} {
// CHECK: [[BASIC_OOB:%[0-9]+]] = icmp uge i64 %0, 25
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 [[BASIC_OOB]], i64 %0, i1 false, i64 25)
// CHECK: [[BASIC_SLOT:%[0-9]+]] = getelementptr inbounds ptr, ptr @main.basicTypes, i64 %0
// CHECK-NEXT: [[BASIC_TYPE:%[0-9]+]] = load ptr, ptr [[BASIC_SLOT]]
// CHECK-NEXT: ret ptr [[BASIC_TYPE]]
func Basic(kind abi.Kind) *abi.Type {
	return basicTypes[kind]
}

// CHECK-LABEL: define ptr @main.basicType(i64 %0){{.*}} {
// CHECK: [[TYPE_OBJ:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 72)
// CHECK-NEXT: [[TYPE_SIZE_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr [[TYPE_OBJ]], i32 0, i32 0
// CHECK-NEXT: [[TYPE_OOB:%[0-9]+]] = icmp uge i64 %0, 25
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 [[TYPE_OOB]], i64 %0, i1 false, i64 25)
// CHECK: [[SIZE_SLOT:%[0-9]+]] = getelementptr inbounds i64, ptr @main.sizeBasicTypes, i64 %0
// CHECK-NEXT: [[TYPE_SIZE:%[0-9]+]] = load i64, ptr [[SIZE_SLOT]]
// CHECK-NEXT: [[TYPE_HASH_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr [[TYPE_OBJ]], i32 0, i32 2
// CHECK-NEXT: [[TYPE_HASH:%[0-9]+]] = trunc i64 %0 to i32
// CHECK-NEXT: [[TYPE_KIND_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr [[TYPE_OBJ]], i32 0, i32 6
// CHECK-NEXT: [[TYPE_KIND:%[0-9]+]] = trunc i64 %0 to i8
// CHECK-NEXT: store i64 [[TYPE_SIZE]], ptr [[TYPE_SIZE_FIELD]]
// CHECK-NEXT: store i32 [[TYPE_HASH]], ptr [[TYPE_HASH_FIELD]]
// CHECK-NEXT: store i8 [[TYPE_KIND]], ptr [[TYPE_KIND_FIELD]]
// CHECK: ret ptr [[TYPE_OBJ]]
func basicType(kind abi.Kind) *abi.Type {
	return &abi.Type{
		Size_: sizeBasicTypes[kind],
		Hash:  uint32(kind),
		Kind_: uint8(kind),
	}
}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: [[STRING_TYPE:%[0-9]+]] = call ptr @main.basicType(i64 24)
// CHECK-NEXT: store ptr [[STRING_TYPE]], ptr getelementptr inbounds (ptr, ptr @main.basicTypes, i64 24)
var (
	basicTypes = [...]*abi.Type{
		abi.String: basicType(abi.String),
	}
	sizeBasicTypes = [...]uintptr{
		abi.String: 16,
	}
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_TYPE:%[0-9]+]] = call ptr @main.Basic(i64 24)
// CHECK: [[MAIN_KIND_PTR:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr [[MAIN_TYPE]], i32 0, i32 6
// CHECK-NEXT: [[MAIN_KIND8:%[0-9]+]] = load i8, ptr [[MAIN_KIND_PTR]]
// CHECK-NEXT: [[MAIN_KIND:%[0-9]+]] = zext i8 [[MAIN_KIND8]] to i64
// CHECK: [[MAIN_SIZE_PTR:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr [[MAIN_TYPE]], i32 0, i32 0
// CHECK-NEXT: [[MAIN_SIZE:%[0-9]+]] = load i64, ptr [[MAIN_SIZE_PTR]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[MAIN_KIND]], i64 [[MAIN_SIZE]])
func main() {
	t := Basic(abi.String)
	c.Printf(c.Str("Kind: %d, Size: %d\n"), int(t.Kind_), t.Size_)
}

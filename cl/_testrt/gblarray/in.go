// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/gblarray.Basic"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp uge i64 %0, 25
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds ptr, ptr @"{{.*}}/cl/_testrt/gblarray.basicTypes", i64 %0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   ret ptr %3
// CHECK-NEXT: }
func Basic(kind abi.Kind) *abi.Type {
	return basicTypes[kind]
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/gblarray.basicType"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 72)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = icmp uge i64 %0, 25
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr @"{{.*}}/cl/_testrt/gblarray.sizeBasicTypes", i64 %0
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %1, i32 0, i32 2
// CHECK-NEXT:   %7 = trunc i64 %0 to i32
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %1, i32 0, i32 6
// CHECK-NEXT:   %9 = trunc i64 %0 to i8
// CHECK-NEXT:   store i64 %5, ptr %2, align 8
// CHECK-NEXT:   store i32 %7, ptr %6, align 4
// CHECK-NEXT:   store i8 %9, ptr %8, align 1
// CHECK-NEXT:   ret ptr %1
// CHECK-NEXT: }
func basicType(kind abi.Kind) *abi.Type {
	return &abi.Type{
		Size_: sizeBasicTypes[kind],
		Hash:  uint32(kind),
		Kind_: uint8(kind),
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/gblarray.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/gblarray.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/gblarray.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/abi.init"()
// CHECK-NEXT:   store i64 16, ptr getelementptr inbounds (i64, ptr @"{{.*}}/cl/_testrt/gblarray.sizeBasicTypes", i64 24), align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/cl/_testrt/gblarray.basicType"(i64 24)
// CHECK-NEXT:   store ptr %1, ptr getelementptr inbounds (ptr, ptr @"{{.*}}/cl/_testrt/gblarray.basicTypes", i64 24), align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
var (
	basicTypes = [...]*abi.Type{
		abi.String: basicType(abi.String),
	}
	sizeBasicTypes = [...]uintptr{
		abi.String: 16,
	}
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/gblarray.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testrt/gblarray.Basic"(i64 24)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 6
// CHECK-NEXT:   %2 = load i8, ptr %1, align 1
// CHECK-NEXT:   %3 = zext i8 %2 to i64
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   %6 = call i32 (ptr, ...) @printf(ptr @0, i64 %3, i64 %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	t := Basic(abi.String)
	c.Printf(c.Str("Kind: %d, Size: %d\n"), int(t.Kind_), t.Size_)
}

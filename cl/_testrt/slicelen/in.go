// LITTEST
package main

import (
	"unsafe"
)

// CHECK: {{^}}@0 = private unnamed_addr constant [30 x i8] c"unsafe.Slice: len out of range", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [46 x i8] c"unsafe.Slice: nil pointer with non-zero length", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [7 x i8] c"len > 0", align 1{{$}}

func main() {
	var s *int
	var lens uint32
	sl := unsafe.Slice(s, lens)
	slen := len(sl)
	println(slen)
	if slen > 0 {
		println("len > 0")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/slicelen.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/slicelen.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/slicelen.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/slicelen.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 30 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 46 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 30 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 30 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br i1 false, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

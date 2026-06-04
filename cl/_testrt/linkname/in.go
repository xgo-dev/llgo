// LITTEST
package main

import (
	_ "unsafe"

	"github.com/goplus/lib/c"
	_ "github.com/goplus/llgo/cl/_testrt/linkname/linktarget"
)

//go:linkname print github.com/goplus/llgo/cl/_testrt/linkname/linktarget.F
// CHECK-LINE: @0 = private unnamed_addr constant [2 x i8] c"a\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [2 x i8] c"b\00", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [2 x i8] c"c\00", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [2 x i8] c"d\00", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [2 x i8] c"1\00", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [2 x i8] c"2\00", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [2 x i8] c"3\00", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [2 x i8] c"4\00", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [5 x i8] c"hello", align 1

func print(a, b, c, d *c.Char)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/linkname.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/linkname.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/linkname.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/linkname/linktarget.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type m struct {
	s string
}

//go:linkname setInfo github.com/goplus/llgo/cl/_testrt/linkname/linktarget.(*m).setInfo
func setInfo(*m, string)

//go:linkname info github.com/goplus/llgo/cl/_testrt/linkname/linktarget.m.info
func info(m) string

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/linkname.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/linkname/linktarget.F"(ptr @0, ptr @1, ptr @2, ptr @3)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/linkname/linktarget.F"(ptr @4, ptr @5, ptr @6, ptr @7)
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/linkname/linktarget.(*m).setInfo"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 5 })
// CHECK-NEXT:   %1 = load %"{{.*}}/cl/_testrt/linkname.m", ptr %0, align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/linkname/linktarget.m.info"(%"{{.*}}/cl/_testrt/linkname.m" %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	print(c.Str("a"), c.Str("b"), c.Str("c"), c.Str("d"))
	print(c.Str("1"), c.Str("2"), c.Str("3"), c.Str("4"))
	var m m
	setInfo(&m, "hello")
	println(info(m))
}

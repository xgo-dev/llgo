// LITTEST
package main

import (
	_ "unsafe"

	"github.com/goplus/lib/c"
	_ "github.com/goplus/llgo/cl/_testrt/linkname/linktarget"
)

//go:linkname print github.com/goplus/llgo/cl/_testrt/linkname/linktarget.F
func print(a, b, c, d *c.Char)

type m struct {
	s string
}

//go:linkname setInfo github.com/goplus/llgo/cl/_testrt/linkname/linktarget.(*m).setInfo
func setInfo(*m, string)

//go:linkname info github.com/goplus/llgo/cl/_testrt/linkname/linktarget.m.info
func info(m) string

// CHECK: @[[A:[0-9]+]] = private unnamed_addr constant [2 x i8] c"a\00"
// CHECK: @[[B:[0-9]+]] = private unnamed_addr constant [2 x i8] c"b\00"
// CHECK: @[[C:[0-9]+]] = private unnamed_addr constant [2 x i8] c"c\00"
// CHECK: @[[D:[0-9]+]] = private unnamed_addr constant [2 x i8] c"d\00"
// CHECK: @[[ONE:[0-9]+]] = private unnamed_addr constant [2 x i8] c"1\00"
// CHECK: @[[TWO:[0-9]+]] = private unnamed_addr constant [2 x i8] c"2\00"
// CHECK: @[[THREE:[0-9]+]] = private unnamed_addr constant [2 x i8] c"3\00"
// CHECK: @[[FOUR:[0-9]+]] = private unnamed_addr constant [2 x i8] c"4\00"
// CHECK: @[[HELLO:[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"{{.*}}/cl/_testrt/linkname/linktarget.F"(ptr @[[A]], ptr @[[B]], ptr @[[C]], ptr @[[D]])
// CHECK-NEXT: call void @"{{.*}}/cl/_testrt/linkname/linktarget.F"(ptr @[[ONE]], ptr @[[TWO]], ptr @[[THREE]], ptr @[[FOUR]])
// CHECK: [[INFO_STORAGE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: call void @"{{.*}}/cl/_testrt/linkname/linktarget.(*m).setInfo"(ptr [[INFO_STORAGE]], %"{{.*}}/runtime/internal/runtime.String" { ptr @[[HELLO]], i64 5 })
// CHECK-NEXT: [[INFO_RECEIVER:%[0-9]+]] = load %main.m, ptr [[INFO_STORAGE]]
// CHECK-NEXT: [[INFO:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/linkname/linktarget.m.info"(%main.m [[INFO_RECEIVER]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" [[INFO]])
func main() {
	print(c.Str("a"), c.Str("b"), c.Str("c"), c.Str("d"))
	print(c.Str("1"), c.Str("2"), c.Str("3"), c.Str("4"))
	var m m
	setInfo(&m, "hello")
	println(info(m))
}

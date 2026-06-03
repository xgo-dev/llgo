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

// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

type Foo struct {
	A byte
	B uint8
	C uint16
	D byte
	E [8]int8
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/structsize.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @0, i64 14)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	c.Printf(c.Str("%d"), unsafe.Sizeof(Foo{}))
}

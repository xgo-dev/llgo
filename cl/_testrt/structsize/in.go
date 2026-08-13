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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 14)
func main() {
	c.Printf(c.Str("%d"), unsafe.Sizeof(Foo{}))
}

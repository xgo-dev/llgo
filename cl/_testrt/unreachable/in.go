// LITTEST
package main

// CHECK-LABEL: define void @main.foo(){{.*}} {
// CHECK: unreachable

import (
	"github.com/goplus/lib/c"
)

func foo() {
	c.Unreachable()
}

func main() {
	foo()
	c.Printf(c.Str("Hello\n"))
}

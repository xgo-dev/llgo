// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testlto/globaldce_unexported_method_identity/base"
	"github.com/xgo-dev/llgo/cl/_testlto/globaldce_unexported_method_identity/other"
)

// CHECK-DAG: !"go.method.github.com/xgo-dev/llgo/cl/_testlto/globaldce_unexported_method_identity/base.hidden:func() int"
// CHECK-DAG: !"go.method.github.com/xgo-dev/llgo/cl/_testlto/globaldce_unexported_method_identity/other.hidden:func() int"
// CHECK-DAG: !"go.method.github.com/xgo-dev/llgo/cl/_testlto/globaldce_unexported_method_identity.hidden:func() int"
// SYMBOL-NOT: globaldce_unexported_method_identity/other{{.*}}Other{{.*}}hidden
// SYMBOL-NOT: main{{.*}}Local{{.*}}hidden
// SYMBOL-DAG: globaldce_unexported_method_identity/base{{.*}}Exported{{.*}}hidden
// SYMBOL-NOT: globaldce_unexported_method_identity/other{{.*}}Other{{.*}}hidden
// SYMBOL-NOT: main{{.*}}Local{{.*}}hidden

type Combined struct {
	base.Exported
	other.Other
}

type Local struct{}

//go:noinline
func (Local) hidden() int {
	panic("Local.hidden should be unreachable")
}

func keepAbi(v any) {
	if v == nil {
		panic("nil")
	}
}

func main() {
	keepAbi(Local{})
	println(base.Call(Combined{}))
}

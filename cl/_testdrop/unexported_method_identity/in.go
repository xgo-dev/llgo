// LITTEST
package main

import "github.com/xgo-dev/llgo/cl/_testdrop/unexported_method_identity/api"

// SYMBOL-NOT: testdrop/unexported_method_identity{{.*}}Local{{.*}}hidden
// SYMBOL-DAG: testdrop/unexported_method_identity/api{{.*}}Good{{.*}}hidden
// SYMBOL-NOT: testdrop/unexported_method_identity{{.*}}Local{{.*}}hidden

var sink any

// Local has a same-spelled unexported hidden method and its type metadata is
// reachable through any. The api.hiddenIface demand must not keep Local.hidden
// alive because unexported method identity includes the defining package.
type Local struct {
	n int
}

//go:noinline
func (l Local) hidden() int {
	panic("Local.hidden should be unreachable")
}

func main() {
	sink = Local{n: 1}
	println(api.Use(api.NewGood(41)))
}

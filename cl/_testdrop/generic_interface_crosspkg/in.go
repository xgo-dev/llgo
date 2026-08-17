// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/generic_interface_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/generic_interface_crosspkg/model"
)

// SYMBOL-NOT: testdrop/generic_interface_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_crosspkg/model{{.*}}Box{{.*}}string{{.*}}Value
// SYMBOL-DAG: testdrop/generic_interface_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Value
// SYMBOL-NOT: testdrop/generic_interface_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_crosspkg/model{{.*}}Box{{.*}}string{{.*}}Value

var sink any

func main() {
	// api.UseInt demands I[int].Value, so the instantiated interface method
	// signature is Value() int, not Value() T.
	n := api.UseInt(model.NewIntBox(40))

	// Box[string] reaches interface-domain metadata through any, but the only
	// reachable generic interface demand is I[int].Value. Box[string].Value has
	// the same method name and a different instantiated signature, so it must
	// not be retained by name alone.
	text := model.NewStringBox("go")
	sink = text

	println(n + model.UseStringBox(text))
}

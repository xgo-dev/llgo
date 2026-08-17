// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/generic_interface_func_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/generic_interface_func_crosspkg/model"
)

// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}uint{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}string{{.*}}Value
// SYMBOL-DAG: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Value
// SYMBOL-DAG: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}uint{{.*}}Value
// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}int{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}uint{{.*}}Drop
// SYMBOL-NOT: testdrop/generic_interface_func_crosspkg/model{{.*}}Box{{.*}}string{{.*}}Value

var sink any

func main() {
	// api.Use is generic, but this call instantiates it as Use[int]. The
	// interface method demand inside Use is therefore I[int].Value, whose ABI
	// method signature is Value() int.
	n := api.Use[int](model.NewIntBox(40))

	// This call relies on type argument inference. The compiler still
	// instantiates api.Use as Use[uint], so it should produce an independent
	// I[uint].Value demand.
	u := api.Use(model.NewUintBox(1))

	// Keep Box[string] metadata reachable without creating an I[string].Value
	// demand. This guards the instantiated generic path against matching only
	// by the method name Value.
	text := model.NewStringBox("go")
	sink = text

	println(n + int(u) + model.UseStringBox(text) - 1)
}

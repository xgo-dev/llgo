// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/iface_flow_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/iface_flow_crosspkg/factory"
)

// SYMBOL-NOT: testdrop/iface_flow_crosspkg/factory{{.*}}T{{.*}}Drop
// SYMBOL-DAG: testdrop/iface_flow_crosspkg/factory{{.*}}T{{.*}}Keep
// SYMBOL-NOT: testdrop/iface_flow_crosspkg/factory{{.*}}T{{.*}}Drop

func main() {
	// factory.Make performs the concrete-to-interface conversion in another
	// package. api.Use performs the interface method call in a third package.
	// The global analysis must merge both facts before preserving T.Keep.
	println(api.Use(factory.Make(41)))
}

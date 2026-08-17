// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/interface_demand_fixedpoint/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/interface_demand_fixedpoint/flow"
	"github.com/xgo-dev/llgo/cl/_testdrop/interface_demand_fixedpoint/model"
)

// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/model{{.*}}Runner{{.*}}Drop
// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/flow{{.*}}Worker{{.*}}Drop
// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/flow{{.*}}Finisher{{.*}}Drop
// SYMBOL-DAG: testdrop/interface_demand_fixedpoint/model{{.*}}Runner{{.*}}Run
// SYMBOL-DAG: testdrop/interface_demand_fixedpoint/flow{{.*}}Step
// SYMBOL-DAG: testdrop/interface_demand_fixedpoint/flow{{.*}}Worker{{.*}}Next
// SYMBOL-DAG: testdrop/interface_demand_fixedpoint/flow{{.*}}Finisher{{.*}}Done
// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/model{{.*}}Runner{{.*}}Drop
// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/flow{{.*}}Worker{{.*}}Drop
// SYMBOL-NOT: testdrop/interface_demand_fixedpoint/flow{{.*}}Finisher{{.*}}Drop

var sink any

func main() {
	// Keep a second Worker type descriptor reachable without directly creating
	// a Second.Next demand from main. The actual Worker.Next demand is produced
	// only after Runner.Run is kept and reaches flow.Step.
	sink = flow.Worker{N: 0}
	println(api.UseFirst(model.NewRunner()))
}

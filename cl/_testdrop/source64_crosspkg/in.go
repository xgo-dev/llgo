// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdrop/source64_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/source64_crosspkg/model"
)

// SYMBOL-NOT: testdrop/source64_crosspkg/model{{.*}}RuntimeSource{{.*}}Drop
// SYMBOL-NOT: testdrop/source64_crosspkg/model{{.*}}Uint64Only{{.*}}Uint64
// SYMBOL-DAG: testdrop/source64_crosspkg/model{{.*}}RuntimeSource{{.*}}Uint64
// SYMBOL-NOT: testdrop/source64_crosspkg/model{{.*}}RuntimeSource{{.*}}Drop
// SYMBOL-NOT: testdrop/source64_crosspkg/model{{.*}}Uint64Only{{.*}}Uint64

var sink any

func main() {
	// This mirrors math/rand's Source/Source64 shape without importing rand.
	// RuntimeSource enters the interface domain as api.Source, then api.New
	// discovers the wider api.Source64 interface and Rand.Uint64 performs the
	// dynamic Source64.Uint64 call.
	r := api.New(model.NewRuntimeSource(41))

	// Uint64Only reaches type metadata through any and has the same Uint64
	// method name, but it does not fully implement api.Source64. The Source64
	// demand must not keep this method alive by name alone.
	only := model.NewUint64Only(100)
	sink = only

	println(r.Uint64() + model.UseUint64Only(only) - 100)
}

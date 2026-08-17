// LITTEST
package main

import (
	"reflect"

	"github.com/xgo-dev/llgo/cl/_testdrop/reflect_dynamic_iface_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testdrop/reflect_dynamic_iface_crosspkg/model"
)

// SYMBOL-NOT: testdrop/reflect_dynamic_iface_crosspkg/model{{.*}}Unused{{.*}}ReflectKeep
// SYMBOL-DAG: testdrop/reflect_dynamic_iface_crosspkg/model{{.*}}Used{{.*}}ReflectKeep
// SYMBOL-NOT: testdrop/reflect_dynamic_iface_crosspkg/model{{.*}}Unused{{.*}}ReflectKeep

//go:noinline
func dynamicName(name string) string {
	return name
}

func main() {
	// api.Accept makes model.Used enter a reachable non-empty interface without
	// calling ReflectKeep through that interface. The dynamic MethodByName below
	// must conservatively keep exported methods for interface-domain types.
	api.Accept(model.NewUsed(0))

	// model.Unused also implements api.Reflector, but it is only used as a
	// concrete value. It must not be retained just because a reachable interface
	// has the same exported method.
	unused := model.UseUnused(model.NewUnused(0))

	out := reflect.ValueOf(model.NewUsed(41)).MethodByName(dynamicName("ReflectKeep")).Call(nil)
	println(out[0].Int() + int64(unused))
}

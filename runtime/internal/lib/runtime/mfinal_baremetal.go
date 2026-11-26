//go:build baremetal && !nogc && !llgo.baremetal.stdio

package runtime

// The tinygogc finalizer bridge is only wired for WebAssembly. Baremetal
// builds still provide the entry point so finalizer calls type-check and
// link; registration is a no-op until that bridge is ported.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}

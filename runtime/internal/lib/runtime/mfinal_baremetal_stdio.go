//go:build baremetal && !nogc && llgo.baremetal.stdio

package runtime

// SetFinalizer has no dispatch on bare metal. Targets that patch the
// standard library (llgo.baremetal.stdio) fail loudly instead of silently
// dropping a registered finalizer.
func SetFinalizer(_ any, finalizer any) {
	if finalizer != nil {
		panic("runtime.SetFinalizer: no finalizer dispatch on bare metal")
	}
}

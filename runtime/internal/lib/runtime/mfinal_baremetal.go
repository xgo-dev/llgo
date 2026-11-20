//go:build baremetal && !nogc && llgo.baremetal.stdio

package runtime

func SetFinalizer(_ any, finalizer any) {
	if finalizer != nil {
		panic("runtime.SetFinalizer: no finalizer dispatch on bare metal")
	}
}

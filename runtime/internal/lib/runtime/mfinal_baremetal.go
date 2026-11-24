//go:build baremetal && !nogc && llgo.baremetal.stdio

package runtime

import "unsafe"

func SetFinalizer(_ any, finalizer any) {
	if finalizer != nil {
		panic("runtime.SetFinalizer: no finalizer dispatch on bare metal")
	}
}

func llgoSetFinalizerPointer(_ unsafe.Pointer, finalizer unsafe.Pointer, _ uintptr) {
	if finalizer != nil {
		panic("runtime.SetFinalizer: no finalizer dispatch on bare metal")
	}
}

//go:build llgo && windows && !nogc && !baremetal && !llgo_noffi

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/ffi"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func makeFuncCallback(nout int) func(*ffi.Signature, unsafe.Pointer, *unsafe.Pointer, unsafe.Pointer) {
	switch nout {
	case 0:
		return bind0ForeignThread
	case 1:
		return bind1ForeignThread
	default:
		return bindnForeignThread
	}
}

func bind0ForeignThread(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	registered := runtime.EnterForeignThread()
	bind0(cif, ret, args, userdata)
	// Retained registrations belong to the thread's G lifecycle, including
	// runtime.Goexit. Keep cleanup off the defer chain: a defer here would try
	// to unwind across the libffi callback frame, which is not a Go frame.
	runtime.ExitForeignThread(registered)
}

func bind1ForeignThread(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	registered := runtime.EnterForeignThread()
	bind1(cif, ret, args, userdata)
	runtime.ExitForeignThread(registered)
}

func bindnForeignThread(cif *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	registered := runtime.EnterForeignThread()
	bindn(cif, ret, args, userdata)
	runtime.ExitForeignThread(registered)
}

//go:build !windows && !wasm && !nogc && !baremetal

package thread

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
)

//go:linkname create C.GC_pthread_create
func create(thread *nativeThread, attr *nativeAttr, routine RoutineFunc, arg c.Pointer) c.Int

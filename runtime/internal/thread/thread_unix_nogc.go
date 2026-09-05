//go:build !windows && (nogc || baremetal || wasm)

package thread

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname create C.pthread_create
func create(thread *nativeThread, attr *nativeAttr, routine RoutineFunc, arg c.Pointer) c.Int

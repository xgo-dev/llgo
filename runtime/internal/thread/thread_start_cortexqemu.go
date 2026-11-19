//go:build baremetal && llgo.singlecontext

package thread

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func CreateDetached(_ uintptr, _ RoutineFunc, _ c.Pointer) c.Int {
	return 38 // ENOSYS: the board has no native threads.
}

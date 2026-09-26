//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/thread"
)

func EnableForeignThreadRegistration() {}

func gLifecycleDestructor(_ thread.KeyDestructor) thread.KeyDestructor {
	return destroyWasiForeignG
}

func destroyWasiForeignG(ptr c.Pointer) {
	destroyG(ptr)
	unregisterWasiGCThread()
}

func EnterForeignThread() bool {
	registerWasiGCThread()
	getg()
	return false
}

func ExitForeignThread(bool) {}

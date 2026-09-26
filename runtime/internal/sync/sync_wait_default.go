//go:build !windows && (!llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear)

package sync

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func condWait(cond *Cond, mutex *Mutex) c.Int {
	return pthreadCondWait(cond, mutex)
}

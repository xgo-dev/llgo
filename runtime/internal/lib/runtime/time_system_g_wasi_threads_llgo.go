//go:build llgo && wasip1 && wasm && llgo.wasi_threads

package runtime

import _ "unsafe"

//go:linkname markTimerSystemG github.com/xgo-dev/llgo/runtime/internal/runtime.MarkTimerSystemGoroutine
func markTimerSystemG()

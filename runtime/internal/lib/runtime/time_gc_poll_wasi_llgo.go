//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package runtime

import llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"

const timerGCWaitQuantum = int64(20 * 1e6)

func timerGCSafepoint() { llruntime.CooperativeSafepoint() }

package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname AsyncInitNoop C.llgo_uv_async_init_noop
func AsyncInitNoop(loop *Loop, async *Async) c.Int

//go:linkname AsyncInitRuntime C.llgo_uv_async_init_runtime
func AsyncInitRuntime(loop *Loop, async *Async) c.Int

//go:linkname TimerStartRuntime C.llgo_uv_timer_start_runtime
func TimerStartRuntime(timer *Timer, timeoutMs uint64, repeat uint64) c.Int

//go:linkname SignalStartRuntime C.llgo_uv_signal_start_runtime
func SignalStartRuntime(handle *Signal, signum c.Int) c.Int

//go:linkname SignalStartOneshotRuntime C.llgo_uv_signal_start_oneshot_runtime
func SignalStartOneshotRuntime(handle *Signal, signum c.Int) c.Int

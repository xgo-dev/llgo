package runtime

import llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"

// LockOSThread wires the calling goroutine to its current OS thread.
// Hosted LLGo backends already keep each running goroutine on one OS thread;
// recording the lock count ensures a goroutine that exits while still locked
// terminates its OS thread rather than returning it to the worker pool.
func LockOSThread() {
	llruntime.LockOSThread()
}

func UnlockOSThread() {
	llruntime.UnlockOSThread()
}

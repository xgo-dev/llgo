//go:build baremetal && !nogc

package runtime

import "unsafe"

// Scheduler-free bare-metal targets retain the pre-existing no-op lifecycle
// contract. They cannot run user cleanup callbacks asynchronously.
func AddCleanupPtr(ptr unsafe.Pointer, cleanup func()) (cancel func()) {
	return func() {}
}

func AddCancelableCleanupPtr(ptr unsafe.Pointer, cleanup func()) uint64 {
	return 0
}

func StopCleanupPtr(id uint64) {}

//go:build llgo && wasip1 && wasm && llgo.wasi_threads

package runtime

func pthreadStackSize(size uintptr) uintptr {
	if size != 0 {
		return size
	}
	// wasi-libc derives its default from the 10 MiB main stack. A 1 MiB
	// worker stack keeps the current one-G-per-pthread backend within WAMR's
	// bounded shared memory while preserving -pthread-stack-size overrides.
	return 1 << 20
}

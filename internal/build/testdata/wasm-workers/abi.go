package main

import _ "unsafe"

const LLGoFiles = "workers.c"

//go:linkname parallelWorkerBarrier C.llgo_test_parallel_worker_barrier
func parallelWorkerBarrier() int32

//go:linkname parallelWorkerThread C.llgo_test_parallel_worker_thread
func parallelWorkerThread(slot int32) uintptr

//go:linkname currentWorkerThread C.llgo_test_current_worker_thread
func currentWorkerThread() uintptr

//go:linkname configuredWorkerCount C.llgo_test_worker_count
func configuredWorkerCount() int32

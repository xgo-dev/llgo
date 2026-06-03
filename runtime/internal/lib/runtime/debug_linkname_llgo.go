package runtime

import (
	_ "unsafe"
)

var (
	llgoMaxStack   int
	llgoGCPercent  int32 = 100
	llgoMemLimit   int64
	llgoMaxThreads int
)

//go:linkname setTraceback runtime/debug.SetTraceback
func setTraceback(level string) {}

//go:linkname readGCStats runtime/debug.readGCStats
func readGCStats(_ *[]uint64) {}

//go:linkname freeOSMemory runtime/debug.freeOSMemory
func freeOSMemory() {}

//go:linkname setMaxStack runtime/debug.setMaxStack
func setMaxStack(in int) (out int) {
	out = llgoMaxStack
	llgoMaxStack = in
	return out
}

//go:linkname setGCPercent runtime/debug.setGCPercent
func setGCPercent(in int32) (out int32) {
	out = llgoGCPercent
	llgoGCPercent = in
	return out
}

//go:linkname setPanicOnFault runtime/debug.setPanicOnFault
func setPanicOnFault(_ bool) (old bool) { return false }

//go:linkname setMaxThreads runtime/debug.setMaxThreads
func setMaxThreads(in int) (out int) {
	out = llgoMaxThreads
	llgoMaxThreads = in
	return out
}

//go:linkname setMemoryLimit runtime/debug.setMemoryLimit
func setMemoryLimit(in int64) (out int64) {
	out = llgoMemLimit
	llgoMemLimit = in
	return out
}

//go:linkname runtime_debug_WriteHeapDump runtime/debug.WriteHeapDump
func runtime_debug_WriteHeapDump(uintptr) {}

//go:linkname debug_modinfo runtime/debug.modinfo
func debug_modinfo() string { return "" }

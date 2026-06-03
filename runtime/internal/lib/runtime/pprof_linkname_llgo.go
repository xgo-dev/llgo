//go:build darwin || linux

package runtime

import "unsafe"

var pprofLabel unsafe.Pointer

//go:linkname runtime_setProfLabel runtime/pprof.runtime_setProfLabel
func runtime_setProfLabel(labels unsafe.Pointer) {
	pprofLabel = labels
}

//go:linkname runtime_getProfLabel runtime/pprof.runtime_getProfLabel
func runtime_getProfLabel() unsafe.Pointer {
	return pprofLabel
}

//go:linkname runtime_FrameStartLine runtime/pprof.runtime_FrameStartLine
func runtime_FrameStartLine(f *Frame) int {
	if f == nil {
		return 0
	}
	return f.startLine
}

//go:linkname runtime_FrameSymbolName runtime/pprof.runtime_FrameSymbolName
func runtime_FrameSymbolName(f *Frame) string {
	if f == nil {
		return ""
	}
	return f.Function
}

//go:linkname runtime_expandFinalInlineFrame runtime/pprof.runtime_expandFinalInlineFrame
func runtime_expandFinalInlineFrame(stk []uintptr) []uintptr {
	return stk
}

//go:linkname pprof_cyclesPerSecond runtime/pprof.runtime_cyclesPerSecond
func pprof_cyclesPerSecond() int64 {
	return 1
}

var (
	cpuProfilePeriodRecord = []uint64{3, 0, 100}   // [len, timestamp, hz]
	cpuProfilePeriodTags   = []unsafe.Pointer{nil} // one tag slot for the period record
	cpuProfileEOF          = true                  // minimal stub: no streaming samples
)

//go:linkname runtime_pprof_readProfile runtime/pprof.readProfile
func runtime_pprof_readProfile() (data []uint64, tags []unsafe.Pointer, eof bool) {
	// Provide a minimal, valid profile stream for runtime/pprof.
	// The stdlib expects at least the initial "period" record (3 uint64s).
	return cpuProfilePeriodRecord, cpuProfilePeriodTags, cpuProfileEOF
}

//go:linkname pprof_goroutineProfileWithLabels runtime.pprof_goroutineProfileWithLabels
func pprof_goroutineProfileWithLabels(p []StackRecord, labels []unsafe.Pointer) (n int, ok bool) {
	return 0, true
}

//go:linkname runtime_goroutineLeakGC runtime/pprof.runtime_goroutineLeakGC
func runtime_goroutineLeakGC() {}

//go:linkname runtime_goroutineleakcount runtime/pprof.runtime_goroutineleakcount
func runtime_goroutineleakcount() int {
	return 0
}

//go:linkname pprof_goroutineLeakProfileWithLabels runtime.pprof_goroutineLeakProfileWithLabels
func pprof_goroutineLeakProfileWithLabels(p []StackRecord, labels []unsafe.Pointer) (n int, ok bool) {
	return 0, true
}

//go:linkname pprof_memProfileInternal runtime.pprof_memProfileInternal
func pprof_memProfileInternal(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	return 0, true
}

//go:linkname pprof_blockProfileInternal runtime.pprof_blockProfileInternal
func pprof_blockProfileInternal(p []BlockProfileRecord) (n int, ok bool) {
	return 0, true
}

//go:linkname pprof_mutexProfileInternal runtime.pprof_mutexProfileInternal
func pprof_mutexProfileInternal(p []BlockProfileRecord) (n int, ok bool) {
	return 0, true
}

//go:linkname pprof_threadCreateInternal runtime.pprof_threadCreateInternal
func pprof_threadCreateInternal(p []StackRecord) (n int, ok bool) {
	return 0, true
}

//go:linkname pprof_fpunwindExpand runtime.pprof_fpunwindExpand
func pprof_fpunwindExpand(dst, src []uintptr) int {
	return copy(dst, src)
}

//go:linkname pprof_makeProfStack runtime.pprof_makeProfStack
func pprof_makeProfStack() []uintptr {
	return make([]uintptr, 64)
}

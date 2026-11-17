//go:build nogc || baremetal

package runtime

import "unsafe"

func heapProfileG() *g { return nil }

func recordHeapProfileAlloc(*g, unsafe.Pointer, uintptr) {}

func recordHeapProfileWithoutG(unsafe.Pointer, uintptr) {}

func readHeapProfile(p []MemProfileRecord, inuseZero bool) (int, bool) {
	return memProfileClassRecords(p, inuseZero)
}

func PauseHeapProfile() unsafe.Pointer { return nil }
func ResumeHeapProfile(unsafe.Pointer) {}

func releaseHeapProfileG(gp *g) {}

func heapProfileFrameForPC(pc uintptr) (CallerFrame, bool) { return CallerFrame{}, false }

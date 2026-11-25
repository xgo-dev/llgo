//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (C glue, the llgo
// runtime's stream handles). Without definitions the hosted link fails on the
// unresolved relocations. Push linkname supplies same-named symbols that exist
// only on this side of the build split, so the demo builds hosted with inert
// stubs; the real implementations come back on the llgo.idf build, where this
// file is excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubHeapPrint C.llgo_demo_heap
func stubHeapPrint(tag *byte, n int32) { _, _ = tag, n }

//go:linkname stubVTaskDelay C.vTaskDelay
func stubVTaskDelay(ticks uint32) { _ = ticks }

//go:linkname stubLpProgPtr C.llgo_demo_lp_prog_ptr
func stubLpProgPtr() *byte { return nil }

//go:linkname stubLpProgSize C.llgo_demo_lp_prog_size
func stubLpProgSize() int32 { return 0 }

//go:linkname stubLpMarkers C.llgo_demo_lp_markers
func stubLpMarkers(out *uint32) int32 { _ = out; return 0 }

//go:linkname stubLpRamRead C.llgo_demo_lp_ram_read
func stubLpRamRead(wordOff int32, out *uint32, words int32) int32 {
	_, _, _ = wordOff, out, words
	return 0
}

//go:linkname stubWakeupCause C.llgo_demo_lp_wakeup_cause
func stubWakeupCause(out *byte, max int32) int32 { _, _ = out, max; return 0 }

//go:linkname stubSleepTimer C.llgo_demo_sleep_timer
func stubSleepTimer(usLo int32, usHi int32) int32 { _, _ = usLo, usHi; return 0 }

//go:linkname stubDeepSleep C.llgo_demo_deep_sleep
func stubDeepSleep() {}

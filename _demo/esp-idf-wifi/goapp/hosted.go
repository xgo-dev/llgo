//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname
// externs whose definitions live in the ESP-IDF link environment (C glue,
// FreeRTOS, heap_caps) or in the llgo runtime tree (tinygogc, which cannot
// be imported from this module - runtime/internal is internal). Without
// definitions the hosted link fails on the unresolved relocations. Push
// linkname supplies same-named symbols that exist only on this side of the
// build split, so the demo builds hosted with inert stubs; the real
// implementations come back on the llgo.idf build, where this file is
// excluded.
package main

import "unsafe"

//go:linkname stubXTaskGetTickCount C.xTaskGetTickCount
func stubXTaskGetTickCount() uint32 { return 0 }

//go:linkname stubVTaskDelay C.vTaskDelay
func stubVTaskDelay(ticks uint32) { _ = ticks }

//go:linkname stubRdcycle C.llgo_demo_rdcycle
func stubRdcycle() uint32 { return 0 }

//go:linkname stubDemoTLSCA C.llgo_demo_tls_ca
func stubDemoTLSCA(buf *byte, cap int32, outLen *int32) {
	_, _, _ = buf, cap, outLen
}

//go:linkname stubDemoTLSCAWrong C.llgo_demo_tls_ca_wrong
func stubDemoTLSCAWrong(buf *byte, cap int32, outLen *int32) {
	_, _, _ = buf, cap, outLen
}

//go:linkname stubHeapCapsMalloc C.heap_caps_malloc
func stubHeapCapsMalloc(size, caps uint32) unsafe.Pointer {
	_, _ = size, caps
	return nil
}

//go:linkname stubHeapCapsFree C.heap_caps_free
func stubHeapCapsFree(ptr unsafe.Pointer) { _ = ptr }

// The gcExtBytes/gcExtHeapCap vars need no stubs: their pull linkname in
// main.go renames the Go var to the C symbol, which defines it for the
// hosted link as well (they simply read zero on a host).

//go:linkname stubHeapStats github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.HeapStats
func stubHeapStats() (usable, reserved uintptr, segments int, external uintptr) {
	return 0, 0, 0, 0
}

//go:linkname stubGcCollect github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.GC
func stubGcCollect() {}

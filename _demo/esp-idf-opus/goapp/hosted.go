//go:build !llgo.idf

// Host build (go build / go vet): main.go declares a pull //go:linkname
// extern (llgo_opus_heap) whose definition lives in the ESP-IDF link
// environment (main/demo_glue.c). Push linkname supplies a same-named inert
// stub on this side of the build split, so the demo builds hosted; the real
// implementation comes back on the llgo.idf build, where this file is
// excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubHeapLedger C.llgo_opus_heap
func stubHeapLedger(tag *byte, n int32) { _, _ = tag, n }

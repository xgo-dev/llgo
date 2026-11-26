//go:build !llgo.idf

// Host build (go build / go vet): linkname.go declares pull //go:linkname
// externs (llgo_opus_heap, llgoFixStreams) whose definitions live in the
// ESP-IDF link environment (main/demo_glue.c, host/llgo_syms.S). Push
// linkname supplies same-named inert stubs on this side of the build split,
// so the demo builds hosted; the real implementations come back on the
// llgo.idf build, where this file is excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubHeapLedger C.llgo_opus_heap
func stubHeapLedger(tag *byte, n int32) { _, _ = tag, n }

//go:linkname stubFixStreams C.llgoFixStreams
func stubFixStreams() {}

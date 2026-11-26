//go:build !llgo.idf

// Host build (go build / go vet): linkname.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (esp_restart in the
// IDF's esp_runtime component, the heap ledger in main/demo_glue.c). Without
// definitions the hosted link fails on the unresolved relocations. Push
// linkname supplies same-named inert stubs on this side of the build split,
// so the demo builds hosted; the real implementations come back on the
// llgo.idf build, where this file is excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubRestart C.esp_restart
func stubRestart() {}

//go:linkname stubHeapPrint C.llgo_demo_heap
func stubHeapPrint(tag *byte, n int32) { _, _ = tag, n }

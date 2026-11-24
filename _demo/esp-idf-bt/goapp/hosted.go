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

//go:linkname stubFixStreams C.llgoFixStreams
func stubFixStreams() {}

//go:linkname stubHeapPrint C.llgo_demo_heap
func stubHeapPrint(tag *byte, n int32) { _, _ = tag, n }

// fixStreams and heapPrint keep their names on both sides; the stubs above
// claim the C symbols on the host, so main.go's declarations resolve either
// way. The llgo.idf build resolves the same names against llgo_syms.S and the
// demo glue instead.

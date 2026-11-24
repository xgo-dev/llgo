//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (the demo glue in
// host/llgo_syms.S and FreeRTOS). Without definitions the hosted link fails on
// the unresolved relocations. Push linkname supplies same-named symbols that
// exist only on this side of the build split, so the demo builds hosted with
// inert stubs; the real implementations come back on the llgo.idf build, where
// this file is excluded.
package main

import _ "unsafe" // for the push //go:linkname definitions below

//go:linkname stubFixStreams C.llgoFixStreams
func stubFixStreams() {}

//go:linkname stubVTaskDelay C.vTaskDelay
func stubVTaskDelay(ticks uint32) { _ = ticks }

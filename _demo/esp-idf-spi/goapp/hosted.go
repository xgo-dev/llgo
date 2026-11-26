//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (C glue, FreeRTOS).
// Push linkname supplies same-named symbols that exist only on this side of
// the build split, so the demo builds hosted with inert stubs; the real
// implementations come back on the llgo.idf build, where this file is
// excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubVTaskDelay C.vTaskDelay
func stubVTaskDelay(ticks uint32) { _ = ticks }

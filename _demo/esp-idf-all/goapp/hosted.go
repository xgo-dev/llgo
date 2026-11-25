//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (host/llgo_syms.S,
// the net glue and this project's ble_glue.c). Push linkname supplies
// same-named inert stubs on this side of the build split, so the package
// builds hosted; the real implementations come back on the llgo.idf build,
// where this file is excluded.
package main

import _ "unsafe" // for the push //go:linkname definitions below

//go:linkname stubNowMs C.llgo_net_now_ms
func stubNowMs() uint32 { return 0 }

//go:linkname stubDmaMax C.llgo_all_dma_max
func stubDmaMax() int32 { return 0 }

//go:linkname stubSecInitiate C.llgo_ble_security
func stubSecInitiate(conn uint32) int32 { _ = conn; return 0 }

//go:linkname stubConnSec C.llgo_ble_conn_sec
func stubConnSec(conn uint32) int32 { _ = conn; return -1 }

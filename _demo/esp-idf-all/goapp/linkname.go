// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号，llgo.idf 构建
// 时由 ESP-IDF 链接环境解析。标准 Go 要求 //go:linkname 只能出现在 import 了
// "unsafe" 的文件里，故本文件显式 import；宿主构建时这些符号由 hosted.go 的
// 同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

// nowMs is the millisecond clock of the net glue (esp_timer domain), used
// for the AP station-age checks.
//
//go:linkname nowMs C.llgo_net_now_ms
func nowMs() uint32

// The pairing passthroughs this firmware's glue copy (idfproj/main/ble_glue.c)
// adds on top of the pkg/esp/ble template: secInitiate starts the SMP
// procedure on a live connection (asynchronous; the pairing result prints on
// the C side), connSec reports the link's security state as bits
// (bit 0 encrypted, bit 1 authenticated, bit 2 bonded; -1 = not a live
// connection).
//
//go:linkname secInitiate C.llgo_ble_security
func secInitiate(conn uint32) int32

//go:linkname connSec C.llgo_ble_conn_sec
func connSec(conn uint32) int32

// dmaMax is the largest free contiguous DMA-capable internal block
// (app_main.c helper): the WiFi driver needs ~1.6 KB of contiguous DMA
// memory per packet, so this is the ledger line that actually predicts
// whether the radio keeps working.
//
//go:linkname dmaMax C.llgo_all_dma_max
func dmaMax() int32

//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/bt_glue_bluedroid.c — the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design (host/make-artifact.sh lists them); the IDF component archive
// provides the definitions at the final link.
package bt

import _ "unsafe" // for go:linkname

//go:linkname glueABI C.llgo_bt_glue_abi
func glueABI() int32

//go:linkname initBT C.llgo_bt_init
func initBT() int32

//go:linkname deinitBT C.llgo_bt_deinit
func deinitBT() int32

//go:linkname readyBT C.llgo_bt_ready
func readyBT() int32

//go:linkname backendName C.llgo_bt_backend_name
func backendName(buf *uint8, n int32) int32

//go:linkname setName C.llgo_bt_set_name
func setName(name *uint8, n int32) int32

//go:linkname setScanMode C.llgo_bt_set_scan_mode
func setScanMode(discoverable int32) int32

// ---- SPP ----

//go:linkname sppInit C.llgo_bt_spp_init
func sppInit() int32

//go:linkname sppStartSrv C.llgo_bt_spp_start_srv
func sppStartSrv(scn int32, name *uint8, nameLen int32) int32

//go:linkname sppDisconnect C.llgo_bt_spp_disconnect
func sppDisconnect(handle uint32) int32

//go:linkname sppWrite C.llgo_bt_spp_write
func sppWrite(handle uint32, data *uint8, n int32) int32

//go:linkname sppRead C.llgo_bt_spp_read
func sppRead(handle uint32, dst *uint8, cap int32) int32

// ---- A2DP sink (audio profile) ----

//go:linkname a2dpEnable C.llgo_bt_a2dp_enable
func a2dpEnable(bclk, ws, dout, dmaDesc, dmaFrames int32) int32

//go:linkname a2dpDisable C.llgo_bt_a2dp_disable
func a2dpDisable() int32

//go:linkname a2dpVolumeSet C.llgo_bt_a2dp_volume_set
func a2dpVolumeSet(volume int32) int32

//go:linkname a2dpStats C.llgo_bt_a2dp_stats
func a2dpStats(total, written, dropped *uint32) int32

// ---- mailbox ----

//go:linkname poll C.llgo_bt_poll
func poll(kind *int32, status *int32, handle *uint32, aux *int32, addr *uint8) int32

//go:linkname nowMs C.llgo_bt_now_ms
func nowMs() uint32

// vTaskDelay yields the CPU for ticks. Every Go polling loop must call it:
// a busy loop starves the idle tasks and trips the task watchdog.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// gosched hands the fiber scheduler to the next runnable goroutine (the
// runtime's runtime.Gosched, reached by linkname so the runtime package is not
// imported - importing it would pull its finalizer/reflect machinery into the
// image). The poller yields through it between batches, and a bounded wait
// through it between deadline checks.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()

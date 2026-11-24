//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/ble_glue_<backend>.c - the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design; the IDF component archive provides the definitions at the final
// link.
package ble

import _ "unsafe" // for go:linkname

//go:linkname initHost C.llgo_ble_init
func initHost() int32

//go:linkname deinitHost C.llgo_ble_deinit
func deinitHost() int32

//go:linkname ready C.llgo_ble_ready
func ready() int32

//go:linkname printVersion C.llgo_ble_print_version
func printVersion() int32

//go:linkname nowMs C.llgo_ble_now_ms
func nowMs() uint32

//go:linkname poll C.llgo_ble_poll
func poll(out *Event) int32

//go:linkname scanStart C.llgo_ble_scan_start
func scanStart(itvlMs, windowMs, passive, filterDup, durationMs int32, opID uint32) int32

//go:linkname scanStop C.llgo_ble_scan_stop
func scanStop(opID uint32) int32

//go:linkname connect C.llgo_ble_connect
func connect(addr *uint8, addrType, timeoutMs int32, opID uint32) int32

//go:linkname cancel C.llgo_ble_cancel
func cancel(opID uint32) int32

//go:linkname disconnect C.llgo_ble_disconnect
func disconnect(connID uint32) int32

//go:linkname backendName C.llgo_ble_backend_name
func backendName(buf *uint8, n int32) int32

// vTaskDelay yields the CPU for ticks. The poller rests one tick per pass: a
// busy loop would starve the idle tasks and trip the task watchdog.
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

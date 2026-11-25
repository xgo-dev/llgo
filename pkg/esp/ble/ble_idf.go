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

//go:linkname bondedPeers C.llgo_ble_bond_list
func bondedPeers(out *uint8, maxPeers int32) int32

//go:linkname backendName C.llgo_ble_backend_name
func backendName(buf *uint8, n int32) int32

// ---- GATT client ----
//
// Same contract as the GAP commands: a control call returns 0 when the
// procedure was submitted to the host task and a NimBLE status when it was not,
// and the outcome arrives through the mailbox. The record accessors and the MTU
// query are plain reads of a table the host task filled and of a stack value
// NimBLE locks internally.

//go:linkname discSvcs C.llgo_ble_disc_svcs
func discSvcs(connID, opID uint32) int32

//go:linkname discChrs C.llgo_ble_disc_chrs
func discChrs(connID uint32, startHandle, endHandle uint16, opID uint32) int32

//go:linkname readAttr C.llgo_ble_read
func readAttr(connID uint32, handle uint16, opID uint32) int32

//go:linkname writeAttr C.llgo_ble_write
func writeAttr(connID uint32, handle uint16, data *uint8, n, mode int32, opID uint32) int32

//go:linkname svcCount C.llgo_ble_svc_count
func svcCount(connID uint32) int32

//go:linkname svcAt C.llgo_ble_svc_at
func svcAt(connID uint32, idx int32, out *svcRec) int32

//go:linkname chrCount C.llgo_ble_chr_count
func chrCount(connID uint32) int32

//go:linkname chrAt C.llgo_ble_chr_at
func chrAt(connID uint32, idx int32, out *chrRec) int32

//go:linkname mtuGet C.llgo_ble_mtu
func mtuGet(connID uint32, out *int32) int32

// ---- peripheral ----
//
// Same contract as the GAP commands: a control call returns 0 when the stack
// accepted it and a status when it did not, and the outcome arrives through
// the mailbox (EvAdvDone, EvServicesDone, EvPeriphConnect, ...). The value
// cache write and the registry accessors are plain copies into/from static
// tables the stack task and the Go task share under the glue's critical
// section.

//go:linkname advStart C.llgo_ble_adv_start
func advStart(adv *uint8, advLen int32, rsp *uint8, rspLen, itvlMs, connectable int32, opID uint32) int32

//go:linkname advStop C.llgo_ble_adv_stop
func advStop(opID uint32) int32

//go:linkname gattsDeclare C.llgo_ble_gatts_declare
func gattsDeclare(svcs *gsvcWire, nsvc int32, chrs *gchrWire, nchr int32, opID uint32) int32

//go:linkname gattsAttr C.llgo_ble_gatts_attr
func gattsAttr(idx int32, out *uint16) int32

//go:linkname gattsValue C.llgo_ble_gatts_value
func gattsValue(handle uint16, data *uint8, n int32) int32

//go:linkname gattsNotify C.llgo_ble_gatts_notify
func gattsNotify(connID uint32, handle uint16, data *uint8, n, indicate int32) int32

//go:linkname gattsClose C.llgo_ble_gatts_close
func gattsClose(connID uint32) int32

//go:linkname securityParams C.llgo_ble_security
func securityParams(ioCap, mitm, bond int32, passkey uint32, keySize int32) int32

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

//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/espnow_glue.c - the same pattern as pkg/esp/wifi/wifi_idf.go and
// runtime/internal/runtime/idf_freertos.go. The symbols stay undefined in
// llgo_out.o by design; the IDF component archive provides the definitions
// at the final link.
package espnow

import _ "unsafe" // for go:linkname

//go:linkname glueABI C.llgo_espnow_glue_abi
func glueABI() int32

//go:linkname initNow C.llgo_espnow_init
func initNow() int32

//go:linkname deinitNow C.llgo_espnow_deinit
func deinitNow() int32

//go:linkname version C.llgo_espnow_version
func version(out *uint32) int32

//go:linkname send C.llgo_espnow_send
func send(mac *uint8, data *uint8, len int32) int32

//go:linkname addPeer C.llgo_espnow_add_peer
func addPeer(mac *uint8, lmk *uint8, channel int32, ifIdx int32, encrypt int32) int32

//go:linkname delPeer C.llgo_espnow_del_peer
func delPeer(mac *uint8) int32

//go:linkname peerExists C.llgo_espnow_peer_exists
func peerExists(mac *uint8) int32

//go:linkname peerNum C.llgo_espnow_peer_num
func peerNum(total *int32, encrypted *int32) int32

//go:linkname setPMK C.llgo_espnow_set_pmk
func setPMK(key *uint8) int32

//go:linkname setPeerKey C.llgo_espnow_set_peer_key
func setPeerKey(mac *uint8, key *uint8) int32

//go:linkname setChannel C.llgo_espnow_set_channel
func setChannel(ch int32) int32

//go:linkname ownMAC C.llgo_espnow_own_mac
func ownMAC(mac *uint8, ifIdx int32)

//go:linkname poll C.llgo_espnow_poll
func poll(kind *int32, mac *uint8, aux *int32, data *uint8, length *int32) int32

// vTaskDelay yields the CPU for ticks. Every Go polling loop must call it:
// a busy loop starves the idle tasks and trips the task watchdog.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// gosched hands the fiber scheduler to the next runnable goroutine (the
// runtime's runtime.Gosched, reached by linkname like the wifi package's
// poller so the runtime package is not imported - importing it would pull
// its finalizer/reflect machinery into the image). The pump yields through
// it between quanta so other goroutines keep running.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()

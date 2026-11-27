//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/wifi_glue.c — the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design (host/make-artifact.sh lists them); the IDF component archive
// provides the definitions at the final link.
package wifi

import _ "unsafe" // for go:linkname

//go:linkname glueABI C.llgo_wifi_glue_abi
func glueABI() int32

//go:linkname setup C.llgo_wifi_setup
func setup() int32

//go:linkname eventsRegister C.llgo_wifi_events_register
func eventsRegister() int32

//go:linkname printVersion C.llgo_wifi_print_version
func printVersion() int32

//go:linkname start C.llgo_wifi_start
func start() int32

//go:linkname stop C.llgo_wifi_stop
func stop() int32

//go:linkname setMode C.llgo_wifi_set_mode
func setMode(mode int32) int32

//go:linkname setSTA C.llgo_wifi_set_sta
func setSTA(ssid *uint8, ssidLen int32, pass *uint8, passLen int32) int32

//go:linkname setAP C.llgo_wifi_set_ap
func setAP(ssid *uint8, ssidLen int32, pass *uint8, passLen int32,
	channel int32, hidden int32, maxSTA int32, authMode int32) int32

//go:linkname getAPConfig C.llgo_wifi_get_ap_config
func getAPConfig(ssid *uint8, ssidMax int32, ssidLen *int32,
	channel *int32, hidden *int32, maxSTA *int32, authMode *int32) int32

//go:linkname scanStart C.llgo_wifi_scan
func scanStart(blocking int32) int32

//go:linkname scanFetch C.llgo_wifi_scan_fetch
func scanFetch(out *ApInfo, max int32, count *int32) int32

//go:linkname setPowerSave C.llgo_wifi_set_ps
func setPowerSave(ps int32) int32

//go:linkname getPowerSave C.llgo_wifi_get_ps
func getPowerSave(ps *int32) int32

//go:linkname connect C.llgo_wifi_connect
func connect() int32

//go:linkname disconnect C.llgo_wifi_disconnect
func disconnect() int32

//go:linkname getIP C.llgo_wifi_get_ip
func getIP() uint32

//go:linkname getAPIP C.llgo_wifi_get_ap_ip
func getAPIP(ip *uint32) int32

//go:linkname apStaList C.llgo_wifi_ap_sta_list
func apStaList(out *Station, max int32, count *int32) int32

//go:linkname maxSTA C.llgo_wifi_max_sta
func maxSTA() int32

//go:linkname deauthSta C.llgo_wifi_deauth_sta
func deauthSta(mac *uint8) int32

//go:linkname defaultSTA C.llgo_wifi_default_sta
func defaultSTA(ssid *uint8, ssidMax int32, ssidLen *int32,
	pass *uint8, passMax int32, passLen *int32)

//go:linkname finish C.llgo_wifi_finish
func finish()

//go:linkname heapFree C.llgo_wifi_heap_free
func heapFree() uint32

//go:linkname heapMin C.llgo_wifi_heap_min
func heapMin() uint32

//go:linkname heapInternalFree C.llgo_wifi_heap_internal_free
func heapInternalFree() uint32

//go:linkname heapSpiramFree C.llgo_wifi_heap_spiram_free
func heapSpiramFree() uint32

//go:linkname poll C.llgo_wifi_poll
func poll(kind *int32, aux *int32, ip *uint32, mac *uint8, aid *uint16) int32

// vTaskDelay yields the CPU for ticks. Every Go polling loop must call it:
// a busy loop starves the idle tasks and trips the task watchdog.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// gosched hands the fiber scheduler to the next runnable goroutine (the
// runtime's runtime.Gosched, reached by linkname like the demo's HeapStats
// hook so the runtime package is not imported - importing it would pull its
// finalizer/reflect machinery into the image). The subscriber poller yields
// through it between quanta so other goroutines keep running.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()

//go:linkname getMAC C.llgo_wifi_get_mac
func getMAC(out *uint8) int32

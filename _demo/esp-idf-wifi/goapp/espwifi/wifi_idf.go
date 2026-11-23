//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/wifi_glue.c — the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design (host/make-artifact.sh lists them); the IDF component archive
// provides the definitions at the final link.
package espwifi

import _ "unsafe" // for go:linkname

//go:linkname setup C.llgo_wifi_setup
func setup() int32

//go:linkname eventsRegister C.llgo_wifi_events_register
func eventsRegister() int32

//go:linkname printVersion C.llgo_wifi_print_version
func printVersion() int32

//go:linkname start C.llgo_wifi_start
func start() int32

//go:linkname setSTA C.llgo_wifi_set_sta
func setSTA(ssid *uint8, ssidLen int32, pass *uint8, passLen int32) int32

//go:linkname scanStart C.llgo_wifi_scan
func scanStart(blocking int32) int32

//go:linkname scanFetch C.llgo_wifi_scan_fetch
func scanFetch(out *ApInfo, max int32, count *int32) int32

//go:linkname poll C.llgo_wifi_poll
func poll(kind *int32, aux *int32, ip *uint32) int32

// vTaskDelay yields the CPU for ticks. Every Go polling loop must call it:
// a busy loop starves the idle tasks and trips the task watchdog.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

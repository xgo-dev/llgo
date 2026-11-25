//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/touch_glue.c - the same pattern as pkg/esp/gpio/gpio_idf.go. The
// symbols stay undefined in llgo_out.o by design; the IDF component archive
// provides the definitions at the final link.
package touch

import _ "unsafe" // for go:linkname

//go:linkname init_ C.llgo_touch_init
func init_(chargeTimes, divNum, powerOnWaitUs, measIntervalUs, maxMeasTimeUs, outMode int32) int32

//go:linkname configFilter C.llgo_touch_config_filter
func configFilter(bmMode, jitterStep, denoiseLvl, smooth, hysteresis, debounce int32) int32

//go:linkname configChannel C.llgo_touch_config_channel
func configChannel(chanID int32, activeThresh int32) int32

//go:linkname start C.llgo_touch_start
func start() int32

//go:linkname stop C.llgo_touch_stop
func stop() int32

//go:linkname readChan_ C.llgo_touch_read
func readChan_(chanID int32, dataType int32, out *uint32) int32

//go:linkname calibrate C.llgo_touch_recalibrate
func calibrate(chanID int32) int32

//go:linkname setThreshold C.llgo_touch_set_threshold
func setThreshold(chanID int32, thresh int32) int32

//go:linkname pollEvent C.llgo_touch_poll_event
func pollEvent(outChan *int32, outActive *int32) bool

//go:linkname deinit C.llgo_touch_deinit
func deinit() int32

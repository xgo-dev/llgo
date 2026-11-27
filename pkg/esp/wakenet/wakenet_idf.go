//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/wakenet_glue.c - the same pattern as pkg/esp/i2s/i2s_idf.go. The
// records cross the boundary as pointers to a mirrored struct
// (evRecord in wakenet.go / llgo_wakenet_ev_t in the glue); everything else
// crosses as integers, floats never cross at all (the threshold travels as
// thousandths).
package wakenet

import _ "unsafe" // for go:linkname

//go:linkname wnInit C.llgo_wakenet_init
func wnInit(model *uint8, modelLen int32, source int32, bclk int32, ws int32,
	din int32, thresholdMilli int32) int32

//go:linkname wnChunk C.llgo_wakenet_chunk
func wnChunk(chunkSamples *int32, sampleRate *int32) int32

//go:linkname wnModelName C.llgo_wakenet_model_name
func wnModelName(out *uint8, cap int32) int32

//go:linkname wnStart C.llgo_wakenet_start
func wnStart() int32

//go:linkname wnStop C.llgo_wakenet_stop
func wnStop() int32

//go:linkname wnClose C.llgo_wakenet_close
func wnClose() int32

//go:linkname wnDetected C.llgo_wakenet_detected
func wnDetected() int32

//go:linkname wnPoll C.llgo_wakenet_poll
func wnPoll(out *evRecord) int32

//go:linkname wnOverflowDelta C.llgo_wakenet_overflow_delta
func wnOverflowDelta() int32

//go:linkname wnStats C.llgo_wakenet_stats
func wnStats(frames *int32, avgUs *int32, maxUs *int32) int32

//go:linkname wnLastRMS C.llgo_wakenet_last_rms
func wnLastRMS() int32

//go:linkname wnPeakRMS C.llgo_wakenet_peak_rms
func wnPeakRMS() int32

//go:linkname wnHits C.llgo_wakenet_hits
func wnHits() int32

// vTaskDelay yields the CPU for ticks. The poller rests one tick per pass: a
// measured FreeRTOS tick (10 ms at the IDF default 100 Hz), which also feeds
// the idle tasks and the watchdog.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// gosched hands the fiber scheduler to the next runnable goroutine (the
// runtime's own Gosched): the poller's yield between batches, so a burst of
// records cannot hold the scheduler.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()

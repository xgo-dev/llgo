//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host. What a host build can still
// exercise - the config validation, the state-machine guards, the error
// spaces - is covered in wakenet_test.go; the real engine behavior is
// verified under esp-emu by the _demo/esp-idf-wakenet probe.
package wakenet

import "time"

const errUnsupported = codeUnsupported // 0x7F00, the family's stub tag

func wnInit(model *uint8, modelLen int32, source int32, bclk int32, ws int32,
	din int32, thresholdMilli int32) int32 {
	_, _, _, _, _, _, _ = model, modelLen, source, bclk, ws, din, thresholdMilli
	return errUnsupported
}

func wnChunk(chunkSamples *int32, sampleRate *int32) int32 {
	_, _ = chunkSamples, sampleRate
	return errUnsupported
}

func wnModelName(out *uint8, cap int32) int32 { _, _ = out, cap; return 0 }

func wnStart() int32 { return errUnsupported }
func wnStop() int32  { return errUnsupported }
func wnClose() int32 { return errUnsupported }

func wnDetected() int32 { return 0 }

func wnPoll(out *evRecord) int32 { _ = out; return 0 }

func wnOverflowDelta() int32 { return 0 }

func wnStats(frames *int32, avgUs *int32, maxUs *int32) int32 {
	_, _, _ = frames, avgUs, maxUs
	return errUnsupported
}

// vTaskDelay rests the host thread for the ticks a firmware build would
// yield (10 ms per tick at the IDF default 100 Hz).
func vTaskDelay(ticks uint32) { time.Sleep(time.Duration(ticks) * 10 * time.Millisecond) }

// gosched is a no-op on the host: the stub build never runs the poller (Init
// refuses with ErrUnsupported before the goroutine starts), and the host
// scheduler preempts on its own.
func gosched() {}

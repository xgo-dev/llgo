//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package lpcore

const errUnsupported = 0x7F00

func load(prog *byte, length int32) int32 {
	_, _ = prog, length
	return errUnsupported
}
func run(wakeupSource int32, timerSleepUs int32) int32 {
	_, _ = wakeupSource, timerSleepUs
	return errUnsupported
}
func stop() int32 { return errUnsupported }
func mailboxSend(msg int32) int32 {
	_ = msg
	return errUnsupported
}
func mailboxRecv(out *int32) int32 { _ = out; return errUnsupported }

func vTaskDelay(ticks uint32) { _ = ticks }
func gosched()                {}

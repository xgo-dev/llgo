//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package i2s

const errUnsupported = 0x7F00

func newChannel(handle *uintptr, tx int32, slave int32) int32 {
	_, _, _ = handle, tx, slave
	return errUnsupported
}
func initStd(chan_ uintptr, mclk int32, bclk int32, ws int32, dout int32,
	din int32, bits int32, slotMode int32, sampleRateHz int32) int32 {
	_, _, _, _, _, _, _, _, _ = chan_, mclk, bclk, ws, dout, din, bits, slotMode, sampleRateHz
	return errUnsupported
}
func enable(chan_ uintptr) int32  { _ = chan_; return errUnsupported }
func disable(chan_ uintptr) int32 { _ = chan_; return errUnsupported }
func write(chan_ uintptr, data *uint8, length int32, written *int32, timeoutMs int32) int32 {
	_, _, _, _, _ = chan_, data, length, written, timeoutMs
	return errUnsupported
}
func read(chan_ uintptr, data *uint8, length int32, filled *int32, timeoutMs int32) int32 {
	_, _, _, _, _ = chan_, data, length, filled, timeoutMs
	return errUnsupported
}
func del(chan_ uintptr) int32 { _ = chan_; return errUnsupported }

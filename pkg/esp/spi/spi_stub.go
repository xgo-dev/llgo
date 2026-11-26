//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package spi

const errUnsupported = 0x7F00

func init_(mosi, miso, sclk, cs int32, hz uint32) int32 {
	_, _, _, _, _ = mosi, miso, sclk, cs, hz
	return errUnsupported
}
func setFrequency(hz uint32) int32 { _ = hz; return errUnsupported }
func transfer(tx *byte, txLen int32, rx *byte, rxLen int32) int32 {
	_, _, _, _ = tx, txLen, rx, rxLen
	return errUnsupported
}
func write(tx *byte, txLen int32) int32 {
	_, _ = tx, txLen
	return errUnsupported
}
func writeRead(tx *byte, txLen int32, rx *byte, rxLen int32) int32 {
	_, _, _, _ = tx, txLen, rx, rxLen
	return errUnsupported
}
func actualFreq(outKHz *int32) int32 { _ = outKHz; return errUnsupported }
func acquireBus() int32              { return errUnsupported }
func releaseBus() int32              { return errUnsupported }
func deinit() int32                  { return errUnsupported }

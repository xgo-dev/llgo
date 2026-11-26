//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package adc

const errUnsupported = 0x7F00

func init_(unit int32, bitwidth int32) int32 {
	_, _ = unit, bitwidth
	return errUnsupported
}
func config_(pin int32, atten int32) int32 {
	_, _ = pin, atten
	return errUnsupported
}
func read_(pin int32, outRaw *int32) int32 {
	_, _ = pin, outRaw
	return errUnsupported
}
func readMillivolts_(pin int32, outMV *int32) int32 {
	_, _ = pin, outMV
	return errUnsupported
}
func deinit_(unit int32) int32 {
	_ = unit
	return errUnsupported
}
func efuseVersion(outVer *int32) int32 {
	_ = outVer
	return errUnsupported
}

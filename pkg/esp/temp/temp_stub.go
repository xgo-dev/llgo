//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding reports errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package temp

const errUnsupported = 0x7F00

func install(rangeMinC, rangeMaxC, allowPD int32) int32 {
	_, _, _ = rangeMinC, rangeMaxC, allowPD
	return errUnsupported
}
func start() int32 { return errUnsupported }
func readMilli(outMilliC *int32) int32 {
	_ = outMilliC
	return errUnsupported
}
func stop() int32      { return errUnsupported }
func uninstall() int32 { return errUnsupported }

//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package gpio

const errUnsupported = 0x7F00

func setDirection(pin int32, mode int32, pull int32) int32 {
	_, _, _ = pin, mode, pull
	return errUnsupported
}
func setLevel(pin int32, level int32) int32 {
	_, _ = pin, level
	return errUnsupported
}
func getLevel(pin int32) int32 { _ = pin; return 0 }
func toggle(pin int32) int32   { _ = pin; return errUnsupported }

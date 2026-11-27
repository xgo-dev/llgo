//go:build !llgo.idf && !(llgo && baremetal && (esp32c3 || esp32s31))

// Every build without a pin driver behind it: a hosted go build (go build,
// go vet, gopls), a baremetal llgo target this package has no register table
// for (Cortex-M, AVR, Xtensa, ...), and anything whose tags match neither
// backend. Every binding returns errUnsupported instead of faking a pin.
// Nothing here is meant to run on a host.
package gpio

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

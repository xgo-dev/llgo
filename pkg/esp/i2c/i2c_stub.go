//go:build !llgo.idf && !(llgo && baremetal && (esp32c3 || esp32s31))

// Every build without a transfer driver behind it: a hosted go build (go
// build, go vet, gopls), a baremetal llgo target this package has no register
// table for (Cortex-M, AVR, Xtensa, ...), and anything whose tags match
// neither backend. Every binding returns errUnsupported instead of faking a
// transfer. Nothing here is meant to run on a host.
package i2c

func newBus(handle *uintptr, sda int32, scl int32) int32 {
	_, _, _ = handle, sda, scl
	return errUnsupported
}
func addDevice(bus uintptr, handle *uintptr, addr uint32, speedHz uint32) int32 {
	_, _, _, _ = bus, handle, addr, speedHz
	return errUnsupported
}
func transmit(dev uintptr, data *uint8, length int32, timeoutMs int32) int32 {
	_, _, _, _ = dev, data, length, timeoutMs
	return errUnsupported
}
func receive(dev uintptr, data *uint8, length int32, timeoutMs int32) int32 {
	_, _, _, _ = dev, data, length, timeoutMs
	return errUnsupported
}
func transmitReceive(dev uintptr, write *uint8, writeLen int32,
	read *uint8, readLen int32, timeoutMs int32) int32 {
	_, _, _, _, _, _ = dev, write, writeLen, read, readLen, timeoutMs
	return errUnsupported
}
func probe(bus uintptr, addr uint32, timeoutMs int32) int32 {
	_, _, _ = bus, addr, timeoutMs
	return errUnsupported
}
func rmDevice(dev uintptr) int32 { _ = dev; return errUnsupported }
func delBus(bus uintptr) int32   { _ = bus; return errUnsupported }

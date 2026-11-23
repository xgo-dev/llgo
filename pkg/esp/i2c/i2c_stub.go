//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package i2c

const errUnsupported = 0x7F00

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
func rmDevice(dev uintptr) int32 { _ = dev; return errUnsupported }
func delBus(bus uintptr) int32   { _ = bus; return errUnsupported }

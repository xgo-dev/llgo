//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package uart

const errUnsupported = 0x7F00

func uartInit(port int32, txPin int32, rxPin int32, baud int32) int32 {
	_, _, _, _ = port, txPin, rxPin, baud
	return errUnsupported
}
func uartSetBaud(port int32, baud int32) int32 {
	_, _ = port, baud
	return errUnsupported
}
func uartSetConfig(port int32, wordLen int32, stopBits int32, parity int32) int32 {
	_, _, _, _ = port, wordLen, stopBits, parity
	return errUnsupported
}
func uartWrite(port int32, data *byte, length int32, outN *int32) int32 {
	_, _, _ = port, data, length
	*outN = 0
	return errUnsupported
}
func uartRead(port int32, buf *byte, length int32, timeoutMs int32, outN *int32) int32 {
	_, _, _, _ = port, buf, length, timeoutMs
	*outN = 0
	return errUnsupported
}
func uartAvailable(port int32, outN *int32) int32 {
	_ = port
	*outN = 0
	return errUnsupported
}
func uartFlushInput(port int32) int32 { _ = port; return errUnsupported }
func uartFlushOutput(port int32, timeoutMs int32) int32 {
	_, _ = port, timeoutMs
	return errUnsupported
}
func uartDeinit(port int32) int32 { _ = port; return errUnsupported }

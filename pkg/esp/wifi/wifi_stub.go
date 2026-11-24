//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported and
// vTaskDelay is a no-op. Nothing here is meant to run on a host.
package wifi

// errUnsupported is what every stub below returns: it lies between the real
// esp_err ranges so it can never be mistaken for one.
const errUnsupported = 0x7F00

func setup() int32          { return errUnsupported }
func eventsRegister() int32 { return errUnsupported }
func printVersion() int32   { return errUnsupported }
func start() int32          { return errUnsupported }
func setSTA(ssid *uint8, ssidLen int32, pass *uint8, passLen int32) int32 {
	_, _, _, _ = ssid, ssidLen, pass, passLen
	return errUnsupported
}
func scanStart(blocking int32) int32 { _ = blocking; return errUnsupported }
func scanFetch(out *ApInfo, max int32, count *int32) int32 {
	_, _, _ = out, max, count
	return errUnsupported
}
func connect() int32    { return errUnsupported }
func disconnect() int32 { return errUnsupported }
func getIP() uint32     { return 0 }
func defaultSTA(ssid *uint8, ssidMax int32, ssidLen *int32,
	pass *uint8, passMax int32, passLen *int32) {
	_, _, _, _, _, _ = ssid, ssidMax, ssidLen, pass, passMax, passLen
}
func finish()          {}
func heapFree() uint32 { return 0 }

func setModeSTA() int32 { return errUnsupported }

func heapInternalFree() uint32 { return 0 }

func heapSpiramFree() uint32 { return 0 }
func heapMin() uint32        { return 0 }
func poll(kind *int32, aux *int32, ip *uint32) int32 {
	_, _, _ = kind, aux, ip
	return 0
}

func vTaskDelay(ticks uint32) { _ = ticks }

// gosched is a no-op on the host: the stub build never runs the subscriber
// poller, and the host scheduler has no fibers to hand off to.
func gosched() {}

//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported and
// vTaskDelay is a no-op. Nothing here is meant to run on a host - the
// package's hosted tests exercise the pure-Go layer (config validation,
// error naming, event fan-out) and check that the driver calls report
// errUnsupported instead of pretending to work. In particular the
// Subscribe poller must not be started on a host: vTaskDelay is a no-op
// there, so its loop would spin at full CPU (see wifi_test.go).
package wifi

// errUnsupported is what every stub below returns: it lies between the real
// esp_err ranges so it can never be mistaken for one.
const errUnsupported = 0x7F00

// glueABI reports the contract version of the host stub. There is no C glue in
// a host build, and this stub compiles from the same source as the package it
// stands in for, so it always speaks the current version: the version check
// exists to catch stale copies of the C template, which a host build has none
// of, and Setup must keep reporting errUnsupported rather than a fake mismatch.
func glueABI() int32 { return glueABIVersion }

func setup() int32          { return errUnsupported }
func eventsRegister() int32 { return errUnsupported }
func printVersion() int32   { return errUnsupported }
func start() int32          { return errUnsupported }
func stop() int32           { return errUnsupported }
func setMode(mode int32) int32 {
	_ = mode
	return errUnsupported
}
func setSTA(ssid *uint8, ssidLen int32, pass *uint8, passLen int32) int32 {
	_, _, _, _ = ssid, ssidLen, pass, passLen
	return errUnsupported
}
func setAP(ssid *uint8, ssidLen int32, pass *uint8, passLen int32,
	channel int32, hidden int32, maxSTA int32, authMode int32) int32 {
	_, _, _, _, _, _, _, _ = ssid, ssidLen, pass, passLen, channel, hidden, maxSTA, authMode
	return errUnsupported
}
func getAPConfig(ssid *uint8, ssidMax int32, ssidLen *int32,
	channel *int32, hidden *int32, maxSTA *int32, authMode *int32) int32 {
	_, _, _, _, _, _, _ = ssid, ssidMax, ssidLen, channel, hidden, maxSTA, authMode
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
func getAPIP(ip *uint32) int32 {
	_ = ip
	return errUnsupported
}
func apStaList(out *Station, max int32, count *int32) int32 {
	_, _, _ = out, max, count
	return errUnsupported
}
func maxSTA() int32 { return MaxAPStations }
func deauthSta(mac *uint8) int32 {
	_ = mac
	return errUnsupported
}
func defaultSTA(ssid *uint8, ssidMax int32, ssidLen *int32,
	pass *uint8, passMax int32, passLen *int32) {
	_, _, _, _, _, _ = ssid, ssidMax, ssidLen, pass, passMax, passLen
}
func finish()          {}
func heapFree() uint32 { return 0 }

func heapInternalFree() uint32 { return 0 }

func heapSpiramFree() uint32 { return 0 }
func heapMin() uint32        { return 0 }
func poll(kind *int32, aux *int32, ip *uint32, mac *uint8, aid *uint16) int32 {
	_, _, _, _, _ = kind, aux, ip, mac, aid
	return 0
}

func vTaskDelay(ticks uint32) { _ = ticks }

// gosched is a no-op on the host: the stub build never runs the subscriber
// poller, and the host scheduler has no fibers to hand off to.
func gosched() {}

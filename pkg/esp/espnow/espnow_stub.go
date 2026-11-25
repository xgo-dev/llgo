//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported and
// vTaskDelay is a no-op. Nothing here is meant to run on a host - the
// package's hosted tests exercise the pure-Go layer (payload validation,
// error naming, the recv dispatch and the send-result queues) and check
// that the driver calls report errUnsupported instead of pretending to
// work. In particular Init must never start the pump goroutine on a host:
// vTaskDelay is a no-op there, so its loop would spin at full CPU (see
// espnow_test.go).
package espnow

func glueABI() int32 { return glueABIVersion }

func initNow() int32   { return errUnsupported }
func deinitNow() int32 { return errUnsupported }
func version(out *uint32) int32 {
	_ = out
	return errUnsupported
}
func send(mac *uint8, data *uint8, length int32) int32 {
	_, _, _ = mac, data, length
	return errUnsupported
}
func addPeer(mac *uint8, lmk *uint8, channel int32, ifIdx int32, encrypt int32) int32 {
	_, _, _, _, _ = mac, lmk, channel, ifIdx, encrypt
	return errUnsupported
}
func delPeer(mac *uint8) int32 {
	_ = mac
	return errUnsupported
}
func peerExists(mac *uint8) int32 {
	_ = mac
	return 0
}
func peerNum(total *int32, encrypted *int32) int32 {
	_, _ = total, encrypted
	return errUnsupported
}
func setPMK(key *uint8) int32 {
	_ = key
	return errUnsupported
}
func setPeerKey(mac *uint8, key *uint8) int32 {
	_, _ = mac, key
	return errUnsupported
}
func setChannel(ch int32) int32 {
	_ = ch
	return errUnsupported
}
func ownMAC(mac *uint8, ifIdx int32) {
	_, _ = mac, ifIdx
}
func poll(kind *int32, mac *uint8, aux *int32, data *uint8, length *int32) int32 {
	_, _, _, _, _ = kind, mac, aux, data, length
	return 0
}

func vTaskDelay(ticks uint32) { _ = ticks }

// gosched is a no-op on the host: the stub build never runs the pump, and
// the host scheduler has no fibers to hand off to.
func gosched() {}

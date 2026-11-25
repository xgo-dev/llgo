//go:build !llgo.idf

// Host build (go build / go vet / gopls): the C glue only exists in the
// ESP-IDF link environment, so every binding returns errUnsupported. It lies
// between the real esp_err ranges so it can never be mistaken for one.
// Nothing here is meant to run on a host.
package nvs

const errUnsupported = 0x7F00

func flashInit() int32   { return errUnsupported }
func flashDeinit() int32 { return errUnsupported }

func openNs(ns *byte, nsLen int32, handle *uintptr) int32 {
	_, _, _ = ns, nsLen, handle
	return errUnsupported
}
func closeNs(handle uintptr) { _ = handle }

func getU8(h uintptr, key *byte, keyLen int32, out *uint8) int32 {
	_, _, _, _ = h, key, keyLen, out
	return errUnsupported
}
func getI32(h uintptr, key *byte, keyLen int32, out *int32) int32 {
	_, _, _, _ = h, key, keyLen, out
	return errUnsupported
}
func getU64(h uintptr, key *byte, keyLen int32, out *uint64) int32 {
	_, _, _, _ = h, key, keyLen, out
	return errUnsupported
}
func getStr(h uintptr, key *byte, keyLen int32, outLen *int32, outBuf **byte) int32 {
	_, _, _, _, _ = h, key, keyLen, outLen, outBuf
	return errUnsupported
}
func getBlob(h uintptr, key *byte, keyLen int32, outLen *int32, outBuf **byte) int32 {
	_, _, _, _, _ = h, key, keyLen, outLen, outBuf
	return errUnsupported
}
func setU8(h uintptr, key *byte, keyLen int32, v uint8) int32 {
	_, _, _, _ = h, key, keyLen, v
	return errUnsupported
}
func setI32(h uintptr, key *byte, keyLen int32, v int32) int32 {
	_, _, _, _ = h, key, keyLen, v
	return errUnsupported
}
func setU64(h uintptr, key *byte, keyLen int32, v uint64) int32 {
	_, _, _, _ = h, key, keyLen, v
	return errUnsupported
}
func setStr(h uintptr, key *byte, keyLen int32, val *byte) int32 {
	_, _, _, _ = h, key, keyLen, val
	return errUnsupported
}
func setBlob(h uintptr, key *byte, keyLen int32, val *byte, valLen int32) int32 {
	_, _, _, _, _ = h, key, keyLen, val, valLen
	return errUnsupported
}
func eraseKey(h uintptr, key *byte, keyLen int32) int32 {
	_, _, _ = h, key, keyLen
	return errUnsupported
}
func commit(h uintptr) int32 { _ = h; return errUnsupported }

func listKeys(ns *byte, nsLen int32, outCount *int32, outTable **byte) int32 {
	_, _, _, _ = ns, nsLen, outCount, outTable
	return errUnsupported
}
func getStats(used *int32, free *int32, available *int32, total *int32, nsCount *int32) int32 {
	_, _, _, _, _ = used, free, available, total, nsCount
	return errUnsupported
}

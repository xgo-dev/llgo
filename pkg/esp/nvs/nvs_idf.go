//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/nvs_glue.c - the same pattern as pkg/esp/wifi/wifi_idf.go. Handles
// cross the boundary as uintptr integers; the glue casts them back to the
// C handle types. Strings travel as pointer/length pairs (the glue
// NUL-terminates into bounded locals); Get results travel as a pointer to
// the glue's static buffer plus its length, copied out immediately.
package nvs

import _ "unsafe" // for go:linkname

//go:linkname flashInit C.llgo_nvs_flash_init
func flashInit() int32

//go:linkname flashDeinit C.llgo_nvs_flash_deinit
func flashDeinit() int32

//go:linkname openNs C.llgo_nvs_open
func openNs(ns *byte, nsLen int32, handle *uintptr) int32

//go:linkname closeNs C.llgo_nvs_close
func closeNs(handle uintptr)

//go:linkname getU8 C.llgo_nvs_get_u8
func getU8(h uintptr, key *byte, keyLen int32, out *uint8) int32

//go:linkname getI32 C.llgo_nvs_get_i32
func getI32(h uintptr, key *byte, keyLen int32, out *int32) int32

//go:linkname getU64 C.llgo_nvs_get_u64
func getU64(h uintptr, key *byte, keyLen int32, out *uint64) int32

//go:linkname getStr C.llgo_nvs_get_str
func getStr(h uintptr, key *byte, keyLen int32, outLen *int32, outBuf **byte) int32

//go:linkname getBlob C.llgo_nvs_get_blob
func getBlob(h uintptr, key *byte, keyLen int32, outLen *int32, outBuf **byte) int32

//go:linkname setU8 C.llgo_nvs_set_u8
func setU8(h uintptr, key *byte, keyLen int32, v uint8) int32

//go:linkname setI32 C.llgo_nvs_set_i32
func setI32(h uintptr, key *byte, keyLen int32, v int32) int32

//go:linkname setU64 C.llgo_nvs_set_u64
func setU64(h uintptr, key *byte, keyLen int32, v uint64) int32

//go:linkname setStr C.llgo_nvs_set_str
func setStr(h uintptr, key *byte, keyLen int32, val *byte) int32

//go:linkname setBlob C.llgo_nvs_set_blob
func setBlob(h uintptr, key *byte, keyLen int32, val *byte, valLen int32) int32

//go:linkname eraseKey C.llgo_nvs_erase_key
func eraseKey(h uintptr, key *byte, keyLen int32) int32

//go:linkname commit C.llgo_nvs_commit
func commit(h uintptr) int32

//go:linkname listKeys C.llgo_nvs_keys
func listKeys(ns *byte, nsLen int32, outCount *int32, outTable **byte) int32

//go:linkname getStats C.llgo_nvs_stats
func getStats(used *int32, free *int32, available *int32, total *int32, nsCount *int32) int32

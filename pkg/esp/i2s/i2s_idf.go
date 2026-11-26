//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/i2s_glue.c - the same pattern as pkg/esp/wifi/wifi_idf.go. Handles
// cross the boundary as uintptr integers; the glue casts them back to the
// C handle types.
package i2s

import _ "unsafe" // for go:linkname

//go:linkname newChannel C.llgo_i2s_new_channel
func newChannel(handle *uintptr, tx int32) int32

//go:linkname initStd C.llgo_i2s_init_std
func initStd(chan_ uintptr, mclk int32, bclk int32, ws int32, dout int32,
	din int32, bits int32, slotMode int32, sampleRateHz int32) int32

//go:linkname enable C.llgo_i2s_enable
func enable(chan_ uintptr) int32

//go:linkname disable C.llgo_i2s_disable
func disable(chan_ uintptr) int32

//go:linkname write C.llgo_i2s_write
func write(chan_ uintptr, data *uint8, length int32, written *int32, timeoutMs int32) int32

//go:linkname read C.llgo_i2s_read
func read(chan_ uintptr, data *uint8, length int32, filled *int32, timeoutMs int32) int32

//go:linkname del C.llgo_i2s_del
func del(chan_ uintptr) int32

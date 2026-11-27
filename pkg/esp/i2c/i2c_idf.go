//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/i2c_glue.c - the same pattern as pkg/esp/wifi/wifi_idf.go. Handles
// cross the boundary as uintptr integers; the glue casts them back to the
// C handle types.
package i2c

import _ "unsafe" // for go:linkname

//go:linkname newBus C.llgo_i2c_new_bus
func newBus(handle *uintptr, sda int32, scl int32) int32

//go:linkname addDevice C.llgo_i2c_add_device
func addDevice(bus uintptr, handle *uintptr, addr uint32, speedHz uint32) int32

//go:linkname transmit C.llgo_i2c_transmit
func transmit(dev uintptr, data *uint8, length int32, timeoutMs int32) int32

//go:linkname receive C.llgo_i2c_receive
func receive(dev uintptr, data *uint8, length int32, timeoutMs int32) int32

//go:linkname transmitReceive C.llgo_i2c_transmit_receive
func transmitReceive(dev uintptr, write *uint8, writeLen int32,
	read *uint8, readLen int32, timeoutMs int32) int32

//go:linkname probe C.llgo_i2c_probe
func probe(bus uintptr, addr uint32, timeoutMs int32) int32

//go:linkname rmDevice C.llgo_i2c_rm_device
func rmDevice(dev uintptr) int32

//go:linkname delBus C.llgo_i2c_del_bus
func delBus(bus uintptr) int32

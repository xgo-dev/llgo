//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/spi_glue.c - the same pattern as pkg/esp/gpio/gpio_idf.go. The
// symbols stay undefined in llgo_out.o by design; the IDF component archive
// provides the definitions at the final link.
package spi

import _ "unsafe" // for go:linkname

//go:linkname init_ C.llgo_spi_init
func init_(mosi, miso, sclk, cs int32, hz uint32) int32

//go:linkname setFrequency C.llgo_spi_set_frequency
func setFrequency(hz uint32) int32

//go:linkname transfer C.llgo_spi_transfer
func transfer(tx *byte, txLen int32, rx *byte, rxLen int32) int32

//go:linkname write C.llgo_spi_write
func write(tx *byte, txLen int32) int32

//go:linkname writeRead C.llgo_spi_write_read
func writeRead(tx *byte, txLen int32, rx *byte, rxLen int32) int32

//go:linkname actualFreq C.llgo_spi_actual_freq_khz
func actualFreq(outKHz *int32) int32

//go:linkname acquireBus C.llgo_spi_acquire_bus
func acquireBus() int32

//go:linkname releaseBus C.llgo_spi_release_bus
func releaseBus() int32

//go:linkname deinit C.llgo_spi_deinit
func deinit() int32

//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/adc_glue.c - the same pattern as pkg/esp/spi/spi_idf.go. The
// symbols stay undefined in llgo_out.o by design; the IDF component archive
// provides the definitions at the final link.
package adc

import _ "unsafe" // for go:linkname

//go:linkname init_ C.llgo_adc_init
func init_(unit int32, bitwidth int32) int32

//go:linkname config_ C.llgo_adc_config
func config_(pin int32, atten int32) int32

//go:linkname read_ C.llgo_adc_read
func read_(pin int32, outRaw *int32) int32

//go:linkname readMillivolts_ C.llgo_adc_read_millivolts
func readMillivolts_(pin int32, outMV *int32) int32

//go:linkname deinit_ C.llgo_adc_deinit
func deinit_(unit int32) int32

//go:linkname efuseVersion C.llgo_adc_efuse_version
func efuseVersion(outVer *int32) int32

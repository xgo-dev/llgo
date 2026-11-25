//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/temp_glue.c - the same pattern as pkg/esp/gpio/gpio_idf.go. The
// symbols stay undefined in llgo_out.o by design; the IDF component archive
// provides the definitions at the final link.
package temp

import _ "unsafe" // for go:linkname

//go:linkname install C.llgo_temp_install
func install(rangeMinC, rangeMaxC, allowPD int32) int32

//go:linkname start C.llgo_temp_start
func start() int32

//go:linkname readMilli C.llgo_temp_read_milli
func readMilli(outMilliC *int32) int32

//go:linkname stop C.llgo_temp_stop
func stop() int32

//go:linkname uninstall C.llgo_temp_uninstall
func uninstall() int32

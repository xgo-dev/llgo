//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/gpio_glue.c - the same pattern as pkg/esp/wifi/wifi_idf.go and
// runtime/internal/runtime/idf_freertos.go. The symbols stay undefined in
// llgo_out.o by design; the IDF component archive provides the definitions
// at the final link.
package gpio

import _ "unsafe" // for go:linkname

//go:linkname setDirection C.llgo_gpio_set_direction
func setDirection(pin int32, mode int32, pull int32) int32

//go:linkname setLevel C.llgo_gpio_set_level
func setLevel(pin int32, level int32) int32

//go:linkname getLevel C.llgo_gpio_get_level
func getLevel(pin int32) int32

//go:linkname toggle C.llgo_gpio_toggle
func toggle(pin int32) int32

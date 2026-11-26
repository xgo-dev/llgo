//go:build llgo && baremetal && linux && arm && llgo.idf

package runtime

import "unsafe"

// llgoEntropyFill is the sysrand boundary behind crypto/rand.Read on
// baremetal targets (see runtime/_patch/crypto/internal/sysrand). On the
// llgo.idf targets it bridges to the ESP-IDF hardware TRNG: esp_fill_random
// is an external reference that the IDF link environment resolves from
// esp_hw_support, exactly like the FreeRTOS bindings in idf_freertos.go.
// The runtime tree compiles without IDF headers, so there is deliberately
// no _wrap C glue here. One call fills the whole buffer; the function has
// no failure mode.
//
//go:linkname llgoEntropyIdfFill runtime.llgoEntropyFill
func llgoEntropyIdfFill(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	cEspFillRandom(unsafe.Pointer(&b[0]), uintptr(len(b)))
	return true
}

//go:linkname cEspFillRandom C.esp_fill_random
func cEspFillRandom(buf unsafe.Pointer, length uintptr)

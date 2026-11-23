//go:build llgo && llgo.esp32c3.printf_float

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// The normal link supplies a no-op definition and newlib's float backend.
// A link omitting the backend succeeds only if LTO discards all callers.
//
//go:linkname requirePrintfFloat C.__llgo_esp32c3_requires_printf_float
func requirePrintfFloat()

func formatFloatWithC(v float64, format *c.Char) string {
	requirePrintfFloat()
	var buf [64]byte
	c.Snprintf((*c.Char)(unsafe.Pointer(&buf[0])), uintptr(len(buf)), format, v)
	return c.GoString((*c.Char)(unsafe.Pointer(&buf[0])))
}

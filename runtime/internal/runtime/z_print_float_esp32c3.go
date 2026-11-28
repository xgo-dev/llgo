//go:build llgo && llgo.esp32c3.printf_float

package runtime

import _ "unsafe"

// The normal link supplies a no-op definition and newlib's float backend.
// A link omitting the backend succeeds only if LTO discards all callers.
//
//go:linkname requirePrintfFloat C.__llgo_esp32c3_requires_printf_float
func requirePrintfFloat()

package writer

import _ "unsafe"

//go:linkname callbacks github.com/xgo-dev/llgo/cl/_testdrop/reflect_callback_foreign/names.callbacks
var callbacks []string

//go:noinline
func Replace() { callbacks[0] = "Foreign" }

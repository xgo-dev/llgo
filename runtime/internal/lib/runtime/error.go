package runtime

import (
	_ "unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

type TypeAssertionError = llruntime.TypeAssertionError
type PanicNilError = llruntime.PanicNilError

// The public runtime package aliases these LLGo runtime error types. Provide
// the method symbols under their public names as well, so method expressions
// and C-library builds can retain them.

//go:linkname typeAssertionErrorRuntimeError runtime.(*TypeAssertionError).RuntimeError
func typeAssertionErrorRuntimeError(e *llruntime.TypeAssertionError) {
	e.RuntimeError()
}

//go:linkname typeAssertionErrorError runtime.(*TypeAssertionError).Error
func typeAssertionErrorError(e *llruntime.TypeAssertionError) string {
	return e.Error()
}

//go:linkname panicNilErrorRuntimeError runtime.(*PanicNilError).RuntimeError
func panicNilErrorRuntimeError(e *llruntime.PanicNilError) {
	e.RuntimeError()
}

//go:linkname panicNilErrorError runtime.(*PanicNilError).Error
func panicNilErrorError(e *llruntime.PanicNilError) string {
	return e.Error()
}

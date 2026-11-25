//go:build darwin && arm64 && !baremetal && !wasm

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Keep in sync with ssa/eh.go useFastDarwinArm64DeferSetjmp.
//
//go:linkname deferLongjmp C.llgo_longjmp
func deferLongjmp(env unsafe.Pointer, val c.Int)

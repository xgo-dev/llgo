//go:build !darwin || !arm64 || baremetal || wasm

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

func deferLongjmp(env unsafe.Pointer, val c.Int) {
	c.Siglongjmp(env, val)
}

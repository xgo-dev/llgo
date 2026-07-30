//go:build linux && !wasm && !baremetal

package runtime

import c "github.com/goplus/llgo/runtime/internal/clite"

const SIGBUS = c.Int(0x7)

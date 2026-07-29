//go:build llgo && js && wasm

package runtime

import c "github.com/goplus/llgo/runtime/internal/clite"

func fatal(s string) {
	print("fatal error: ", s, "\n")
	c.Exit(2)
}

//go:build !js || !wasm

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func exitPanic() {
	crashAfterPanic()
	c.Exit(2)
}

//go:build llgo && js && wasm

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
)

func fatal(s string) {
	print("fatal error: ", s, "\n")
	exitPanic()
}

func exitPanic() {
	// A pending Asyncify host wait keeps the Emscripten runtime alive after
	// exit(2). Force the process to terminate so fatal scheduler errors retain
	// Go's observable non-zero-exit behavior.
	emscripten.ForceExit(c.Int(2))
}

//go:build !llgo.baremetal.directconsole && (!llgo || !js || !wasm || !llgo.wasm.emscripten)

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func printScalarBytes(data c.Pointer, size uint32) {
	c.Fwrite(data, 1, uintptr(size), c.Stderr)
}

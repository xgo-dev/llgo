//go:build llgo && js && wasm && !llgo.wasm.emscripten

package runtime

// The syscall/js source patch calls this host library through linknames, so
// GOROOT sources do not import LLGo's internal packages.
import _ "github.com/xgo-dev/llgo/runtime/internal/wasmjs"

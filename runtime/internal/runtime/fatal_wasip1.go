//go:build llgo && wasip1 && wasm && !llgo.wasi_threads

package runtime

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

//llgo:cold
//llgo:noreturn
func fatal(s string) {
	print("fatal error: ", s, "\n")
	c.Exit(2)
}

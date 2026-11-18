//go:build llgo && wasip1 && wasm && llgo.fmt.value

package fmt

// Load the typed formatting helpers for compiler-generated functions.
import _ "internal/llgofmt"

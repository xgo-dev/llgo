//go:build !llgo || !js || !wasm

package runtime

func fatal(s string) {
	print("fatal error: ", s, "\n")
}

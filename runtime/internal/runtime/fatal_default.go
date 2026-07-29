//go:build !llgo || !wasm || (!js && !wasip1)

package runtime

func fatal(s string) {
	print("fatal error: ", s, "\n")
}

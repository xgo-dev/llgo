//go:build !llgo || !wasm || (!js && !wasip1) || (wasip1 && llgo.wasi_threads)

package runtime

//go:noinline
func fatal(s string) {
	print("fatal error: ", s, "\n")
	exitPanic()
}

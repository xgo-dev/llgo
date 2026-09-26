package main

import "github.com/xgo-dev/llgo/dev/testdata/wasm-eh/go-cpp-boundary/cpp"

func main() {
	if cpp.Catch() != 7 {
		panic("C++ did not catch its exception")
	}
	defer func() {
		if recover() != "C++ status translated to Go panic" {
			panic("Go did not recover the translated status")
		}
		println("go cpp boundary ok")
	}()
	// Translate the C ABI status in Go. Never unwind a C++ exception through Go.
	panic("C++ status translated to Go panic")
}

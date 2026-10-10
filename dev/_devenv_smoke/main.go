package main

import (
	"fmt"
	_ "unsafe"
)

const (
	LLGoPackage = "link: $(pkg-config --libs bdw-gc libffi openssl zlib sqlite3 libuv)"
	LLGoFiles   = "$(pkg-config --cflags bdw-gc libffi openssl zlib sqlite3 libuv): _wrap/native.c"
)

//go:linkname checkNative C.llgo_check_native_dependencies
func checkNative() int32

func main() {
	if code := checkNative(); code != 0 {
		panic(fmt.Sprintf("native dependency check failed: %d", code))
	}
	fmt.Println("LLGo native dependencies work")
}

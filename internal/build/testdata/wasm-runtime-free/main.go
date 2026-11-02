package main

import "github.com/xgo-dev/llgo/internal/build/testdata/wasm-runtime-free/helper"

var state = helper.Next(40)

func main() {
	state = helper.Next(state)
	print("zero\x00 ")
	println("世界", state, false, int64(-9223372036854775808), uint64(18446744073709551615))
}

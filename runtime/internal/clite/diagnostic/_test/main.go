package main

import (
	"github.com/xgo-dev/llgo/runtime/internal/clite/diagnostic"
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	_ "runtime"
)

func main() {
	rt.PrintHex(0)
	println()
	rt.PrintHex(0xffffffffffffffff)
	println()
	diagnostic.Error("", 42)
	diagnostic.Error("zero: ", 0)
	diagnostic.Error("negative: ", -1)
	diagnostic.Error("minimum: ", -2147483648)
	diagnostic.Error("maximum: ", 2147483647)
}

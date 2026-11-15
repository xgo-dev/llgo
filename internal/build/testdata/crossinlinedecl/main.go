package main

import (
	"github.com/xgo-dev/llgo/internal/build/testdata/crossinlinedecl/decl"
	_ "github.com/xgo-dev/llgo/internal/build/testdata/crossinlinedecl/impl"
)

func main() {
	println(decl.Number(41).Value())
}

package main

import (
	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/stackcache/instance"
	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/stackcache/plain"
	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/stackcache/spawn"
)

func main() {
	if plain.Value() != 42 || spawn.Value() != 42 || instance.Value() != 42 {
		panic("stack cache regression")
	}
}

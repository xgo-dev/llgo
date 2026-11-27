package main

import (
	"fmt"

	"github.com/xgo-dev/llgo/cmd/xtool/env"
)

func main() {
	fmt.Println(env.ExpandEnv("$(pkg-config --libs bdw-gc)"))
}

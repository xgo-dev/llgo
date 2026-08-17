package main

import (
	"fmt"

	"github.com/xgo-dev/llgo/xtool/env"
)

func main() {
	fmt.Println(env.ExpandEnv("$(pkg-config --libs bdw-gc)"))
}

package main

import (
	"github.com/goplus/lib/c"
	_ "github.com/xgo-dev/llgo/_demo/embed/targetsbuild/C"
)

func main() {
	defer c.Printf(c.Str("defer 1\n"))
	defer c.Printf(c.Str("defer 2\n"))
	var counter int = 1

	if counter > 0 {
		defer c.Printf(c.Str("defer in if\n"))
	} else {
		defer c.Printf(c.Str("defer in else\n"))
	}

	for i := 0; i < 3; i++ {
		defer c.Printf(c.Str("defer in loop %d\n"), i)
	}
	panic("panic occured")
}

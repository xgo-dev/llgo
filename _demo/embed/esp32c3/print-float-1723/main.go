package main

import "github.com/goplus/lib/c"

func main() {
	println("go", 0)
	c.Fflush(nil)
	c.Printf(c.Str("f=%f\n"), 1.1)
}

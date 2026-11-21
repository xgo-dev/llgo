package main

import "github.com/goplus/lib/c"

//go:noinline
func format() *c.Char {
	return c.Str("C float %.2f\n")
}

func main() {
	c.Printf(format(), c.Double(1.25))
}

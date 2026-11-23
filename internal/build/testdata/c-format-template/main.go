package main

import "github.com/goplus/lib/c"

//go:noinline
func print(n c.Int) {
	var short [6]c.Char
	count := c.Snprintf(&short[0], uintptr(len(short)), c.Str("%s=%d:%04d"), c.Str("p%"), c.Int(-7), n)
	println(int(count), c.GoString(&short[0]))
	var whole [32]c.Char
	count = c.Snprintf(&whole[0], uintptr(len(whole)), c.Str("%d/%x"), c.Int(12), c.Uint(255))
	println(int(count), c.GoString(&whole[0]))
}

func main() {
	print(3)
}

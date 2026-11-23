package main

import (
	"fmt"
	"os"
)

//go:noinline
func emit(n int, s string, b bool) {
	fmt.Printf("%06d %q %v %% %X\n", n, s, b, uint16(255))
	println(fmt.Sprintf("%08x %4s %d", n, s, uint(5)))
	fmt.Fprintf(os.Stdout, "%o %s\n", n, s)
	dst := [2]byte{'p', '-'}
	println(string(fmt.Appendf(dst[:1], ":%t %s", b, "")), string(dst[:]))
}

func main() {
	emit(-15, "世界", true)
}

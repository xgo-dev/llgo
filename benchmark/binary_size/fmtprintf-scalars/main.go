package main

import "fmt"

//go:noinline
func show(i int64, u uint64, ok bool, s string, f float64, z complex128) {
	fmt.Printf("%d %d %t %s %g %g\n", i, u, ok, s, f, z)
}

func main() {
	show(-17, 42, true, "lab", 1.25, 2+3i)
}

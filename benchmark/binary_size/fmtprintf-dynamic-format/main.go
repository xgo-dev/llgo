package main

import (
	"fmt"
	"os"
)

//go:noinline
func printValue(format string, value float64) {
	fmt.Printf(format, value)
}

func main() {
	printValue(os.Args[1], 42.125)
}

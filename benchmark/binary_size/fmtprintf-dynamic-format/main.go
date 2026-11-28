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
	format := "%.2f\n"
	if len(os.Args) > 1 {
		format = os.Args[1]
	}
	printValue(format, 42.125)
}

package main

import (
	"fmt"
	"os"
)

//go:noinline
func printValue(value any) {
	fmt.Printf("%v\n", value)
}

func main() {
	if len(os.Args) > 1 {
		printValue(os.Args[1])
		return
	}
	printValue(42.125)
}

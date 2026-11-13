package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("stdout before exit")
	fmt.Fprintln(os.Stderr, "stderr before exit")
	os.Exit(7)
}

package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.CreateTemp("", "llgo-stdout-")
	if err != nil {
		panic(err)
	}
	path := f.Name()
	defer os.Remove(path)
	if n, err := f.WriteString("disk"); n != len("disk") || err != nil {
		panic("file write failed")
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	if data, err := os.ReadFile(path); string(data) != "disk" || err != nil {
		panic("file write changed")
	}
	if n, err := fmt.Printf("console and file writes ok\n"); n != len("console and file writes ok\n") || err != nil {
		panic("console write failed")
	}
}

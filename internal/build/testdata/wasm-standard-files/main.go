package main

import (
	"fmt"
	"os"
)

func main() {
	if os.Stdin.Fd() != 0 || os.Stdout.Fd() != 1 || os.Stderr.Fd() != 2 {
		panic("incorrect standard files")
	}
	f, err := os.CreateTemp("", "llgo-stdout-")
	if err != nil {
		panic(err)
	}
	path := f.Name()
	defer os.Remove(path)

	stdout := os.Stdout
	os.Stdout = f
	n, err := fmt.Printf("captured %v", struct{ Destination string }{"stdout"})
	os.Stdout = stdout
	if n != len("captured {stdout}") || err != nil {
		panic("incorrect write result")
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "captured {stdout}" {
		panic("stdout was not redirected")
	}
	fmt.Println("standard files and redirected stdout ok")
}

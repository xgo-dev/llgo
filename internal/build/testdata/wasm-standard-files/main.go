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
	if n != len("captured {stdout}") || err != nil {
		panic("incorrect write result")
	}
	if n, err := printCaptured(" %s", "value"); n != len(" value") || err != nil {
		panic("incorrect redirected dynamic write result")
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
	if n, err := printCaptured(" %v", "closed"); n != 0 || err == nil {
		panic("incorrect redirected dynamic write error")
	}
	os.Stdout = stdout
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "captured {stdout} value" {
		panic("stdout was not redirected")
	}
	fmt.Println("standard files and redirected stdout ok")
}

//go:noinline
func printCaptured(format string, value any) (int, error) {
	return fmt.Printf(format, value)
}

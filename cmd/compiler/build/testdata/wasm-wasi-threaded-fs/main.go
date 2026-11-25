package main

import (
	"os"
	"path/filepath"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	wd, err := os.Getwd()
	must(err)
	if wd == "" {
		panic("empty working directory")
	}
	dir, err := os.MkdirTemp("", "llgo-wasi-threads-")
	must(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "value.txt")
	must(os.WriteFile(path, []byte("threaded filesystem"), 0o600))
	value, err := os.ReadFile(path)
	must(err)
	if string(value) != "threaded filesystem" {
		panic("filesystem round trip failed")
	}
	println("wasi threaded filesystem ok")
}

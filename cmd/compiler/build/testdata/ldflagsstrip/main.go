package main

import (
	"fmt"
	"path/filepath"
	"runtime"
)

//go:noinline
func caller() (string, string, bool) {
	pc, file, line, ok := runtime.Caller(0)
	fn := runtime.FuncForPC(pc)
	return fn.Name(), filepath.Base(file), ok && line > 0
}

func main() {
	name, file, ok := caller()
	fmt.Println(name, file, ok)
}

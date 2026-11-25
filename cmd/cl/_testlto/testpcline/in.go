package main

import (
	"io"
	"runtime/pprof"
)

func main() {
	if err := pprof.Lookup("goroutine").WriteTo(io.Discard, 0); err != nil {
		panic(err)
	}
	println(1)
}

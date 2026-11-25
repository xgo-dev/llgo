package main

import (
	"runtime"
	_ "unsafe"
)

//go:linkname usleep C.usleep
func usleep(useconds uint32) int32

func init() {
	go func() {
		// Keep a user goroutine alive while the initial thread calls Goexit.
		usleep(20_000)
		println("wasi goexit worker done")
	}()
	runtime.Goexit()
	println("runtime.Goexit returned")
}

func main() {}

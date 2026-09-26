package main

import (
	"runtime"
	"time"
)

func init() {
	go func() {
		time.Sleep(20 * time.Millisecond)
		println("wasi goexit timer worker done")
	}()
	runtime.Goexit()
}

func main() {}

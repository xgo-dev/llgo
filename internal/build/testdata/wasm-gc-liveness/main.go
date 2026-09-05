package main

import "runtime"

type payload struct {
	value uint64
}

//go:noinline
func keepActive() uint64 {
	value := &payload{value: 0x12345678}
	runtime.GC()
	return value.value
}

func main() {
	if keepActive() != 0x12345678 {
		panic("active GC root was not retained")
	}
	println("wasm gc liveness ok")
}

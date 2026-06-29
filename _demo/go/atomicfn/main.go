package main

import (
	"sync/atomic"
)

func main() {
	demo(atomic.AddInt32)
}

func demo(fn func(addr *int32, delta int32) (new int32)) {
	var a int32
	fn(&a, 1)
	println(a)
}

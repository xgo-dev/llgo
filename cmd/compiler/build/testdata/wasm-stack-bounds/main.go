package main

import "fmt"

// Keep each frame live across the recursive call. The worker needs over
// 128 KiB, independently of goroutine scheduling or host callbacks.
//
//go:noinline
func sumFrames(depth int) uint64 {
	var frame [256]uint64
	for i := range frame {
		frame[i] = uint64(depth*1000 + i)
	}
	var sum uint64
	if depth != 0 {
		sum = sumFrames(depth - 1)
	}
	for i, value := range frame {
		if value != uint64(depth*1000+i) {
			panic("stack frame corrupted")
		}
		sum += value
	}
	return sum
}

func main() {
	const depth = 96
	const want = 256*1000*depth*(depth+1)/2 + (depth+1)*255*256/2
	ch := make(chan uint64)
	go func() { ch <- sumFrames(depth) }()
	if got := <-ch; got != want {
		panic("recursive result mismatch")
	}
	fmt.Println("wasm stack bounds ok")
}

package main

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
)

type Pair struct{ Lo, Hi uint64 }

//go:noinline
func put(p *Pair, x uint64) { p.Hi = x }

//go:noinline
func pointerBranch(op int, x uint64) (r Pair) {
	if op == 0 {
		return Pair{Lo: x}
	}
	put(&r, x)
	return r
}

var saved *Pair

//go:noinline
func escaping(x uint64) (r Pair) {
	saved = &r
	put(&r, x)
	return
}

func deferred(x uint64) (r Pair) {
	defer put(&r, x)
	return
}

func main() {
	fmt.Println(pointerBranch(0, 17))
	fmt.Println(pointerBranch(1, 17))
	fmt.Println(deferred(23))
	var retained []*Pair
	for i := uint64(0); i < 32; i++ {
		escaping(i)
		retained = append(retained, saved)
	}
	runtime.GC()
	var sum uint64
	for _, p := range retained {
		sum += p.Hi
	}
	fmt.Println(sum)

	ctx, cancel := context.WithCancel(context.Background())
	var value atomic.Value
	fmt.Println(ctx.Err() == nil, value.Load() == nil)
	value.Store(42)
	fmt.Println(value.Load().(int))
	cancel()
	fmt.Println(ctx.Err() == context.Canceled)
}

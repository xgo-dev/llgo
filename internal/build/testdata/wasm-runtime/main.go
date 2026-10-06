package main

import (
	"runtime"
	"time"
)

func main() {
	exerciseDeferContinuations()
	exerciseSuspendedUnwind()
	println(runtime.GOOS)
}

func exerciseDeferContinuations() {
	if got := normalLoopDefers(); got != 321 {
		panic("normal loop defer order")
	}
	order, recovered := panicWhileDrainingLoopDefers()
	if order != 321 || recovered != "wasm-loop-defer-boom" {
		panic("panic loop defer order")
	}
}

func normalLoopDefers() (order int) {
	for i := 1; i <= 3; i++ {
		value := i
		defer func() { order = order*10 + value }()
	}
	return
}

func panicWhileDrainingLoopDefers() (order int, recovered any) {
	defer func() { recovered = recover() }()
	func() {
		for i := 1; i <= 3; i++ {
			value := i
			defer func() {
				order = order*10 + value
				if value == 2 {
					panic("wasm-loop-defer-boom")
				}
			}()
		}
	}()
	return
}

// Exercise native SjLj catch paths while Asyncify suspends a deferred call.
// Keep a heap payload live across collection and another goroutine's unwind.
func exerciseSuspendedUnwind() {
	done := make(chan int, 2)
	for id := 1; id <= 2; id++ {
		go func(id int) {
			payload := &[2]int{id, id * 7}
			defer func() {
				time.Sleep(time.Millisecond)
				runtime.GC()
				if recover() != payload || payload[1] != id*7 {
					panic("panic payload lost across suspended defer")
				}
				done <- id
			}()
			runtime.Gosched()
			panic(payload)
		}(id)
	}
	if a, b := <-done, <-done; a+b != 3 || a == b {
		panic("suspended panic completion")
	}
	go func() {
		defer func() {
			time.Sleep(time.Millisecond)
			runtime.GC()
			done <- 7
		}()
		runtime.Goexit()
	}()
	if <-done != 7 {
		panic("suspended Goexit defer")
	}
}

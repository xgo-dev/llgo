package main

import "runtime"

func main() {
	exerciseDeferContinuations()
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

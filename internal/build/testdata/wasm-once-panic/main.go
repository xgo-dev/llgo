package main

import "sync"

func recovered(fn func()) (value any) {
	defer func() { value = recover() }()
	fn()
	return
}

func main() {
	payload := &struct{ reason string }{"once"}
	starts := 0
	run := sync.OnceFunc(func() {
		starts++
		panic(payload)
	})
	if recovered(run) != payload || recovered(run) != payload || starts != 1 {
		panic("OnceFunc did not preserve the panic payload")
	}
	println("once panic preserved")
}

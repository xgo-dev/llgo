package main

import "runtime"

//go:noinline
func panicSite() {
	panic("shallow-panic")
}

func main() {
	_ = runtime.NumCPU()
	panicSite()
}

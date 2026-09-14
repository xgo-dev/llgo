//go:build !baremetal

package main

func checkGoCall() {
	order := 0
	arg := func(digit int) uintptr { order = order*10 + digit; return uintptr(order) }
	go multiply(arg(1), arg(2))
	if order != 12 {
		panic("goroutine multiply argument evaluation mismatch")
	}
}

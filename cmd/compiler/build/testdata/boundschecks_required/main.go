package main

import "unsafe"

func didPanic(f func()) (panicked bool) {
	defer func() {
		panicked = recover() != nil
	}()
	f()
	return
}

func main() {
	println(
		didPanic(func() {
			var array *[4]byte
			_ = array[0]
		}),
		didPanic(func() {
			var array *[4]byte
			_ = array[:]
		}),
		didPanic(func() {
			_ = unsafe.String((*byte)(nil), 1)
		}),
		didPanic(func() {
			_ = unsafe.Slice((*byte)(nil), 1)
		}),
	)
}

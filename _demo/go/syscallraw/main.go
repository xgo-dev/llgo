package main

import (
	"syscall"
	"unsafe"
)

func main() {
	msg := []byte("Hello from Syscall!\n")
	r1, r2, err := syscall.Syscall(
		syscall.SYS_WRITE,
		1,
		uintptr(unsafe.Pointer(&msg[0])),
		uintptr(len(msg)),
	)
	if r1 != 20 || r2 != 0 || err != 0 {
		panic("syscall error")
	}
}

package main

/*
void *packed_entry_address(void);
*/
import "C"

import (
	"runtime"
	"unsafe"
)

//go:linkname runtimeCgocall runtime.cgocall
func runtimeCgocall(fn, arg unsafe.Pointer) int32

type frame struct {
	left, right uint64
	result      uint64
	calls       uint64
}

func main() {
	entry := C.packed_entry_address()
	for i := uint64(0); i < 128; i++ {
		args := frame{left: i * 0x100000001, right: ^i}
		status := runtimeCgocall(entry, unsafe.Pointer(&args))
		if status != -37 || args.result != args.left^args.right || args.calls != 1 {
			panic("packed call lost arguments, mutation, or signed return")
		}
		runtime.KeepAlive(&args)
	}
	println("packed verified 128")
}

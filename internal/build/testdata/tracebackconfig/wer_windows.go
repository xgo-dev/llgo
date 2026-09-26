package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func probeWER() {
	mode, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetErrorMode").Call()
	var flags uint32
	_, _, _ = syscall.NewLazyDLL("kernel32.dll").NewProc("WerGetFlags").Call(^uintptr(0), uintptr(unsafe.Pointer(&flags)))
	fmt.Printf("disabled=%t no-ui=%t\n", mode&2 != 0, flags&0x20 != 0)
}

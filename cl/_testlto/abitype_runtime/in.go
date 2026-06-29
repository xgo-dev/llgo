package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

func main() {
	var v any = rune(0)
	t := (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "int32" {
		panic("abi rune error")
	}
	v = byte(0)
	t = (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "uint8" {
		panic("abi byte error")
	}
}

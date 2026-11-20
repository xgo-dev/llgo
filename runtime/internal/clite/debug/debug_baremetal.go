//go:build baremetal

package debug

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

type Info struct {
	Fname *c.Char
	Fbase c.Pointer
	Sname *c.Char
	Saddr c.Pointer
}

func Address() unsafe.Pointer {
	panic("not implemented")
}

func Addrinfo(addr unsafe.Pointer, info *Info) c.Int {
	panic("not implemented")
}

// Symbol resolves a C symbol name to its address. Baremetal images carry no
// runtime symbol table, so every lookup reports "not found".
func Symbol(name *c.Char) unsafe.Pointer {
	_ = name
	return nil
}

type Frame struct {
	PC     uintptr
	Offset uintptr
	SP     unsafe.Pointer
	Name   string
}

func StackTrace(skip int, fn func(fr *Frame) bool) {
	panic("not implemented")
}

func PrintStack(skip int) {
	panic("not implemented")

}

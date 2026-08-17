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

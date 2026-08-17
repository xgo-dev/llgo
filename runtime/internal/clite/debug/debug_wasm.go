//go:build wasm

package debug

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoFiles = "_wrap/debug_wasm.c"
)

type Info struct {
	Fname *c.Char
	Fbase c.Pointer
	Sname *c.Char
	Saddr c.Pointer
}

func Address() unsafe.Pointer {
	return nil
}

func Addrinfo(addr unsafe.Pointer, info *Info) c.Int {
	_, _ = addr, info
	return 0
}

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
	_, _ = skip, fn
}

func PrintStack(skip int) {
	print_stack(c.Int(skip + 4))
}

//go:linkname print_stack C.llgo_print_stack
func print_stack(skip c.Int)

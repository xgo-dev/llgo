//go:build wasm

package debug

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
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

type SymbolInfo struct {
	Function *c.Char
	File     *c.Char
	Line     c.Int
	Entry    c.Pointer
}

func Address() unsafe.Pointer {
	panic("not implemented")
}

func Addrinfo(addr unsafe.Pointer, info *Info) c.Int {
	panic("not implemented")
}

func Symbolize(addr unsafe.Pointer, info *SymbolInfo) c.Int {
	panic("not implemented")
}

func FreeSymbolInfo(info *SymbolInfo) {
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
	print_stack(c.Int(skip + 4))
}

//go:linkname print_stack C.llgo_print_stack
func print_stack(skip c.Int)

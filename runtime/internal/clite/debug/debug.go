//go:build !wasm && !baremetal

package debug

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

const (
	LLGoFiles = "$(llvm-config --cflags): _wrap/debug.c"
)

type Info struct {
	Fname *c.Char
	Fbase c.Pointer
	Sname *c.Char
	Saddr c.Pointer
}

//go:linkname Address C.llgo_address
func Address() unsafe.Pointer

//go:linkname Addrinfo C.llgo_addrinfo
func Addrinfo(addr unsafe.Pointer, info *Info) c.Int

//go:linkname Symbol C.llgo_symbol
func Symbol(name *c.Char) unsafe.Pointer

//go:linkname stacktrace C.llgo_stacktrace
func stacktrace(skip c.Int, ctx unsafe.Pointer, fn func(ctx, pc, offset, sp unsafe.Pointer, name *c.Char) c.Int)

type Frame struct {
	PC     uintptr
	Offset uintptr
	SP     unsafe.Pointer
	Name   string
}

func StackTrace(skip int, fn func(fr *Frame) bool) {
	stacktrace(c.Int(1+skip), unsafe.Pointer(&fn), func(ctx, pc, offset, sp unsafe.Pointer, name *c.Char) c.Int {
		fn := *(*func(fr *Frame) bool)(ctx)
		if !fn(&Frame{uintptr(pc), uintptr(offset), sp, c.GoString(name)}) {
			return 0
		}
		return 1
	})
}

func PrintStack(skip int) {
	StackTrace(skip+1, func(fr *Frame) bool {
		var info Info
		Addrinfo(unsafe.Pointer(fr.PC), &info)
		c.Fprintf(c.Stderr, c.Str("[0x%08X %s+0x%x, SP = 0x%x]\n"), fr.PC, fr.Name, fr.Offset, fr.SP)
		return true
	})
}

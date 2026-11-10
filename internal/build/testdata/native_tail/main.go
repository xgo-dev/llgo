package main

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

const LLGoFiles = "_wrap/probe.c"

func Enter(code, context, stack uintptr)
func Resume(code, context, stack uintptr)

//go:linkname guest C.native_tail_guest_addr
func guest() uintptr

//go:linkname resumeGuest C.native_tail_resume_addr
func resumeGuest() uintptr

//go:linkname check C.native_tail_check
func check(entry, code, context, stack uintptr) int

func main() {
	var context [5]uintptr
	stack := make([]byte, 65536)
	top := (uintptr(unsafe.Pointer(&stack[len(stack)-1])) + 1) &^ 15
	for i := uintptr(0); i < 1000; i++ {
		context[0] = i
		ptr := uintptr(unsafe.Pointer(&context[0]))
		if check(reflect.ValueOf(Enter).Pointer(), guest(), ptr, top) != 1 || context[1] != i+42 {
			panic("entry corrupted physical state")
		}
		// Direct calls also exercise LTO's inlining decisions around the
		// physical ABI boundary, independently of the C register checker.
		Enter(guest(), ptr, top)
		if context[1] != i+42 {
			panic("direct entry corrupted physical state")
		}
		// Simulate returning to Go for a host call, then resuming a guest stack.
		context[0] += 100
		if check(reflect.ValueOf(Resume).Pointer(), resumeGuest(), ptr, top) != 1 || context[1] != i+142 {
			panic("resume corrupted physical state")
		}
		Resume(resumeGuest(), ptr, top)
		if context[1] != i+142 {
			panic("direct resume corrupted physical state")
		}
	}
	runtime.KeepAlive(stack)
	fmt.Println("entry/resume/stack/registers: OK")
}

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

//go:linkname gcGuest C.native_tail_gc_guest_addr
func gcGuest() uintptr

//go:linkname waitEntered C.native_tail_wait_entered
func waitEntered(flag *uintptr)

//go:linkname releaseGuest C.native_tail_release_guest
func releaseGuest(flag *uintptr)

func main() {
	var context [7]uintptr
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
	// Force a collection while the native thread is executing on its guest
	// stack. The handshake prevents GC from completing before entry.
	done := make(chan struct{})
	go func() {
		waitEntered(&context[5])
		runtime.GC()
		releaseGuest(&context[6])
		close(done)
	}()
	context[0] = 700
	Enter(gcGuest(), uintptr(unsafe.Pointer(&context[0])), top)
	<-done
	if context[1] != 742 {
		panic("collection during guest execution corrupted state")
	}
	runtime.KeepAlive(stack)
	fmt.Println("entry/resume/stack/registers/gc: OK")
}

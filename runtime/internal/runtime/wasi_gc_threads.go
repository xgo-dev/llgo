//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

//export llgo_wasi_gc_mstart
func wasiGCMStart(arg unsafe.Pointer) unsafe.Pointer { return mstart(arg) }

//go:linkname wasiGCThreadStart C.llgo_wasi_gc_thread_start
func wasiGCThreadStart(arg unsafe.Pointer) unsafe.Pointer

func registerWasiGCThread() {
	if gcroot.CurrentThreadRegistered() {
		return
	}
	wasiGCEnterBegin()
	gcroot.RegisterCurrentThread()
	wasiGCEnterEnd()
}

func unregisterWasiGCThread() {
	if !gcroot.CurrentThreadRegistered() {
		return
	}
	wasiGCLeaveBegin()
	gcroot.UnregisterCurrentThread()
	wasiGCLeaveEnd()
}

// A C call that cannot acknowledge a safepoint causes this GC cycle to be
// skipped. The collector never scans a stack that is still running C code.
func wasiGCStopTheWorld() bool { return wasiGCStop() != 0 }

func wasiGCResumeWorld() { wasiGCResume() }

func wasiGCSafepoint() {
	if !gcroot.CurrentThreadRegistered() || wasiGCPending() == 0 {
		return
	}
	bottom := uintptr(wasiGCStackPointer())
	top := wasiGCStackTop()
	wasiGCPark(uintptr(gcroot.CurrentChain()), bottom, top)
}

//go:linkname wasiGCEnterBegin C.llgo_wasi_gc_enter_begin
func wasiGCEnterBegin()

//go:linkname wasiGCEnterEnd C.llgo_wasi_gc_enter_end
func wasiGCEnterEnd()

//go:linkname wasiGCLeaveBegin C.llgo_wasi_gc_leave_begin
func wasiGCLeaveBegin()

//go:linkname wasiGCLeaveEnd C.llgo_wasi_gc_leave_end
func wasiGCLeaveEnd()

//go:linkname wasiGCPending C.llgo_wasi_gc_pending
func wasiGCPending() c.Int

//go:linkname wasiGCPark C.llgo_wasi_gc_park
func wasiGCPark(chain, bottom, top uintptr)

//go:linkname wasiGCStop C.llgo_wasi_gc_stop
func wasiGCStop() c.Int

//go:linkname wasiGCResume C.llgo_wasi_gc_resume
func wasiGCResume()

//go:linkname wasiGCStackPointer llgo.stackSave
func wasiGCStackPointer() unsafe.Pointer

//go:linkname wasiGCStackTop C.llgo_gc_stack_top
func wasiGCStackTop() uintptr

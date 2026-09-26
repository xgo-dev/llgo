//go:build !baremetal && !wasm

// Package stacktrace owns native thread registrations and raw stack snapshots.
// No Go code runs in its asynchronous capture handler.
package stacktrace

import "unsafe"

const LLGoPackage = "link"

//go:linkname Register C.llgo_traceback_register
func Register(id, parent uint64, created uintptr) unsafe.Pointer

//go:linkname Attach C.llgo_traceback_attach
func Attach(node unsafe.Pointer)

//go:linkname Unregister C.llgo_traceback_unregister
func Unregister(node unsafe.Pointer)

//go:linkname SetState C.llgo_traceback_state
func SetState(node unsafe.Pointer, state uint32)

// Snapshot is opaque: Windows 386 and Go need not align uint64 identically.
type Snapshot struct{}

//go:linkname Next C.llgo_traceback_next
func Next(*Snapshot) *Snapshot

//go:linkname Info C.llgo_traceback_info
func Info(s *Snapshot, id, parent *uint64, created, count *uintptr, state *uint32) *uintptr

//go:linkname Capture C.llgo_traceback_capture
func Capture(except uint64) *Snapshot

//go:linkname Free C.llgo_traceback_free
func Free(snapshots *Snapshot)

//go:linkname FaultBuffer C.llgo_traceback_fault_buffer
func FaultBuffer() *uintptr

//go:linkname ReleaseFaultBuffer C.llgo_traceback_release_fault_buffer
func ReleaseFaultBuffer()

// A bounded native scratch buffer also limits per-thread address space use.
// Keep this in sync with LLGO_TRACEBACK_MAX in _wrap/traceback.h.
const MaxFrames = 1 << 16

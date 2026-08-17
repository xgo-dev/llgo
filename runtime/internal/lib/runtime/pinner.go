package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// A Pinner is a set of Go objects each pinned to a fixed location in memory. The
// [Pinner.Pin] method pins one object, while [Pinner.Unpin] unpins all pinned
// objects. See their comments for more information.
type Pinner struct {
	*pinner
}

// Pin pins a Go object, preventing it from being moved or freed by the garbage
// collector until the [Pinner.Unpin] method has been called.
//
// A pointer to a pinned object can be directly stored in C memory or can be
// contained in Go memory passed to C functions. If the pinned object itself
// contains pointers to Go objects, these objects must be pinned separately if they
// are going to be accessed from C code.
//
// The argument must be a pointer of any type or an [unsafe.Pointer].
// It's safe to call Pin on non-Go pointers, in which case Pin will do nothing.
func (p *Pinner) Pin(pointer any) {
	if p.pinner == nil {
		p.pinner = new(pinner)
		p.refs = p.refStore[:0]

		// We set this finalizer once and never clear it. Thus, if the
		// pinner gets cached, we'll reuse it, along with its finalizer.
		// This lets us avoid the relatively expensive SetFinalizer call
		// when reusing from the cache. The finalizer however has to be
		// resilient to an empty pinner being finalized, which is done
		// by checking p.refs' length.
		SetFinalizer(p.pinner, func(i *pinner) {
			if len(i.refs) != 0 {
				i.unpin() // only required to make the test idempotent
				pinnerLeakPanic()
			}
		})
	}
	ptr := pinnerGetPtr(&pointer)
	p.refs = append(p.refs, ptr)
}

// Unpin unpins all pinned objects of the [Pinner].
func (p *Pinner) Unpin() {
	p.pinner.unpin()
}

const (
	pinnerSize         = 64
	pinnerRefStoreSize = (pinnerSize - unsafe.Sizeof([]unsafe.Pointer{})) / unsafe.Sizeof(unsafe.Pointer(nil))
)

type pinner struct {
	refs     []unsafe.Pointer
	refStore [pinnerRefStoreSize]unsafe.Pointer
}

func (p *pinner) unpin() {
	if p == nil || p.refs == nil {
		return
	}
	// The following two lines make all pointers to references
	// in p.refs unreachable, either by deleting them or dropping
	// p.refs' backing store (if it was not backed by refStore).
	p.refStore = [pinnerRefStoreSize]unsafe.Pointer{}
	p.refs = p.refStore[:0]
}

func pinnerGetPtr(i *any) unsafe.Pointer {
	e := (*eface)(unsafe.Pointer(i))
	etyp := e._type
	if etyp == nil {
		runtime.PanicErrorString("runtime.Pinner: argument is nil")
	}
	if kind := etyp.Kind(); kind != abi.Pointer && kind != abi.UnsafePointer {
		runtime.PanicErrorString("runtime.Pinner: argument is not a pointer: " + etyp.String())
	}
	return e.data
}

// to be able to test that the GC panics when a pinned pointer is leaking, this
// panic function is a variable, that can be overwritten by a test.
var pinnerLeakPanic = func() {
	runtime.PanicErrorString("runtime.Pinner: found leaking pinned pointer; forgot to call Unpin()?")
}

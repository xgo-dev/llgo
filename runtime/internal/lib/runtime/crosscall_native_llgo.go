//go:build (darwin || linux) && !baremetal && !wasm

package runtime

import (
	"unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// A crosscall2 entry is the code address of a noncapturing Go wrapper with one
// frame pointer. Any managed captures belong to that wrapper's dispatch state,
// not to an environment inferred from this raw address.
//
//llgo:cdecl
type packedCallbackEntry func(unsafe.Pointer)

// crosscall2 preserves the public C entry used by native callback trampolines.
// LLGo uses the host ABI for the wrapper, so no gc register/stack transition is
// needed. size is obsolete; context is gc's traceback token, not a closure env.
// Runtime-owned exports bypass the ordinary executable's generated C wrappers,
// so register a foreign thread here before calling Go or allocating a defer.
//
//export crosscall2
func crosscall2(fn, frame unsafe.Pointer, _ int32, _ uintptr) {
	registered := llruntime.EnterForeignThread()
	defer llruntime.ExitForeignThread(registered)
	if fn == nil {
		panic("runtime: nil packed callback entry")
	}
	entry := *(*packedCallbackEntry)(unsafe.Pointer(&fn))
	entry(frame)
	KeepAlive(frame)
}

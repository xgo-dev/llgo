// Package purego implements the explicit Linux/amd64 purego compatibility
// backend. Importing it does not change Go function values or runtime.cgocall.
package purego

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/ffi"
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Purego v0.9.1 promises at least 2000 callbacks, retained for process lifetime.
// The bounded table roots the actual Go closure, including its captured data,
// not only its executable address. Failed creations never occupy a slot.
const callbackLimit = 2000

type callbackEntry struct {
	fn       reflect.Value
	sig      *ffi.Signature
	closure  *ffi.Closure
	args     []*ffi.Type
	omitHead bool
}

var callbacks struct {
	once    sync.Once
	mu      sync.Mutex
	entries [callbackLimit]*callbackEntry
	count   int
}

// NewCallback returns a C function pointer for fn. A successful callback stays
// valid for the process lifetime, as required by purego.NewCallback. This is
// not a general-purpose conversion from a Go function value to a C address.
//
// omitCDecl is set only by the version-checked source adapter after checking
// purego.CDecl's identity. It omits that leading zero-sized marker from the C
// signature, while supplying its zero value when invoking the Go function.
// Callbacks accept scalar and pointer arguments and zero or one integer or
// pointer result. Unsupported signatures panic before allocating executable
// storage or retaining any callback roots.
func NewCallback(fn any, omitCDecl bool) uintptr {
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func || value.IsNil() {
		panic("llgo purego: callback must be a non-nil function")
	}
	typ := value.Type()
	if typ.IsVariadic() || typ.NumOut() > 1 {
		panic("llgo purego: callback must be non-variadic with at most one result")
	}
	first := 0
	if omitCDecl {
		if typ.NumIn() == 0 || typ.In(0).Kind() != reflect.Struct || typ.In(0).Size() != 0 {
			panic("llgo purego: invalid leading CDecl marker")
		}
		first = 1
	}
	if typ.NumIn()-first > 64 {
		panic("llgo purego: callback argument frame exceeds 64 words")
	}
	args := make([]*ffi.Type, typ.NumIn()-first)
	for i := range args {
		args[i] = callbackType(typ.In(i+first), true)
	}
	result := ffi.TypeVoid
	if typ.NumOut() != 0 {
		result = callbackType(typ.Out(0), false)
	}

	callbacks.once.Do(llruntime.EnableForeignThreadRegistration)
	callbacks.mu.Lock()
	defer callbacks.mu.Unlock()
	if callbacks.count == callbackLimit {
		panic("llgo purego: the maximum number of callbacks has been reached")
	}
	sig, err := ffi.NewSignature(result, args...)
	if err != nil {
		panic("llgo purego: callback signature: " + err.Error())
	}
	entry := &callbackEntry{fn: value, sig: sig, args: args, omitHead: omitCDecl}
	entry.closure = ffi.NewClosure()
	if entry.closure.Fn == nil {
		entry.closure.Free()
		panic("llgo purego: callback executable allocation failed")
	}
	if err := entry.closure.Bind(sig, callCallback, unsafe.Pointer(entry)); err != nil {
		entry.closure.Free()
		panic("llgo purego: callback binding: " + err.Error())
	}
	callbacks.entries[callbacks.count] = entry
	callbacks.count++
	return uintptr(entry.closure.Fn)
}

func callbackType(typ reflect.Type, argument bool) *ffi.Type {
	switch typ.Kind() {
	case reflect.Bool:
		return ffi.TypeBool
	case reflect.Int:
		return ffi.TypeInt
	case reflect.Int8:
		return ffi.TypeInt8
	case reflect.Int16:
		return ffi.TypeInt16
	case reflect.Int32:
		return ffi.TypeInt32
	case reflect.Int64:
		return ffi.TypeInt64
	case reflect.Uint:
		return ffi.TypeUint
	case reflect.Uint8:
		return ffi.TypeUint8
	case reflect.Uint16:
		return ffi.TypeUint16
	case reflect.Uint32:
		return ffi.TypeUint32
	case reflect.Uint64:
		return ffi.TypeUint64
	case reflect.Uintptr:
		return ffi.TypeUintptr
	case reflect.Pointer, reflect.UnsafePointer:
		return ffi.TypePointer
	case reflect.Float32:
		if argument {
			return ffi.TypeFloat32
		}
	case reflect.Float64:
		if argument {
			return ffi.TypeFloat64
		}
	}
	role := "result"
	if argument {
		role = "argument"
	}
	panic("llgo purego: unsupported callback " + role + " type " + typ.String())
}

func callCallback(_ *ffi.Signature, ret unsafe.Pointer, args *unsafe.Pointer, userdata unsafe.Pointer) {
	// Registration must precede reflection, slice allocation, or reading Go
	// closure state. A newly adopted thread retains its G/collector registration
	// until TLS teardown; a nested callback reuses the existing registration.
	registered := llruntime.EnterForeignThread()
	entry := (*callbackEntry)(userdata)
	typ := entry.fn.Type()
	in := make([]reflect.Value, typ.NumIn())
	first := 0
	if entry.omitHead {
		in[0] = reflect.Zero(typ.In(0))
		first = 1
	}
	for i := first; i < len(in); i++ {
		// Do not let a callback retain a Value backed by libffi's temporary
		// native argument frame. Detach each scalar before calling Go.
		in[i] = reflect.New(typ.In(i)).Elem()
		in[i].Set(reflect.NewAt(typ.In(i), ffi.Index(args, uintptr(i-first))).Elem())
	}
	out := entry.fn.Call(in)
	if len(out) != 0 {
		storeCallbackResult(ret, out[0])
	}
	runtime.KeepAlive(in)
	runtime.KeepAlive(entry)
	// As with the existing native Windows callback bridge, keep cleanup off
	// the defer chain spanning libffi. Goexit uses the retained G lifecycle.
	llruntime.ExitForeignThread(registered)
}

func storeCallbackResult(ret unsafe.Pointer, value reflect.Value) {
	// libffi's closure return buffer uses ffi_arg width for integer returns,
	// including narrow signed values. Never copy only the low byte/halfword.
	var result uintptr
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		result = uintptr(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		result = uintptr(value.Uint())
	case reflect.Pointer, reflect.UnsafePointer:
		result = value.Pointer()
	case reflect.Bool:
		if value.Bool() {
			result = 1
		}
	default:
		panic("llgo purego: invalid validated callback result")
	}
	*(*uintptr)(ret) = result
}

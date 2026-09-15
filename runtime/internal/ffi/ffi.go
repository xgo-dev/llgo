package ffi

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/ffi"
)

type Type = ffi.Type

type Signature = ffi.Cif

type ABI = c.Uint

const DefaultABI ABI = ffi.DefaultAbi

type Error int

func (s Error) Error() string {
	switch s {
	case ffi.OK:
		return "ok"
	case ffi.BAD_TYPEDEF:
		return "bad type def"
	case ffi.BAD_ABI:
		return "bad ABI"
	case ffi.BAD_ARGTYPE:
		return "bad argument type"
	}
	return "invalid status"
}

func NewSignature(ret *Type, args ...*Type) (*Signature, error) {
	return NewSignatureWithABI(DefaultABI, ret, args...)
}

// NewSignatureWithABI prepares a native call signature using abi. Most calls
// should use NewSignature; the explicit form is needed for platforms such as
// windows/386 that expose more than one C calling convention.
func NewSignatureWithABI(abi ABI, ret *Type, args ...*Type) (*Signature, error) {
	cif, atype := newSignatureStorage(ret, args)
	status := ffi.PrepCif(cif, abi, c.Uint(len(args)), ret, atype)
	if status == 0 {
		return cif, nil
	}
	return nil, Error(status)
}

func NewSignatureVar(ret *Type, fixed int, args ...*Type) (*Signature, error) {
	cif, atype := newSignatureStorage(ret, args)
	status := ffi.PrepCifVar(cif, DefaultABI, c.Uint(fixed), c.Uint(len(args)), ret, atype)
	if status == ffi.OK {
		return cif, nil
	}
	return nil, Error(status)
}

func Call(cif *Signature, fn unsafe.Pointer, ret unsafe.Pointer, args ...unsafe.Pointer) {
	ffi.Call(cif, fn, ret, valuePointerArray(args))
}

type Closure struct {
	ptr unsafe.Pointer
	Fn  unsafe.Pointer
}

func NewClosure() *Closure {
	c := &Closure{}
	c.ptr = ffi.ClosureAlloc(&c.Fn)
	return c
}

func (c *Closure) Free() {
	if c != nil && c.ptr != nil {
		ffi.ClosureFree(c.ptr)
		c.ptr = nil
	}
}

func (c *Closure) Bind(cif *Signature, fn ffi.ClosureFunc, userdata unsafe.Pointer) error {
	status := ffi.PreClosureLoc(c.ptr, cif, fn, userdata, c.Fn)
	if status == ffi.OK {
		return nil
	}
	return Error(status)
}

func Index(args *unsafe.Pointer, i uintptr) unsafe.Pointer {
	return ffi.Index(args, i)
}

// TypeElement returns one entry from a null-terminated aggregate element
// array. The array follows the physical pointer width used by libffi.
func TypeElement(aggregate *Type, i uintptr) *Type {
	if aggregate == nil || aggregate.Elements == nil {
		return nil
	}
	return (*Type)(ffi.Index((*unsafe.Pointer)(unsafe.Pointer(aggregate.Elements)), i))
}

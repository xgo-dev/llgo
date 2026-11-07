package llvmintrinsic

// #cgo CXXFLAGS: -std=c++17
// #include "native.h"
// #include <stdlib.h>
// #include <llvm-c/Core.h>
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

const (
	scalableVectorKind = llvm.TypeKind(C.LLVMScalableVectorTypeKind)
	halfKind           = llvm.TypeKind(C.LLVMHalfTypeKind)
	bfloatKind         = llvm.TypeKind(C.LLVMBFloatTypeKind)
)

// Fail at compile time if the upstream binding handle representation changes.
var (
	_ [unsafe.Sizeof(unsafe.Pointer(nil))]byte = [unsafe.Sizeof(llvm.Type{})]byte{}
	_ [unsafe.Sizeof(unsafe.Pointer(nil))]byte = [unsafe.Sizeof(llvm.Value{})]byte{}
)

// The public LLVM bindings wrap one C handle; copy that handle across cgo's
// package-local typedefs without introducing an alternate LLVM type system.
func wrapType(raw unsafe.Pointer) llvm.Type   { return *(*llvm.Type)(unsafe.Pointer(&raw)) }
func wrapValue(raw unsafe.Pointer) llvm.Value { return *(*llvm.Value)(unsafe.Pointer(&raw)) }

func resolveNative(id int, signature llvm.Type, mod llvm.Module) (typ llvm.Type, decl llvm.Value, name string, immargs []int, err error) {
	r := C.llgo_intrinsic_resolve(C.uint(id), unsafe.Pointer(signature.C), unsafe.Pointer(mod.C))
	defer C.llgo_intrinsic_dispose(r)
	if r.error != nil {
		err = fmt.Errorf("%s", C.GoString(r.error))
		return
	}
	typ, decl, name = wrapType(r._type), wrapValue(r.declaration), C.GoString(r.name)
	for _, index := range unsafe.Slice(r.immargs, int(r.immarg_count)) {
		immargs = append(immargs, int(index))
	}
	return
}

func arrayLength(t llvm.Type) uint64 {
	return uint64(C.llgo_intrinsic_array_length(unsafe.Pointer(t.C)))
}

func scalarBits(t llvm.Type) uint64 {
	return uint64(C.llgo_intrinsic_scalar_bits(unsafe.Pointer(t.C)))
}

func vectorShape(t llvm.Type) (count uint64, scalable bool, ok bool) {
	if t.IsNil() {
		return 0, false, false
	}
	var minimum C.uint
	var scale C.int
	ok = C.llgo_intrinsic_vector_shape(unsafe.Pointer(t.C), &minimum, &scale) != 0
	return uint64(minimum), scale != 0, ok
}

func vectorType(element llvm.Type, count uint64, scalable bool) llvm.Type {
	var scale C.int
	if scalable {
		scale = 1
	}
	return wrapType(C.llgo_intrinsic_vector_type(unsafe.Pointer(element.C), C.uint(count), scale))
}

func integerInRange(value llvm.Value, low, high uint64, unsigned bool) bool {
	var mode C.int
	if unsigned {
		mode = 1
	}
	return C.llgo_intrinsic_integer_range(unsafe.Pointer(value.C), C.uint64_t(low), C.uint64_t(high), mode) != 0
}

// PointerIndexBits returns the index width for a pointer or pointer vector in a
// target data layout. It is independent of the Go word size and pointer size.
func PointerIndexBits(t llvm.Type, layout string) (uint64, error) {
	if t.IsNil() {
		return 0, fmt.Errorf("nil pointer type")
	}
	c := C.CString(layout)
	defer C.free(unsafe.Pointer(c))
	bits := uint64(C.llgo_intrinsic_pointer_index_bits(unsafe.Pointer(t.C), c))
	if bits == 0 {
		return 0, fmt.Errorf("invalid data layout or non-pointer type %s", t)
	}
	return bits, nil
}

type descriptor struct {
	kind                string
	argument, domain, n int
}

func intrinsicCount() int         { return int(C.llgo_intrinsic_count()) }
func targetIntrinsic(id int) bool { return C.llgo_intrinsic_is_target(C.uint(id)) != 0 }
func baseName(id int) string {
	var n C.size_t
	p := C.llgo_intrinsic_base_name(C.uint(id), &n)
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), int(n)))
}

func descriptors(id int) []descriptor {
	n := int(C.llgo_intrinsic_descriptors(C.uint(id), nil))
	if n == 0 {
		return nil
	}
	raw := make([]C.LLGOIntrinsicDescriptor, n)
	C.llgo_intrinsic_descriptors(C.uint(id), &raw[0])
	out := make([]descriptor, n)
	for i, d := range raw {
		out[i] = descriptor{C.GoString(d.kind), int(d.argument), int(d.argument_kind), int(d.extra)}
	}
	return out
}

func sampleSignature(ctx llvm.Context, id, variant int) llvm.Type {
	return wrapType(C.llgo_intrinsic_sample(unsafe.Pointer(ctx.C), C.uint(id), C.uint(variant)))
}

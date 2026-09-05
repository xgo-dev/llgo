package ssa

/*
#include <stdint.h>

// These two public LLVM-C APIs bridge the ConstantRange constructor missing
// from github.com/xgo-dev/llvm v0.9.9. Opaque declarations avoid duplicating
// the binding package's platform-specific LLVM header search paths. Linking
// still uses that package's selected LLVM library.
typedef struct LLVMOpaqueContext *LLVMContextRef;
typedef struct LLVMOpaqueValue *LLVMValueRef;
typedef struct LLVMOpaqueAttributeRef *LLVMAttributeRef;
extern LLVMAttributeRef LLVMCreateConstantRangeAttribute(
    LLVMContextRef, unsigned, unsigned, const uint64_t *, const uint64_t *);
extern void LLVMAddAttributeAtIndex(LLVMValueRef, unsigned, LLVMAttributeRef);

static void llgoAddNonNegativeReturnRange(void *ctx, void *fn,
                                        unsigned kind, unsigned bits) {
    uint64_t lower = 0;
    uint64_t upper = UINT64_C(1) << (bits - 1);
    LLVMAttributeRef attr = LLVMCreateConstantRangeAttribute(
        (LLVMContextRef)ctx, kind, bits, &lower, &upper);
    LLVMAddAttributeAtIndex((LLVMValueRef)fn, 0, attr);
}
*/
import "C"

import (
	"unsafe"

	"github.com/xgo-dev/llvm"
)

func addNonNegativeReturnRange(ctx llvm.Context, fn llvm.Value, bits int) {
	if bits != 32 && bits != 64 {
		panic("ssa: unsupported runtime integer width")
	}
	C.llgoAddNonNegativeReturnRange(unsafe.Pointer(ctx.C), unsafe.Pointer(fn.C),
		C.unsigned(llvm.AttributeKindID("range")), C.unsigned(bits))
}

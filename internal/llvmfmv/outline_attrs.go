package llvmfmv

/*
#include <stdlib.h>

// The Go binding exposes individual call-site attribute lookups, but has
// no enumerator. Use LLVM's stable C API so type and string attributes,
// including target-specific ABI attributes, survive an invoke-to-call copy.
extern unsigned LLVMGetNumOperandBundles(void *);
extern unsigned LLVMGetCallSiteAttributeCount(void *, unsigned);
extern void LLVMGetCallSiteAttributes(void *, unsigned, void **);
extern void LLVMAddCallSiteAttribute(void *, unsigned, void *);
static int hasOutlineAttrs(void *src, int nargs) {
	for (int index = -1; index <= nargs; index++)
		if (LLVMGetCallSiteAttributeCount(src, (unsigned)index)) return 1;
	return 0;
}

static void copyOutlineAttrs(void *src, void *dst, int nargs) {
	for (int index = -1; index <= nargs; index++) {
		unsigned count = LLVMGetCallSiteAttributeCount(src, (unsigned)index);
		if (!count) continue;
		void **attrs = malloc(sizeof(void *) * count);
		LLVMGetCallSiteAttributes(src, (unsigned)index, attrs);
		for (unsigned i = 0; i < count; i++)
			LLVMAddCallSiteAttribute(dst, (unsigned)index, attrs[i]);
		free(attrs);
	}
}
*/
import "C"

import (
	"unsafe"

	"github.com/xgo-dev/llvm"
)

func copyOutlineCallAttrs(src, dst llvm.Value) {
	C.copyOutlineAttrs(unsafe.Pointer(src.C), unsafe.Pointer(dst.C), C.int(src.ArgOperandsCount()))
}

func hasOutlineAttrs(src llvm.Value) bool {
	return C.hasOutlineAttrs(unsafe.Pointer(src.C), C.int(src.ArgOperandsCount())) != 0
}

func hasOutlineBundles(src llvm.Value) bool {
	return C.LLVMGetNumOperandBundles(unsafe.Pointer(src.C)) != 0
}

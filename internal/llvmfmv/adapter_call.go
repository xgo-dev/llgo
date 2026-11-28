package llvmfmv

/*
#include <stdlib.h>
extern unsigned LLVMGetNumOperandBundles(void *);
extern void *LLVMGetOperandBundleAtIndex(void *, unsigned);
extern const char *LLVMGetOperandBundleTag(void *, size_t *);
extern void LLVMDisposeOperandBundle(void *);
extern unsigned LLVMGetCallSiteAttributeCount(void *, unsigned);
extern void LLVMGetCallSiteAttributes(void *, unsigned, void **);
extern void *LLVMBuildCallWithOperandBundles(void *, void *, void *, void **, unsigned, void **, unsigned, const char *);
extern void *LLVMBuildInvokeWithOperandBundles(void *, void *, void *, void **, unsigned, void *, void *, void **, unsigned, const char *);
extern void *LLVMInstructionGetAllMetadataOtherThanDebugLoc(void *, size_t *);
extern unsigned LLVMValueMetadataEntriesGetKind(void *, unsigned);
extern void *LLVMValueMetadataEntriesGetMetadata(void *, unsigned);
extern void LLVMDisposeValueMetadataEntries(void *);
extern void LLVMSetMetadata2(void *, unsigned, void *);
extern int LLVMCanValueUseFastMathFlags(void *);
extern unsigned LLVMGetFastMathFlags(void *);
extern void LLVMSetFastMathFlags(void *, unsigned);

// LLVM's C API can read a PHI predecessor, but cannot change it in place.
static void copyPhiProperties(void *src, void *dst) {
	size_t n;
	void *md = LLVMInstructionGetAllMetadataOtherThanDebugLoc(src, &n);
	for (unsigned i = 0; i < n; i++)
		LLVMSetMetadata2(dst, LLVMValueMetadataEntriesGetKind(md, i), LLVMValueMetadataEntriesGetMetadata(md, i));
	LLVMDisposeValueMetadataEntries(md);
	if (LLVMCanValueUseFastMathFlags(src))
		LLVMSetFastMathFlags(dst, LLVMGetFastMathFlags(src));
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/xgo-dev/llvm"
)

// LLVM uniques attributes in each context. Preserve their parameter indexes
// in the key: two calls with the same IR type may differ in nest, sret, etc.
func adapterAttributeKey(call llvm.Value) string {
	var key strings.Builder
	for index := -1; index <= call.ArgOperandsCount(); index++ {
		count := C.LLVMGetCallSiteAttributeCount(unsafe.Pointer(call.C), C.unsigned(index))
		if count != 0 {
			attrs := make([]unsafe.Pointer, count)
			C.LLVMGetCallSiteAttributes(unsafe.Pointer(call.C), C.unsigned(index), &attrs[0])
			fmt.Fprintf(&key, "%d:%p;", index, attrs[0])
			for _, attr := range attrs[1:] {
				fmt.Fprintf(&key, "%p;", attr)
			}
		}
	}
	return key.String()
}

// A musttail call cannot gain a frame; inalloca/preallocated storage belongs
// to a specific outgoing call frame. None is emitted by a normal Go wide call.
func validateAdapterCall(call llvm.Value) error {
	if !call.IsACallInst().IsNil() && call.TailCallKind() == llvm.TailCallKindMustTail {
		return fmt.Errorf("cannot adapt musttail across register widths")
	}
	returnsTwice := llvm.AttributeKindID("returns_twice")
	callee := call.CalledValue().IsAFunction()
	if !call.GetCallSiteEnumAttribute(-1, returnsTwice).IsNil() ||
		(!callee.IsNil() && !callee.GetEnumFunctionAttribute(returnsTwice).IsNil()) {
		return fmt.Errorf("cannot adapt returns_twice across a call frame")
	}
	for i := 1; i <= call.ArgOperandsCount(); i++ {
		for _, name := range []string{"inalloca", "preallocated"} {
			if !call.GetCallSiteEnumAttribute(i, llvm.AttributeKindID(name)).IsNil() {
				return fmt.Errorf("cannot adapt %s across a call frame", name)
			}
		}
	}
	bundles := adapterBundles(call)
	defer disposeAdapterBundles(bundles)
	for _, bundle := range bundles {
		var n C.size_t
		tag := C.LLVMGetOperandBundleTag(bundle, &n)
		switch name := C.GoStringN(tag, C.int(n)); name {
		case "deopt", "funclet":
		default:
			return fmt.Errorf("cannot move %q operand bundle across a call frame", name)
		}
	}
	return nil
}

// Funclet tokens and deoptimization continuations belong to the original
// caller. Keep those bundles on its replacement call/invoke.
func adapterBundles(call llvm.Value) []unsafe.Pointer {
	bundles := make([]unsafe.Pointer, C.LLVMGetNumOperandBundles(unsafe.Pointer(call.C)))
	for i := range bundles {
		bundles[i] = C.LLVMGetOperandBundleAtIndex(unsafe.Pointer(call.C), C.unsigned(i))
	}
	return bundles
}

func copyAdapterPHIProperties(src, dst llvm.Value) {
	dst.InstructionSetDebugLoc(src.InstructionDebugLoc())
	C.copyPhiProperties(unsafe.Pointer(src.C), unsafe.Pointer(dst.C))
}

func disposeAdapterBundles(bundles []unsafe.Pointer) {
	for _, bundle := range bundles {
		C.LLVMDisposeOperandBundle(bundle)
	}
}

func adapterOuterCall(b llvm.Builder, fn llvm.Value, args []llvm.Value, normal, unwind llvm.BasicBlock, bundles []unsafe.Pointer) llvm.Value {
	ft := fn.GlobalValueType()
	if len(bundles) == 0 {
		if normal.IsNil() {
			return b.CreateCall(ft, fn, args, "")
		}
		return b.CreateInvoke(ft, fn, args, normal, unwind, "")
	}
	argv := (*unsafe.Pointer)(unsafe.Pointer(&args[0].C))
	var result unsafe.Pointer
	empty := C.CString("")
	defer C.free(unsafe.Pointer(empty))
	if normal.IsNil() {
		result = C.LLVMBuildCallWithOperandBundles(unsafe.Pointer(b.C), unsafe.Pointer(ft.C), unsafe.Pointer(fn.C),
			argv, C.unsigned(len(args)), &bundles[0], C.unsigned(len(bundles)), empty)
	} else {
		result = C.LLVMBuildInvokeWithOperandBundles(unsafe.Pointer(b.C), unsafe.Pointer(ft.C), unsafe.Pointer(fn.C),
			argv, C.unsigned(len(args)), unsafe.Pointer(normal.C), unsafe.Pointer(unwind.C), &bundles[0], C.unsigned(len(bundles)), empty)
	}
	return *(*llvm.Value)(unsafe.Pointer(&result))
}

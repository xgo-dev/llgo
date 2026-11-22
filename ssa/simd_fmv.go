package ssa

import (
	"github.com/xgo-dev/llgo/internal/llvmfmv"
	"github.com/xgo-dev/llvm"
)

// MarkSIMDFMV records compiler-checked source identities for the early LLVM
// transform. An available variant is a cross-package ABI promise: the defining
// package emits it, and guarded callers may refer to it without LTO.
func (f Function) MarkSIMDFMV(available bool, query string) {
	if f.Prog.Target().GOARCH != "amd64" {
		return
	}
	if available {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute(llvmfmv.EntryAttribute, ""))
	}
	if query != "" {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute(llvmfmv.QueryAttribute, query))
	}
}

// AllowSIMDDispatcherInlining separates a compiler-required physical source
// frame from the synthetic FMV entry. The implementation retains its frame;
// the tail-only dispatcher may inline without changing the source call stack.
// Callers must not set this for an explicit noinline directive or -l.
func (f Function) AllowSIMDDispatcherInlining() {
	if f.Prog.Target().GOARCH == "amd64" {
		f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute(llvmfmv.InlineEntryAttribute, ""))
	}
}

// MarkMidwayWidth records only the instruction set guaranteed by the
// portable width selector. In particular simd=+512 does not imply VBMI2,
// VNNI or carryless multiplication. The original entry stays at baseline.
func (f Function) MarkMidwayWidth(width int) {
	if f.Prog.Target().GOARCH != "amd64" {
		return
	}
	features := "+avx,+avx2"
	switch width {
	case 256:
	case 512:
		features += ",+avx512f,+avx512cd,+avx512bw,+avx512dq,+avx512vl"
	default:
		return
	}
	f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute(llvmfmv.MidwayWideAttribute, ""))
	attr := f.impl.GetStringAttributeAtIndex(-1, "target-features")
	if !attr.IsNil() {
		features = attr.GetStringValue() + "," + features
	}
	f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute("target-features", features))
}

// MarkRegisterOwner records the owner of an exported Go kernel with a known body.
func (f Function) MarkRegisterOwner(owner string) {
	ft := f.impl.GlobalValueType()
	for _, t := range append(ft.ParamTypes(), ft.ReturnType()) {
		if t.TypeKind() == llvm.VectorTypeKind && f.Prog.TargetData().TypeSizeInBits(t) > 128 {
			f.impl.AddFunctionAttr(f.Prog.ctx.CreateStringAttribute(llvmfmv.RegisterOwnerAttribute, owner))
			return
		}
	}
}

package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

var (
	nativeSignExtKind = llvm.AttributeKindID("signext")
	nativeZeroExtKind = llvm.AttributeKindID("zeroext")
)

var nativeIntegerAttributeKinds = [...]uint{nativeSignExtKind, nativeZeroExtKind}

// nativeIntegerAttrs records signedness while the Go signature is still
// available. LLVM's integer types alone cannot distinguish signed from unsigned
// C values. Apple arm64, SysV amd64, x86-32 and WebAssembly require extension
// attributes for narrow arguments. Native Windows amd64 only requires zeroext
// for bool; other arm64 ABIs leave argument extension to the callee.
func (p Program) nativeIntegerAttrs(sig *types.Signature, ft llvm.Type, add func(int, llvm.Attribute)) {
	if !p.nativeIntegerExtensionRequired() {
		return
	}
	// This runs during SSA codegen, before cabi.TransformModule can reorder or
	// expand parameters. toLLVMFuncBackground preserves signature order and
	// only omits the trailing __llgo_va_list from the fixed LLVM prototype.
	physicalParams := ft.ParamTypes()
	fixedCount := sig.Params().Len()
	if HasNameValist(sig) {
		fixedCount--
	}
	if len(physicalParams) != fixedCount {
		panic("ssa: native integer attributes require the unlowered function signature")
	}
	addInteger := func(index int, raw types.Type, physical llvm.Type) {
		basic, ok := raw.Underlying().(*types.Basic)
		if !ok || physical.TypeKind() != llvm.IntegerTypeKind || physical.IntTypeWidth() >= 32 {
			return
		}
		if target := p.Target(); target.effectiveGOARCH() == "amd64" &&
			target.effectiveGOOS() == "windows" && basic.Info()&types.IsBoolean == 0 {
			return
		}
		kind := nativeSignExtKind
		if basic.Info()&(types.IsUnsigned|types.IsBoolean) != 0 {
			kind = nativeZeroExtKind
		}
		add(index, p.ctx.CreateEnumAttribute(kind, 0))
	}
	// Return attributes require Go-to-native callback adapters: a native
	// function pointer can currently refer to an unadapted Go entry.
	// The LLVM prototype omits __llgo_va_list. Only fixed parameters carry
	// extension attributes; callers supply the promoted ellipsis argument types.
	for i, physical := range physicalParams {
		addInteger(i+1, sig.Params().At(i).Type(), physical)
	}
}

func (p Program) nativeIntegerExtensionRequired() bool {
	target := p.Target()
	switch target.effectiveGOARCH() {
	case "amd64", "386", "wasm":
		return true
	case "arm64":
		return target.effectiveGOOS() == "darwin"
	}
	return false
}

func (b Builder) setNativeIntegerCallAttrs(call llvm.Value, fn Expr, sig *types.Signature) {
	if !b.Prog.nativeIntegerExtensionRequired() {
		return
	}
	if fn.kind == vkFuncPtr {
		b.Prog.nativeIntegerAttrs(sig, call.CalledFunctionType(), call.AddCallSiteAttribute)
		return
	}
	// Direct C declarations already carry their ABI attributes. Copy them onto
	// the call too: LLVM requires matching attributes at both ends of the boundary.
	if direct := fn.impl.IsAFunction(); !direct.IsNil() {
		copyInteger := func(i int, physical llvm.Type) {
			if physical.TypeKind() != llvm.IntegerTypeKind || physical.IntTypeWidth() >= 32 {
				return
			}
			for _, kind := range nativeIntegerAttributeKinds {
				if attr := direct.GetEnumAttributeAtIndex(i, kind); !attr.IsNil() {
					call.AddCallSiteAttribute(i, attr)
				}
			}
		}
		ft := call.CalledFunctionType()
		for i, physical := range ft.ParamTypes() {
			copyInteger(i+1, physical)
		}
	}
}

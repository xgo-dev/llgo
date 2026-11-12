package ssa

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	intllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

var (
	nativeSignExtKind = llvm.AttributeKindID("signext")
	nativeZeroExtKind = llvm.AttributeKindID("zeroext")
)

var nativeIntegerAttributeKinds = [...]uint{nativeSignExtKind, nativeZeroExtKind}

// nativeIntegerAttrs emits LLVM ABI parameter attributes while the Go signature
// still carries signedness. LLVM integer types alone do not distinguish signed
// from unsigned C values. These attributes tell LLVM codegen how to extend
// arguments at the call boundary; no separate integer-extension pass is needed.
// ARM32, RISC-V, Apple arm64, SysV amd64, x86-32 and WebAssembly require
// extension attributes for narrow arguments. RV64 also sign-extends both signed
// and unsigned 32-bit arguments. Windows amd64 only requires zeroext for bool;
// other arm64 ABIs leave argument extension to the callee.
// Return attributes require Go-to-native callback adapters: a native function
// pointer can currently refer to an unadapted Go entry.
func (p Program) nativeIntegerAttrs(sig *types.Signature, ft llvm.Type, add func(int, llvm.Attribute)) {
	extension := p.nativeIntegerExt
	if extension == nativeIntegerNoExt {
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
		if !ok || physical.TypeKind() != llvm.IntegerTypeKind {
			return
		}
		width := physical.IntTypeWidth()
		if width > 32 || width == 32 && extension != nativeIntegerRV64Ext {
			return
		}
		if extension == nativeIntegerBoolExt && basic.Info()&types.IsBoolean == 0 {
			return
		}
		kind := nativeSignExtKind
		if width < 32 && basic.Info()&(types.IsUnsigned|types.IsBoolean) != 0 {
			kind = nativeZeroExtKind
		}
		add(index, p.ctx.CreateEnumAttribute(kind, 0))
	}
	// The LLVM prototype omits __llgo_va_list. Call sites handle any ellipsis
	// attributes separately, using the actual promoted argument types.
	for i, physical := range physicalParams {
		addInteger(i+1, sig.Params().At(i).Type(), physical)
	}
}

type nativeIntegerExtension uint8

const (
	nativeIntegerNoExt nativeIntegerExtension = iota
	nativeIntegerNarrowExt
	nativeIntegerBoolExt
	nativeIntegerRV64Ext
)

func nativeIntegerExtensionForTarget(target *Target) nativeIntegerExtension {
	// Named embedded targets may use GOARCH only for package selection (for
	// example, RISC-V profiles use arm). Their ABI follows the physical triple,
	// even when Spec retains a compatibility target for the host LLVM library.
	triple := target.LLVMTarget
	if triple == "" {
		triple = target.Spec().Triple
	}
	triple = strings.ToLower(triple)
	arch, _, _ := strings.Cut(triple, "-")
	switch {
	case arch == "arm64", arch == "arm64_32", arch == "aarch64", arch == "aarch64_be":
		if intllvm.IsDarwinTarget(triple) {
			return nativeIntegerNarrowExt
		}
	case arch == "x86_64", arch == "amd64":
		if strings.Contains(triple, "windows") || strings.Contains(triple, "win32") || strings.Contains(triple, "mingw") {
			return nativeIntegerBoolExt
		}
		return nativeIntegerNarrowExt
	case arch == "riscv64":
		return nativeIntegerRV64Ext
	case arch == "riscv32", arch == "wasm32", arch == "wasm64",
		arch == "x86", arch == "386", arch == "i386", arch == "i486", arch == "i586", arch == "i686",
		strings.HasPrefix(arch, "arm"), strings.HasPrefix(arch, "thumb"):
		return nativeIntegerNarrowExt
	}
	return nativeIntegerNoExt
}

func (b Builder) setNativeIntegerCallAttrs(call llvm.Value, fn Expr, sig *types.Signature, params []llvm.Value) {
	extension := b.Prog.nativeIntegerExt
	if extension == nativeIntegerNoExt {
		return
	}
	ft := call.CalledFunctionType()
	if fn.kind == vkFuncPtr {
		b.Prog.nativeIntegerAttrs(sig, ft, call.AddCallSiteAttribute)
	} else if direct := fn.impl.IsAFunction(); !direct.IsNil() {
		// Direct C declarations already carry their ABI attributes. Copy them
		// onto the call too, including RV64's 32-bit integer parameters.
		copyInteger := func(i int, physical llvm.Type) {
			if physical.TypeKind() != llvm.IntegerTypeKind || physical.IntTypeWidth() > 32 {
				return
			}
			for _, kind := range nativeIntegerAttributeKinds {
				if attr := direct.GetEnumAttributeAtIndex(i, kind); !attr.IsNil() {
					call.AddCallSiteAttribute(i, attr)
				}
			}
		}
		for i, physical := range ft.ParamTypes() {
			copyInteger(i+1, physical)
		}
	}
	if extension == nativeIntegerRV64Ext && ft.IsFunctionVarArg() {
		// C's promoted int/unsigned int ellipsis arguments also use RV64's
		// sign-extended 32-bit representation. They have no declaration slots.
		for i := len(ft.ParamTypes()); i < len(params); i++ {
			if typ := params[i].Type(); typ.TypeKind() == llvm.IntegerTypeKind && typ.IntTypeWidth() == 32 {
				call.AddCallSiteAttribute(i+1, b.Prog.ctx.CreateEnumAttribute(nativeSignExtKind, 0))
			}
		}
	}
}

// needsNativeIntegerFuncval identifies C entries whose physical prototype or
// integer extension differs from the Go funcval ABI. Adapt at the conversion
// boundary instead of changing calls through arbitrary Go values.
func (b Builder) needsNativeIntegerFuncval(fn Expr) bool {
	if b.Prog.usesWideGoStorage() {
		switch fn.kind {
		case vkFuncPtr:
			sig := fn.raw.Type.Underlying().(*types.Signature)
			if b.Prog.FuncDecl(sig, CDecl).ll != b.Prog.FuncDecl(sig, GoDecl).ll {
				return true
			}
		case vkFuncDecl:
			if b.Prog.isNativeStorage(fn.Type) && fn.ll != b.Prog.FuncDecl(fn.raw.Type.(*types.Signature), GoDecl).ll {
				return true
			}
		}
	}
	if b.Prog.nativeIntegerExt == nativeIntegerNoExt {
		return false
	}
	switch fn.kind {
	case vkFuncPtr:
		sig := fn.raw.Type.Underlying().(*types.Signature)
		needed := false
		b.Prog.nativeIntegerAttrs(sig, b.Prog.FuncDecl(sig, CDecl).ll, func(int, llvm.Attribute) {
			needed = true
		})
		return needed
	case vkFuncDecl:
		if direct := fn.impl.IsAFunction(); !direct.IsNil() {
			for i := range fn.ll.ParamTypes() {
				for _, kind := range nativeIntegerAttributeKinds {
					if !direct.GetEnumAttributeAtIndex(i+1, kind).IsNil() {
						return true
					}
				}
			}
		}
	}
	return false
}

// nativeCallback gives a known Go entry a native prototype. A C callback has
// no environment slot, so a dynamically selected Go funcval cannot retain the
// code pointer needed by an adapter when the two physical ABIs differ.
func (b Builder) nativeCallback(dst Type, fn Expr, cc CallConv, abi string) Expr {
	direct := fn.impl.IsAFunction()
	if direct.IsNil() {
		panic("C callback with a different ABI must be a direct function reference")
	}
	sig := dst.raw.Type.Underlying().(*types.Signature)
	if HasNameValist(sig) {
		panic("C variadic callbacks require a native function pointer")
	}
	name := b.Pkg.Path() + ".__llgo_" + abi + "$" + direct.Name() + "$" + b.Prog.abi.FuncName(sig)
	wrapper := b.Pkg.FuncOf(name)
	if wrapper == nil {
		wrapper = b.Pkg.NewFunc(name, sig, cc)
		wrapper.impl.SetLinkage(llvm.InternalLinkage)
		body := wrapper.MakeBody(1)
		args := make([]Expr, sig.Params().Len())
		for i := range args {
			args[i] = wrapper.Param(i)
		}
		result := body.Call(fn, args...)
		switch n := sig.Results().Len(); n {
		case 0:
			body.Return()
		case 1:
			body.Return(result)
		default:
			results := make([]Expr, n)
			for i := range results {
				results[i] = body.Extract(result, i)
			}
			body.Return(results...)
		}
	}
	return Expr{wrapper.impl, dst}
}

// nativeFuncval adapts a native function pointer to LLGo's ordinary funcval
// representation. Its environment holds the native pointer directly, and the
// Go entry calls that pointer with the native ABI. This avoids an allocation
// while preserving nil: a nil native pointer produces a
// funcval with a nil code pointer.
func (b Builder) nativeFuncval(dst Type, fn Expr, callConv llvm.CallConv, abi string) Expr {
	closure, ok := types.Unalias(dst.raw.Type).Underlying().(*types.Struct)
	if !ok || !IsClosure(closure) {
		panic(fmt.Errorf("native funcval target must be a Go function, got %s", dst.raw.Type))
	}
	goSig := closure.Field(0).Type().(*types.Signature)
	nativeSig := types.Unalias(fn.raw.Type).Underlying().(*types.Signature)
	// A Go variadic slice cannot forward an arbitrary C ellipsis tail. Keep
	// those calls on the native function-pointer path, where the caller knows
	// the concrete promoted argument types.
	if HasNameValist(nativeSig) {
		panic("C variadic functions require a native function pointer; use //llgo:cdecl")
	}
	wrapperName := b.Pkg.Path() + ".__llgo_" + abi + "_funcval$" + b.Prog.abi.FuncName(goSig)
	wrapper := b.Pkg.FuncOf(wrapperName)
	if wrapper == nil {
		env := types.NewVar(token.NoPos, nil, "$env", types.Typ[types.UnsafePointer])
		wrapper = b.Pkg.NewEnvFunc(wrapperName, goSig, GoDecl, env, false)
		wrapper.impl.SetLinkage(llvm.InternalLinkage)
		body := wrapper.MakeBody(1)
		args := make([]Expr, goSig.Params().Len())
		for i := range args {
			args[i] = wrapper.Param(i)
		}
		native := Expr{wrapper.Env().impl, b.Prog.rawType(nativeSig)}
		result := body.Call(native, args...)
		if callConv != llvm.CCallConv {
			result.impl.SetInstructionCallConv(callConv)
		}
		switch n := goSig.Results().Len(); n {
		case 0:
			body.Return()
		case 1:
			body.Return(result)
		default:
			results := make([]Expr, n)
			for i := range results {
				results[i] = body.Extract(result, i)
			}
			body.Return(results...)
		}
	}

	isNil := llvm.CreateICmp(b.impl, llvm.IntEQ, fn.impl, llvm.ConstNull(fn.impl.Type()))
	code := llvm.CreateSelect(b.impl, isNil, llvm.ConstNull(wrapper.impl.Type()), wrapper.impl)
	return b.aggregateValue(dst, code, fn.impl)
}

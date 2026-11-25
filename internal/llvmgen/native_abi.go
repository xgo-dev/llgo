package llvmgen

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go/token"
	"go/types"

	lltypes "github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llvm"
)

var (
	nativeSignExtKind = llvm.AttributeKindID("signext")
	nativeZeroExtKind = llvm.AttributeKindID("zeroext")
)

var nativeIntegerAttributeKinds = [...]uint{nativeSignExtKind, nativeZeroExtKind}

const nativeCallbackABIError = "C callback with a different ABI must be a direct function reference"

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
	if extension == lltypes.NativeIntNoExt {
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
		if width > 32 || width == 32 && extension != lltypes.NativeIntRV64Ext {
			return
		}
		if extension == lltypes.NativeIntBoolExt && basic.Info()&types.IsBoolean == 0 {
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

func (b Builder) setNativeIntegerCallAttrs(call llvm.Value, fn Expr, sig *types.Signature, params []llvm.Value) {
	extension := b.Prog.nativeIntegerExt
	if extension == lltypes.NativeIntNoExt {
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
	if extension == lltypes.NativeIntRV64Ext && ft.IsFunctionVarArg() {
		// C's promoted int/unsigned int ellipsis arguments also use RV64's
		// sign-extended 32-bit representation. They have no declaration slots.
		for i := len(ft.ParamTypes()); i < len(params); i++ {
			if typ := params[i].Type(); typ.TypeKind() == llvm.IntegerTypeKind && typ.IntTypeWidth() == 32 {
				call.AddCallSiteAttribute(i+1, b.Prog.ctx.CreateEnumAttribute(nativeSignExtKind, 0))
			}
		}
	}
}

func (p Program) isWasm32NativeCall(fn Expr) bool {
	return p.usesWideGoStorage() && (fn.kind == vkFuncPtr || fn.kind == vkFuncDecl && p.isNativeStorage(fn.Type))
}

// needsNativeFuncval identifies C entries whose prototype, result normalization
// or integer extension differs from the Go funcval ABI. Adapt at conversion
// instead of changing calls through arbitrary Go values.
func (b Builder) needsNativeFuncval(fn Expr) bool {
	prog := b.Prog
	if fn.kind != vkFuncPtr && fn.kind != vkFuncDecl {
		return false
	}
	wasmNative := prog.isWasm32NativeCall(fn)
	if !wasmNative && prog.nativeIntegerExt == lltypes.NativeIntNoExt {
		return false
	}
	sig, native := prog.callSignature(fn)
	if wasmNative && (prog.hasWasmABIPadding(native.ReturnType()) || native != prog.funcDeclType(sig, GoDecl).ll) {
		return true
	}
	if fn.kind == vkFuncPtr {
		needed := false
		prog.nativeIntegerAttrs(sig, native, func(int, llvm.Attribute) { needed = true })
		return needed
	}
	if direct := fn.impl.IsAFunction(); !direct.IsNil() {
		for i := range fn.ll.ParamTypes() {
			for _, kind := range nativeIntegerAttributeKinds {
				if !direct.GetEnumAttributeAtIndex(i+1, kind).IsNil() {
					return true
				}
			}
		}
	}
	return false
}

// canCallGoEntry tells reflection when an environment-free Go function can
// occupy this native callback slot without a compiler-generated adapter.
func (p Program) canCallGoEntry(t types.Type) bool {
	sig, ok := t.Underlying().(*types.Signature)
	if !ok || sig.Variadic() || HasNameValist(sig) || p.isStdcallType(t) && p.stdcallCallConv() != llvm.CCallConv {
		return false
	}
	return p.toLLVMFunc(p.gocvt.cvtFunc(sig, nil)) == p.funcDeclType(sig, CDecl).ll
}

// closureToNative recovers the original C entry when a native funcval is
// passed back as a callback. An ordinary Go callback must be environment-free.
func (b Builder) closureToNative(dst Type, fn Expr) Expr {
	nativeSig := dst.raw.Type.Underlying().(*types.Signature)
	goSig := fn.raw.Type.Underlying().(*types.Struct).Field(0).Type().(*types.Signature)
	prog := b.Prog
	nativeType := prog.funcDeclType(nativeSig, CDecl).ll
	sameABI := prog.toLLVMFunc(goSig) == nativeType
	if sameABI && !fn.impl.IsAFunction().IsNil() {
		return Expr{fn.impl, dst}
	}
	code, env := b.Field(fn, 0), b.Field(fn, 1)
	if !sameABI {
		if code.impl.IsNull() && env.impl.IsNull() {
			return Expr{llvm.ConstNull(dst.ll), dst}
		}
		if code.impl.IsAFunction().IsNil() || !env.impl.IsNull() {
			panic(nativeCallbackABIError)
		}
		return b.nativeCallback(dst, Expr{code.impl, prog.FuncDecl(goSig, GoDecl)}, CDecl, "c_callback")
	}
	if env.impl.IsNull() {
		return Expr{code.impl, dst}
	}

	entry := code.impl
	captured := llvm.CreateICmp(b.impl, llvm.IntNE, env.impl, llvm.ConstNull(env.impl.Type()))
	if prog.hasWasmABIPadding(nativeType.ReturnType()) && !HasNameValist(nativeSig) {
		// The adapter's environment is the native pointer itself. Compare the
		// code address before accepting it: real closures also have an environment.
		adapter := b.nativeFuncvalAdapter(goSig, nativeSig, llvm.CCallConv, "c", nativeType)
		isAdapter := llvm.CreateICmp(b.impl, llvm.IntEQ, entry, adapter.impl)
		entry = llvm.CreateSelect(b.impl, isAdapter, env.impl, entry)
		captured = b.impl.CreateAnd(captured, b.impl.CreateNot(isAdapter, ""), "")
	}
	b.assertRuntimeError(captured, "C callback must not capture variables")
	return Expr{entry, dst}
}

// nativeCallback gives a known Go entry a native prototype. A C callback has
// no environment slot, so a dynamically selected Go funcval cannot retain the
// code pointer needed by an adapter when the two physical ABIs differ.
func (b Builder) nativeCallback(dst Type, fn Expr, cc CallConv, abi string) Expr {
	direct := fn.impl.IsAFunction()
	if direct.IsNil() {
		panic(nativeCallbackABIError)
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
		wrapper.buildAdapterBody(fn, direct.FunctionCallConv())
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
	wrapper := b.funcvalAdapterFor(fn, goSig, callConv, abi)

	isNil := llvm.CreateICmp(b.impl, llvm.IntEQ, fn.impl, llvm.ConstNull(fn.impl.Type()))
	code := llvm.CreateSelect(b.impl, isNil, llvm.ConstNull(wrapper.impl.Type()), wrapper.impl)
	return b.aggregateValue(dst, code, fn.impl)
}

func (b Builder) funcvalAdapterFor(fn Expr, goSig *types.Signature, callConv llvm.CallConv, abi string) Function {
	nativeSig, prototype := b.Prog.callSignature(fn)
	var sharedPrototype llvm.Type
	if b.Prog.usesWideGoStorage() && b.Prog.hasWasmABIPadding(prototype.ReturnType()) && prototype == b.Prog.toLLVMFunc(goSig) {
		sharedPrototype = prototype
	}
	return b.nativeFuncvalAdapter(goSig, nativeSig, callConv, abi, sharedPrototype)
}

// A padded result needs a Go entry even when the two prototypes match. Give
// that entry one address across packages so closureToNative can recognize it.
func (b Builder) nativeFuncvalAdapter(goSig, nativeSig *types.Signature, callConv llvm.CallConv, abi string, sharedPrototype llvm.Type) Function {
	prefix := "__llgo_" + abi + "_funcval$"
	var name string
	if !sharedPrototype.IsNil() {
		name = prefix + b.Prog.wasmNativeFuncvalKey(goSig, nativeSig, sharedPrototype)
	} else {
		name = b.Pkg.Path() + "." + prefix + b.Prog.abi.FuncName(goSig)
	}
	if wrapper := b.Pkg.FuncOf(name); wrapper != nil {
		return wrapper
	}
	env := types.NewVar(token.NoPos, nil, "$env", types.Typ[types.UnsafePointer])
	wrapper := b.Pkg.NewEnvFunc(name, goSig, GoDecl, env, false)
	if !sharedPrototype.IsNil() {
		b.Pkg.setODRLinkage(wrapper.impl, llvm.LinkOnceODRLinkage)
		wrapper.impl.SetVisibility(llvm.HiddenVisibility)
	} else {
		wrapper.impl.SetLinkage(llvm.InternalLinkage)
	}
	native := Expr{wrapper.Env().impl, b.Prog.rawType(nativeSig)}
	wrapper.buildAdapterBody(native, callConv)
	return wrapper
}

// The shared adapter has the same physical Go and C prototype. Its linkage
// identity must also describe header normalization, Go storage and C integer
// extensions without depending on how different packages spell the types.
func (p Program) wasmNativeFuncvalKey(goSig, nativeSig *types.Signature, prototype llvm.Type) string {
	shape, _ := p.wasmCallShape(goSig, nativeSig, prototype, prototype)
	digest := sha256.Sum256([]byte(shape))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// buildAdapterBody forwards the wrapper's parameters to the Go or native callee.
func (wrapper Function) buildAdapterBody(fn Expr, callConv llvm.CallConv) {
	body := wrapper.MakeBody(1)
	defer body.Dispose()
	args := make([]Expr, len(wrapper.params))
	for i := range args {
		args[i] = wrapper.Param(i)
	}
	result := body.callFunction(fn, args, callConv)
	switch n := wrapper.raw.Type.(*types.Signature).Results().Len(); n {
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
	body.EndBuild()
}

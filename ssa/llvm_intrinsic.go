package ssa

import (
	"fmt"
	"go/types"
	"sync"

	"github.com/xgo-dev/llgo/internal/llvmintrinsic"
	"github.com/xgo-dev/llvm"
)

var (
	llvmBswapType = llvmintrinsic.Func(
		llvmintrinsic.Bind("T", llvmintrinsic.Where(llvmintrinsic.Integers, "bswap width", func(_ llvmintrinsic.Context, t llvm.Type) error {
			bits, err := llvmintrinsic.Bits(t)
			if err != nil {
				return err
			}
			if bits%16 != 0 {
				return fmt.Errorf("bswap must be an even number of bytes")
			}
			return nil
		})), llvmintrinsic.Ref("T"))
	llvmFloatToIntType = llvmintrinsic.Func(
		llvmintrinsic.Bind("result", llvmintrinsic.Integers),
		llvmintrinsic.Shape(llvmintrinsic.Ref("result"), llvmintrinsic.Floating))
	llvmCopyType       = llvmintrinsic.Func(llvmintrinsic.Void, llvmintrinsic.AnyPointer, llvmintrinsic.AnyPointer, llvmintrinsic.Integer, llvmintrinsic.I1)
	llvmCopyInlineType = llvmintrinsic.Func(llvmintrinsic.Void, llvmintrinsic.AnyPointer, llvmintrinsic.AnyPointer, llvmintrinsic.Const(llvmintrinsic.Integer), llvmintrinsic.I1)
	llvmSetType        = llvmintrinsic.Func(llvmintrinsic.Void, llvmintrinsic.AnyPointer, llvmintrinsic.I8, llvmintrinsic.Integer, llvmintrinsic.I1)
	llvmSetInlineType  = llvmintrinsic.Func(llvmintrinsic.Void, llvmintrinsic.AnyPointer, llvmintrinsic.I8, llvmintrinsic.Const(llvmintrinsic.Integer), llvmintrinsic.I1)
)

// This is an availability policy. Nil descriptions use LLVM's metadata;
// refinements express constraints missing from that metadata.
var llvmSupportedIntrinsics = []llvmintrinsic.Intrinsic{
	{Name: "llvm.abs"}, {Name: "llvm.bitreverse"}, {Name: "llvm.bswap", Type: llvmBswapType},
	{Name: "llvm.ctlz"}, {Name: "llvm.cttz"}, {Name: "llvm.ctpop"},
	{Name: "llvm.fshl"}, {Name: "llvm.fshr"},
	{Name: "llvm.smax"}, {Name: "llvm.smin"}, {Name: "llvm.umax"}, {Name: "llvm.umin"},
	{Name: "llvm.sadd.sat"}, {Name: "llvm.uadd.sat"}, {Name: "llvm.ssub.sat"}, {Name: "llvm.usub.sat"},
	{Name: "llvm.sshl.sat"}, {Name: "llvm.ushl.sat"},
	{Name: "llvm.sadd.with.overflow"}, {Name: "llvm.uadd.with.overflow"},
	{Name: "llvm.ssub.with.overflow"}, {Name: "llvm.usub.with.overflow"},
	{Name: "llvm.smul.with.overflow"}, {Name: "llvm.umul.with.overflow"},
	{Name: "llvm.fabs"}, {Name: "llvm.sqrt"}, {Name: "llvm.sin"}, {Name: "llvm.cos"},
	{Name: "llvm.exp"}, {Name: "llvm.exp2"}, {Name: "llvm.log"}, {Name: "llvm.log2"}, {Name: "llvm.log10"},
	{Name: "llvm.pow"}, {Name: "llvm.powi", Type: llvmintrinsic.Func(nil, nil, llvmintrinsic.Integer)},
	{Name: "llvm.fma"}, {Name: "llvm.fmuladd"}, {Name: "llvm.floor"}, {Name: "llvm.ceil"},
	{Name: "llvm.trunc"}, {Name: "llvm.rint"}, {Name: "llvm.nearbyint"},
	{Name: "llvm.round"}, {Name: "llvm.roundeven"}, {Name: "llvm.copysign"},
	{Name: "llvm.minnum"}, {Name: "llvm.maxnum"}, {Name: "llvm.minimum"}, {Name: "llvm.maximum"},
	{Name: "llvm.fptosi.sat", Type: llvmFloatToIntType}, {Name: "llvm.fptoui.sat", Type: llvmFloatToIntType},
	{Name: "llvm.memcpy", Type: llvmCopyType}, {Name: "llvm.memcpy.inline", Type: llvmCopyInlineType},
	{Name: "llvm.memmove", Type: llvmCopyType}, {Name: "llvm.memset", Type: llvmSetType}, {Name: "llvm.memset.inline", Type: llvmSetInlineType},
}

var llvmIntrinsicNames = sync.OnceValue(func() map[int]llvmintrinsic.Intrinsic {
	names := make(map[int]llvmintrinsic.Intrinsic, len(llvmSupportedIntrinsics))
	for _, spec := range llvmSupportedIntrinsics {
		if id := llvm.LookupIntrinsicID(spec.Name); id != 0 {
			names[id] = spec
		}
	}
	return names
})

// LLVMIntrinsic connects a checked LLVM binding to its Go result representation.
type LLVMIntrinsic struct {
	binding *llvmintrinsic.Binding
	result  Type
}

// ResolveLLVMIntrinsic maps Go value types, then delegates validation and
// overload resolution to llvmintrinsic. Callers cache it per Go declaration.
func (p Program) ResolveLLVMIntrinsic(name string, sig *types.Signature) (*LLVMIntrinsic, error) {
	id := llvm.LookupIntrinsicID(name)
	if id == 0 {
		return nil, fmt.Errorf("unknown LLVM intrinsic %q", name)
	}
	spec, supported := llvmIntrinsicNames()[id]
	if !supported {
		return nil, fmt.Errorf("LLVM intrinsic %q is not supported: only portable math and memory operations are supported", name)
	}
	signature, result, err := p.llvmIntrinsicSignature(sig)
	if err != nil {
		return nil, fmt.Errorf("LLVM intrinsic %q: %w", name, err)
	}
	spec.Name = name // Preserve explicit overload suffixes for validation.
	binding, err := spec.Resolve(signature, llvmintrinsic.WithDataLayout(p.DataLayout()))
	if err != nil {
		return nil, err
	}
	return &LLVMIntrinsic{binding, result}, nil
}

func (p Program) llvmIntrinsicSignature(sig *types.Signature) (llvm.Type, Type, error) {
	if sig.Recv() != nil || sig.TypeParams().Len() != 0 {
		return llvm.Type{}, nil, fmt.Errorf("requires a non-generic function without a receiver")
	}
	n := sig.Params().Len()
	variadic := HasNameValist(sig)
	if variadic {
		last, ok := sig.Params().At(n - 1).Type().Underlying().(*types.Slice)
		if !ok {
			return llvm.Type{}, nil, fmt.Errorf("%s must have type ...any", NameValist)
		}
		elem, ok := last.Elem().Underlying().(*types.Interface)
		if !ok || !elem.Empty() {
			return llvm.Type{}, nil, fmt.Errorf("%s must have type ...any", NameValist)
		}
		n--
	}
	// Ordinary Go ...T retains its slice parameter. Only NameValist turns into
	// LLVM ellipsis, and intrinsic signatures never undergo the native C ABI.
	params := p.toLLVMTypes(sig.Params(), n)
	result := p.retType(sig)
	ret := result.ll
	if sig.Results().Len() > 1 {
		// LLVM overflow results are logical structs even when Go tuple storage
		// has a different layout (notably 64-bit fields on 386).
		ret = p.ctx.StructType(p.toLLVMTypes(sig.Results(), sig.Results().Len()), false)
	}
	return llvm.FunctionType(ret, params, variadic), result, nil
}

func (b Builder) CallLLVMIntrinsic(d *LLVMIntrinsic, args ...Expr) (Expr, error) {
	values := make([]llvm.Value, len(args))
	for i, arg := range args {
		values[i] = b.logicalValue(arg).impl
	}
	value, err := d.binding.Call(b.impl, b.Pkg.mod, values...)
	if err != nil {
		return Nil, err
	}
	return Expr{b.fitLLVMResult(value, d.result), d.result}, nil
}

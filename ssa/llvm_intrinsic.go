package ssa

import (
	"fmt"
	"go/types"
	"strings"
	"sync"

	"github.com/xgo-dev/llvm"
)

// llvmScalarIntrinsics is a support policy, not a signature table. LLVM owns
// the signatures, overloads, attributes and additional verifier constraints.
// Keep this surface limited to portable scalar math: other intrinsic families
// can require target features, special calling conventions or memory semantics.
var llvmScalarIntrinsics = []string{
	"abs", "bitreverse", "bswap", "ctlz", "cttz", "ctpop", "fshl", "fshr",
	"smax", "smin", "umax", "umin",
	"sadd.sat", "uadd.sat", "ssub.sat", "usub.sat", "sshl.sat", "ushl.sat",
	"sadd.with.overflow", "uadd.with.overflow", "ssub.with.overflow",
	"usub.with.overflow", "smul.with.overflow", "umul.with.overflow",
	"fabs", "sqrt", "sin", "cos", "exp", "exp2", "log", "log2", "log10",
	"pow", "powi", "fma", "fmuladd", "floor", "ceil", "trunc", "rint",
	"nearbyint", "round", "roundeven", "copysign",
	"minnum", "maxnum", "minimum", "maximum", "fptosi.sat", "fptoui.sat",
}

// Resolve the policy names through LLVM once. Name recognition belongs to
// LookupIntrinsicID; LLGo only decides which resolved families it supports.
var llvmScalarIntrinsicNames = sync.OnceValue(func() map[int]string {
	names := make(map[int]string, len(llvmScalarIntrinsics))
	for _, family := range llvmScalarIntrinsics {
		name := "llvm." + family
		if id := llvm.LookupIntrinsicID(name); id != 0 {
			names[id] = name
		}
	}
	return names
})

// LLVMIntrinsic is a checked scalar intrinsic declaration. It uses LLVM value
// types, independently of the platform C ABI and the Go aggregate storage ABI.
type LLVMIntrinsic struct {
	id      int
	name    string
	params  []llvm.Type
	result  Type
	ret     llvm.Type
	immargs []int
}

// ResolveLLVMIntrinsic validates a Go declaration against the linked LLVM's
// intrinsic metadata. Callers should cache the result for each declaration.
func (p Program) ResolveLLVMIntrinsic(name string, sig *types.Signature) (*LLVMIntrinsic, error) {
	id := llvm.LookupIntrinsicID(name)
	if id == 0 {
		return nil, fmt.Errorf("unknown LLVM intrinsic %q", name)
	}
	base, supported := llvmScalarIntrinsicNames()[id]
	if !supported {
		return nil, fmt.Errorf("LLVM intrinsic %q is not supported: only portable scalar math intrinsics are supported", name)
	}
	if sig.Recv() != nil || sig.TypeParams().Len() != 0 || sig.Variadic() {
		return nil, fmt.Errorf("LLVM intrinsic %q requires a non-generic, non-variadic function without a receiver", name)
	}
	for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
		for i := 0; i < tuple.Len(); i++ {
			t := tuple.At(i).Type()
			basic, ok := t.Underlying().(*types.Basic)
			if !ok || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat) == 0 {
				return nil, fmt.Errorf("LLVM intrinsic %q: unsupported type %s; expected a scalar boolean, integer or floating-point type", name, t)
			}
		}
	}
	d := &LLVMIntrinsic{id: id, result: p.retType(sig)}
	d.params = p.toLLVMTypes(sig.Params(), sig.Params().Len())
	d.ret = d.result.ll
	if sig.Results().Len() > 1 {
		// For example, overflow intrinsics return {i64, i1}, including on
		// 386 where the corresponding Go tuple has a different layout.
		d.ret = p.ctx.StructType(p.toLLVMTypes(sig.Results(), sig.Results().Len()), false)
	}

	// The Go binding already matches LLVM's IIT descriptors and infers the
	// overloads before building a call. Use a short-lived module to query that
	// machinery without polluting the output module, then run the verifier to
	// catch constraints beyond the IIT signature (such as bswap's bit width).
	// Zero scalar constants also satisfy the supported families' immarg flags.
	mod := p.ctx.NewModule("intrinsic.validation")
	defer mod.Dispose()
	fn := llvm.AddFunction(mod, "check", llvm.FunctionType(p.tyVoid(), nil, false))
	b := p.ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(p.ctx.AddBasicBlock(fn, "entry"))
	args := make([]llvm.Value, len(d.params))
	for i, t := range d.params {
		args[i] = llvm.ConstNull(t)
	}
	call := b.CreateIntrinsic(d.ret, id, args, "")
	if call.IsNil() {
		return nil, fmt.Errorf("LLVM intrinsic %q: signature %s does not match LLVM's intrinsic signature", name, llvm.FunctionType(d.ret, d.params, false))
	}
	decl := call.CalledValue()
	d.name = decl.Name()
	if name != base && name != d.name {
		return nil, fmt.Errorf("LLVM intrinsic %q: name does not match declaration; expected %q", name, d.name)
	}
	b.CreateRetVoid()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		return nil, fmt.Errorf("invalid LLVM intrinsic %q: %s", name, strings.TrimSpace(err.Error()))
	}
	immarg := llvm.AttributeKindID("immarg")
	for i := range d.params {
		if !decl.GetEnumAttributeAtIndex(i+1, immarg).IsNil() {
			d.immargs = append(d.immargs, i)
		}
	}
	return d, nil
}

// CallLLVMIntrinsic emits a direct call, checking call-site immediate arguments.
func (b Builder) CallLLVMIntrinsic(d *LLVMIntrinsic, args ...Expr) (Expr, error) {
	if len(args) != len(d.params) {
		return Nil, fmt.Errorf("%s: expected %d arguments, got %d", d.name, len(d.params), len(args))
	}
	values := make([]llvm.Value, len(args))
	for i, arg := range args {
		values[i] = b.logicalValue(arg).impl
		if values[i].Type() != d.params[i] {
			return Nil, fmt.Errorf("%s: argument %d has type %s, expected %s", d.name, i+1, values[i].Type(), d.params[i])
		}
	}
	for _, i := range d.immargs {
		v := values[i]
		if v.IsAConstantInt().IsNil() && v.IsAConstantFP().IsNil() {
			return Nil, fmt.Errorf("%s: argument %d must be an immediate constant (immarg)", d.name, i+1)
		}
	}
	if existing := b.Pkg.mod.NamedFunction(d.name); !existing.IsNil() {
		if existing.GlobalValueType() != llvm.FunctionType(d.ret, d.params, false) ||
			!existing.IsDeclaration() || existing.FunctionCallConv() != llvm.CCallConv {
			return Nil, fmt.Errorf("%s: conflicting LLVM intrinsic declaration", d.name)
		}
	}
	value := b.impl.CreateIntrinsic(d.ret, d.id, values, "")
	return Expr{b.fitLLVMResult(value, d.result), d.result}, nil
}

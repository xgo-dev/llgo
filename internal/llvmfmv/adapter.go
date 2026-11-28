package llvmfmv

import (
	"crypto/sha256"
	"fmt"

	xtllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// The actual Go entry keeps its platform register ABI. Only the call from a
// narrower body into this private adapter uses 128-bit pieces. The target is
// an argument, so direct calls and function values can reuse the same adapter.
type adapterKey struct {
	target    llvm.Type
	transport llvm.Type
	varargs   llvm.Type // actual machine argument types beyond a variadic signature
	cc        llvm.CallConv
	attrs     string
}

func (a *wideABI) mismatchedCalls(fn llvm.Value, baseline string) []llvm.Value {
	floor := a.registerFloor(fn, baseline)
	var calls []llvm.Value
	a.calls(fn, func(call llvm.Value, width int) {
		if width > floor {
			calls = append(calls, call)
		}
	})
	return calls
}

func (a *wideABI) adaptCalls(fn llvm.Value, baseline string) {
	for _, call := range a.mismatchedCalls(fn, baseline) {
		a.adaptCall(call)
	}
}

func (a *wideABI) adaptCall(old llvm.Value) {
	bundles := adapterBundles(old)
	defer disposeAdapterBundles(bundles)
	b := a.mod.Context().NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(old)
	args := []llvm.Value{old.CalledValue()}
	for i := 0; i < old.ArgOperandsCount(); i++ {
		args = a.splitArgument(b, old.Operand(i), args)
	}
	ret := old.Type()
	privateRet := ret
	if ret.TypeKind() != llvm.VoidTypeKind {
		// Splitting undef gives the piece types without emitting instructions.
		privateRet = packResult(b, a.splitArgument(b, llvm.Undef(ret), nil)).Type()
	}
	ft := llvm.FunctionType(privateRet, valueTypes(args), false)
	key := adapterKey{target: old.CalledFunctionType(), transport: ft, cc: old.InstructionCallConv(), attrs: adapterAttributeKey(old)}
	if key.target.IsFunctionVarArg() {
		var types []llvm.Type
		for i := key.target.ParamTypesCount(); i < old.ArgOperandsCount(); i++ {
			types = append(types, old.Operand(i).Type())
		}
		key.varargs = llvm.FunctionType(ret, types, false)
	}
	helper := a.adapters[key]
	if helper.IsNil() {
		helper = a.newRegisterAdapter(old, key)
		a.adapters[key] = helper
	}

	var normal, unwind, continuation llvm.BasicBlock
	if !old.IsAInvokeInst().IsNil() {
		normal, unwind = old.Successor(0), old.Successor(1)
		if ret != privateRet && !old.FirstUse().IsNil() {
			// The invoke result exists only on the normal edge, including when
			// its first use is an incoming value of a shared successor's PHI.
			continuation = a.mod.Context().InsertBasicBlock(normal, "regcall.result")
			replacePhiPredecessor(b, normal, old.InstructionParent(), continuation)
		}
	}
	destination := normal
	if !continuation.IsNil() {
		destination = continuation
	}
	b.SetInsertPointBefore(old)
	call := adapterOuterCall(b, helper, args, destination, unwind, bundles)
	call.SetInstructionCallConv(helper.FunctionCallConv())
	call.InstructionSetDebugLoc(old.InstructionDebugLoc())
	if !call.IsACallInst().IsNil() {
		call.SetTailCallKind(llvm.TailCallKindNoTail)
	}
	if !continuation.IsNil() {
		b.SetInsertPointAtEnd(continuation)
	}
	if !old.FirstUse().IsNil() {
		result := call
		if privateRet != ret {
			parts := []llvm.Value{call}
			if privateRet.TypeKind() == llvm.StructTypeKind {
				parts = make([]llvm.Value, privateRet.StructElementTypesCount())
				for i := range parts {
					parts[i] = b.CreateExtractValue(call, i, "")
				}
			}
			result = a.joinArgument(b, ret, &parts)
		}
		old.ReplaceAllUsesWith(result)
	}
	if !continuation.IsNil() {
		b.CreateBr(normal)
	}
	old.EraseFromParentAsInstruction()
}

// Print just the two signatures in their own module. LLVM includes the full
// definitions of referenced types (also inside byval/sret), attribute groups
// and numbered unnamed types, independent of the caller's module.
func (key adapterKey) symbol(mod llvm.Module, call llvm.Value) string {
	contract := mod.Context().NewModule("regcall.v1")
	defer contract.Dispose()
	contract.SetTarget(mod.Target())
	contract.SetDataLayout(mod.DataLayout())
	target := llvm.AddFunction(contract, "target", key.target)
	target.SetFunctionCallConv(key.cc)
	if !key.varargs.IsNil() {
		// Attributes on variadic operands need a corresponding parameter.
		params := append(key.target.ParamTypes(), key.varargs.ParamTypes()...)
		target = llvm.AddFunction(contract, "actual", llvm.FunctionType(key.target.ReturnType(), params, false))
	}
	visitAdapterAttrs(call, target.AddAttributeAtIndex)
	llvm.AddFunction(contract, "transport", key.transport)
	return fmt.Sprintf("__llgo_fmv_regcall.v1.%x", sha256.Sum256([]byte(contract.String())))
}

func (a *wideABI) newRegisterAdapter(old llvm.Value, key adapterKey) llvm.Value {
	ctx := a.mod.Context()
	name := key.symbol(a.mod, old)
	fn := llvm.AddFunction(a.mod, name, key.transport)
	if fn.Name() != name {
		// Distinct unnamed LLVM types in this module can print identically.
		// Keep the nominally typed second definition local.
		fn.SetLinkage(llvm.InternalLinkage)
	} else {
		fn.SetLinkage(llvm.LinkOnceODRLinkage)
		fn.SetVisibility(llvm.HiddenVisibility)
		if !xtllvm.IsDarwinTarget(a.mod.Target()) {
			group := a.mod.Comdat(name)
			group.SetSelectionKind(llvm.AnyComdatSelectionKind)
			fn.SetComdat(group)
		}
	}
	fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	enableWidth(fn, a.signature(key.target))
	a.abi.applyWideCallingConvention(fn)
	a.markAdapterFrame(fn)

	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
	parts := fn.Params()[1:]
	args := make([]llvm.Value, old.ArgOperandsCount())
	for i := range args {
		args[i] = a.joinArgument(b, old.Operand(i).Type(), &parts)
	}
	call := b.CreateCall(key.target, fn.Param(0), args, "")
	call.SetInstructionCallConv(key.cc)
	visitAdapterAttrs(old, call.AddCallSiteAttribute)
	// A Win64 overflow argument may point into this frame. A normal call
	// keeps the copy alive and the result unpacking on the same boundary.
	call.SetTailCallKind(llvm.TailCallKindNoTail)
	ret := key.target.ReturnType()
	switch {
	case ret.TypeKind() == llvm.VoidTypeKind:
		b.CreateRetVoid()
	case ret != key.transport.ReturnType():
		b.CreateRet(packResult(b, a.splitArgument(b, call, nil)))
	default:
		b.CreateRet(call)
	}
	return fn
}

func packResult(b llvm.Builder, parts []llvm.Value) llvm.Value {
	if len(parts) == 1 {
		return parts[0]
	}
	result := llvm.Undef(parts[0].Type().Context().StructType(valueTypes(parts), false))
	for i, part := range parts {
		result = b.CreateInsertValue(result, part, i, "")
	}
	return result
}

func valueTypes(values []llvm.Value) []llvm.Type {
	types := make([]llvm.Type, len(values))
	for i, v := range values {
		types[i] = v.Type()
	}
	return types
}

// On a split invoke edge all PHIs in the normal destination acquire the same
// new predecessor. Other incoming edges, including backedges, retain their values.
func replacePhiPredecessor(b llvm.Builder, dest, old, next llvm.BasicBlock) {
	for phi := dest.FirstInstruction(); !phi.IsNil() && !phi.IsAPHINode().IsNil(); {
		after := llvm.NextInstruction(phi)
		values := make([]llvm.Value, phi.IncomingCount())
		blocks := make([]llvm.BasicBlock, len(values))
		for i := range values {
			values[i], blocks[i] = phi.IncomingValue(i), phi.IncomingBlock(i)
			if blocks[i] == old {
				blocks[i] = next
			}
		}
		b.SetInsertPointBefore(phi)
		replacement := b.CreatePHI(phi.Type(), "")
		replacement.AddIncoming(values, blocks)
		copyAdapterPHIProperties(phi, replacement)
		phi.ReplaceAllUsesWith(replacement)
		phi.EraseFromParentAsInstruction()
		phi = after
	}
}

// splitArgument and joinArgument use raw lane bits, independent of the Go
// element type. Tuples left by ordinary ABI lowering are handled recursively.
func (a *wideABI) splitArgument(b llvm.Builder, v llvm.Value, out []llvm.Value) []llvm.Value {
	t := v.Type()
	if a.splitTuple(t) {
		for i := 0; i < aggregateCount(t); i++ {
			out = a.splitArgument(b, b.CreateExtractValue(v, i, ""), out)
		}
		return out
	}
	w := a.valueWidth(t)
	if w == 0 {
		if narrow := outlineType(t); narrow != t {
			v = b.CreateBitCast(v, narrow, "")
		}
		return append(out, v)
	}
	raw := llvm.VectorType(t.Context().Int32Type(), w/32)
	if t.ElementType().TypeKind() == llvm.PointerTypeKind {
		raw = t // Keep pointer lanes as pointers throughout the shuffle.
	}
	bits := b.CreateBitCast(v, raw, "")
	lanes := raw.VectorSize() / (w / 128)
	for i := 0; i < raw.VectorSize(); i += lanes {
		out = append(out, b.CreateShuffleVector(bits, llvm.Undef(raw), laneRange(t.Context(), i, lanes), ""))
	}
	return out
}

func (a *wideABI) joinArgument(b llvm.Builder, t llvm.Type, parts *[]llvm.Value) llvm.Value {
	if a.splitTuple(t) {
		result := llvm.Undef(t)
		for i := 0; i < aggregateCount(t); i++ {
			element := b.CreateExtractValue(llvm.Undef(t), i, "").Type()
			result = b.CreateInsertValue(result, a.joinArgument(b, element, parts), i, "")
		}
		return result
	}
	w := a.valueWidth(t)
	if w == 0 {
		v := (*parts)[0]
		*parts = (*parts)[1:]
		if t != v.Type() {
			v = b.CreateBitCast(v, t, "")
		}
		return v
	}
	p := (*parts)[:w/128]
	*parts = (*parts)[len(p):]
	join := func(x, y llvm.Value) llvm.Value {
		return b.CreateShuffleVector(x, y, laneRange(t.Context(), 0, 2*x.Type().VectorSize()), "")
	}
	bits := join(p[0], p[1])
	if len(p) == 4 {
		bits = join(bits, join(p[2], p[3]))
	}
	return b.CreateBitCast(bits, t, "")
}

// Ordinary aggregates keep their lowered type. Split the ones containing
// a wide register value or a predicate whose carrier changes with AVX-512.
func (a *wideABI) splitTuple(t llvm.Type) bool {
	switch t.TypeKind() {
	case llvm.ArrayTypeKind:
		return t.ArrayLength() != 0 && a.changesCarrier(t.ElementType())
	case llvm.StructTypeKind:
		for _, field := range t.StructElementTypes() {
			if a.changesCarrier(field) {
				return true
			}
		}
	}
	return false
}

func (a *wideABI) changesCarrier(t llvm.Type) bool {
	return a.splitTuple(t) || t.TypeKind() == llvm.VectorTypeKind && (a.valueWidth(t) != 0 || outlineType(t) != t)
}

func aggregateCount(t llvm.Type) int {
	if t.TypeKind() == llvm.ArrayTypeKind {
		return t.ArrayLength()
	}
	return t.StructElementTypesCount()
}

func laneRange(ctx llvm.Context, start, count int) llvm.Value {
	lanes := make([]llvm.Value, count)
	for i := range lanes {
		lanes[i] = llvm.ConstInt(ctx.Int32Type(), uint64(start+i), false)
	}
	return llvm.ConstVector(lanes, false)
}

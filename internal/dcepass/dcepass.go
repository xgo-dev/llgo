// Package dcepass rewrites ABI metadata so link-time dead code elimination can
// drop method bodies that are no longer referenced by live method slots.
package dcepass

import (
	"fmt"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/meta"
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llvm"
)

// EmitInterfaceUseOverrides resolves predicates only after the complete
// interface domain is known. A nil predicate keeps every assertion possible.
// Constant-returning functions also fold during final Wasm optimization without
// LTO; linear-memory flags cannot be treated as immutable by that optimizer.
func EmitInterfaceUseOverrides(dst llvm.Module, srcMods []llvm.Module, reject func(string) bool) {
	seen := make(map[string]bool)
	builder := dst.Context().NewBuilder()
	defer builder.Dispose()
	for _, src := range srcMods {
		for fn := src.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			name := fn.Name()
			typ, ok := strings.CutPrefix(name, meta.InterfaceUsePrefix)
			if !ok || seen[name] {
				continue
			}
			seen[name] = true
			value := uint64(1)
			if reject != nil && reject(typ) {
				value = 0
			}
			result := dst.Context().IntType(fn.GlobalValueType().ReturnType().IntTypeWidth())
			target := dst.NamedFunction(name)
			if target.IsNil() {
				target = llvm.AddFunction(dst, name, llvm.FunctionType(result, nil, false))
			}
			if target.BasicBlocksCount() != 0 {
				continue
			}
			builder.SetInsertPointAtEnd(dst.Context().AddBasicBlock(target, "entry"))
			builder.CreateRet(llvm.ConstInt(result, value, false))
		}
	}
}

const (
	abiMethodTypeName     = "github.com/xgo-dev/llgo/runtime/abi.Method"
	unreachableMethodName = "github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod"
)

// EmitRegularTypeFieldOverrides removes inaccessible field tables after method
// overrides. Regular-memory types use byte-wise equality and hashing, so neither
// needs the table. Keep descriptor prefixes and any earlier method compaction.
func EmitRegularTypeFieldOverrides(dst llvm.Module, sources []llvm.Module, drop func(string) bool) {
	if drop == nil {
		return
	}
	emitter := newOverrideEmitter(dst)
	seen := make(map[string]bool)
	for _, src := range sources {
		for g := src.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
			if seen[g.Name()] || !g.IsGlobalConstant() || !drop(g.Name()) {
				continue
			}
			trimmed, changed := trimRegularTypeFields(g.Initializer())
			if !changed {
				continue
			}
			seen[g.Name()] = true
			target := dst.NamedGlobal(g.Name())
			if target.IsNil() {
				target = llvm.AddGlobal(dst, emitter.cloneType(g.GlobalValueType()), g.Name())
			}
			value := target.Initializer()
			if value.IsNil() {
				emitter.values[g] = target
				value = emitter.cloneConst(trimmed)
			} else {
				value, _ = trimRegularTypeFields(value)
			}
			target.SetInitializer(value)
			target.SetGlobalConstant(true)
			target.SetLinkage(llvm.ExternalLinkage)
			copyGlobalAttrs(target, g)
		}
	}
}

func trimRegularTypeFields(value llvm.Value) (llvm.Value, bool) {
	if value.IsNil() || value.IsAConstantStruct().IsNil() || value.OperandsCount() == 0 {
		return value, false
	}
	fields := make([]llvm.Value, value.OperandsCount())
	for i := range fields {
		fields[i] = value.Operand(i)
	}
	if value.Type().StructName() == "github.com/xgo-dev/llgo/runtime/abi.StructType" {
		if len(fields) != 3 {
			return value, false
		}
		common := fields[0]
		if common.OperandsCount() < 4 || common.Operand(3).IsAConstantInt().IsNil() ||
			common.Operand(3).ZExtValue()&uint64(abi.TFlagRegularMemory) == 0 || fields[2].IsNull() {
			return value, false
		}
		fields[2] = llvm.ConstNull(fields[2].Type())
	} else {
		first, changed := trimRegularTypeFields(fields[0])
		if !changed {
			return value, false
		}
		fields[0] = first
	}
	return constStructOfType(value.Type(), fields), true
}

// EmitStrongTypeOverrides emits method-pruned strong ABI type symbols into dst.
// srcMods contains the original package modules. For each constant ABI type
// global with a method array, it creates a same-name strong global in dst and
// clears IFn/TFn for method slots not listed in liveSlots[typeName].
// dropMetadata can remove method storage for types proven invisible to
// interfaces and reflection; nil preserves them. verbose logs removed slots.
func EmitStrongTypeOverrides(dst llvm.Module, srcMods []llvm.Module, liveSlots map[string][]int, dropMetadata func(string) bool, verbose bool) {
	emitted := make(map[string]bool)
	emitter := newOverrideEmitter(dst)
	type override struct {
		src, dst, methods llvm.Value
		elem              llvm.Type
		keep              map[int]bool
		compact           bool
	}
	var overrides []override
	var replaced [][2]llvm.Value
	for _, src := range srcMods {
		for g := src.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
			name := g.Name()
			if emitted[name] || !g.IsGlobalConstant() {
				continue
			}
			methodsVal, elemTy, ok := methodArray(g.Initializer())
			if !ok {
				continue
			}
			drop := dropMetadata != nil && dropMetadata(name)
			keep := liveSlotSet(liveSlots[name])
			init := g.Initializer()
			compact := drop && len(keep) == 0 && init.OperandsCount() == 3 &&
				init.Operand(1).Type().TypeKind() == llvm.StructTypeKind &&
				init.Operand(1).Type().StructName() == "github.com/xgo-dev/llgo/runtime/abi.UncommonType"
			typ := emitter.cloneType(g.GlobalValueType())
			if compact {
				fields := typ.StructElementTypes()
				fields[2] = llvm.ArrayType(emitter.cloneType(elemTy), 0)
				typ = dst.Context().StructType(fields, typ.IsStructPacked())
			}
			old := dst.NamedGlobal(name)
			target := old
			if old.IsNil() {
				target = llvm.AddGlobal(dst, typ, name)
			} else if old.GlobalValueType() != typ {
				target = llvm.AddGlobal(dst, typ, "")
				emitter.values[old] = target
				replaced = append(replaced, [2]llvm.Value{old, target})
			}
			emitter.values[g] = target
			overrides = append(overrides, override{g, target, methodsVal, elemTy, keep, compact})
			emitted[name] = true
		}
	}
	// Reserve every descriptor before cloning references between descriptors.
	// Keep replaced globals alive until all initializers have been cloned: dst
	// may also be a source module, and RAUW can invalidate source constants.
	for _, o := range overrides {
		emitter.emitTypeOverride(o.src, o.dst, o.methods, o.elem, o.keep, o.compact, verbose)
	}
	for _, pair := range replaced {
		name := pair[0].Name()
		pair[0].ReplaceAllUsesWith(pair[1])
		pair[0].EraseFromParentAsGlobal()
		pair[1].SetName(name)
	}
}

type overrideEmitter struct {
	dst    llvm.Module
	values map[llvm.Value]llvm.Value
	types  map[llvm.Type]llvm.Type
}

func newOverrideEmitter(dst llvm.Module) *overrideEmitter {
	return &overrideEmitter{
		dst:    dst,
		values: make(map[llvm.Value]llvm.Value),
		types:  make(map[llvm.Type]llvm.Type),
	}
}

func (e *overrideEmitter) emitTypeOverride(srcType, dstType, methodsVal llvm.Value, elemTy llvm.Type, keepIdx map[int]bool, compact, verbose bool) {
	init := srcType.Initializer()

	fieldCount := init.OperandsCount()
	fields := make([]llvm.Value, fieldCount)
	for i := 0; i < fieldCount-1; i++ {
		fields[i] = e.cloneConst(init.Operand(i))
	}
	// Preserve prefix offsets and a valid methodless uncommon header, while
	// removing the inaccessible trailing array and its transitive references.
	if compact {
		uncommon := fields[1]
		parts := make([]llvm.Value, uncommon.OperandsCount())
		for i := range parts {
			parts[i] = uncommon.Operand(i)
		}
		parts[1], parts[2] = llvm.ConstNull(parts[1].Type()), llvm.ConstNull(parts[2].Type())
		fields[1] = constStructOfType(uncommon.Type(), parts)
		fields[2] = llvm.ConstNull(dstType.GlobalValueType().StructElementTypes()[2])
		dstType.SetInitializer(constStructOfType(dstType.GlobalValueType(), fields))
		dstType.SetGlobalConstant(true)
		dstType.SetLinkage(llvm.ExternalLinkage)
		copyGlobalAttrs(dstType, srcType)
		if verbose {
			fmt.Fprintf(os.Stderr, "[dce] drop unobserved method metadata %s\n", srcType.Name())
		}
		return
	}

	unreachableMethod := e.unreachableMethod()
	dstElemTy := e.cloneType(elemTy)
	methodFields := dstElemTy.StructElementTypes()
	methods := make([]llvm.Value, methodsVal.OperandsCount())
	for i := range methods {
		orig := methodsVal.Operand(i)
		if keepIdx[i] {
			methods[i] = e.cloneConst(orig)
			continue
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "[dce] drop method %s[%d] ifn=%s tfn=%s\n", srcType.Name(), i, methodPointerName(orig.Operand(2)), methodPointerName(orig.Operand(3)))
		}
		name := e.cloneConst(orig.Operand(0))
		mtype := e.cloneConst(orig.Operand(1))
		methods[i] = llvm.ConstNamedStruct(dstElemTy, []llvm.Value{
			name,
			mtype,
			methodPointerConstant(methodFields[2], unreachableMethod),
			methodPointerConstant(methodFields[3], unreachableMethod),
		})
	}
	fields[fieldCount-1] = llvm.ConstArray(dstElemTy, methods)

	dstType.SetInitializer(constStructOfType(e.cloneType(init.Type()), fields))
	dstType.SetGlobalConstant(true)
	dstType.SetLinkage(llvm.ExternalLinkage)
	copyGlobalAttrs(dstType, srcType)
}

func methodPointerName(v llvm.Value) string {
	if !v.IsAConstantStruct().IsNil() && v.OperandsCount() == 2 {
		v = v.Operand(0)
	}
	return v.Name()
}

// methodPointerConstant uses the slot's existing storage type. On wasm32 the
// 32-bit pointer is stored in a wider Go slot: { ptr, i32 }, with zero padding.
// Do not infer the layout from the target or change other uses of the method.
func methodPointerConstant(typ llvm.Type, fn llvm.Value) llvm.Value {
	if typ == fn.Type() {
		return fn
	}
	if typ.TypeKind() == llvm.StructTypeKind {
		fields := typ.StructElementTypes()
		if len(fields) == 2 && fields[0] == fn.Type() && fields[1].TypeKind() == llvm.IntegerTypeKind {
			return constStructOfType(typ, []llvm.Value{fn, llvm.ConstNull(fields[1])})
		}
	}
	panic(fmt.Sprintf("dcepass: unsupported method pointer storage type %s", typ))
}

func (e *overrideEmitter) unreachableMethod() llvm.Value {
	fn := e.dst.NamedFunction(unreachableMethodName)
	if fn.IsNil() {
		fn = llvm.AddFunction(e.dst, unreachableMethodName,
			llvm.FunctionType(e.dst.Context().VoidType(), nil, false))
	}
	return fn
}

func (e *overrideEmitter) cloneConst(v llvm.Value) llvm.Value {
	if mapped, ok := e.values[v]; ok {
		return mapped
	}
	if gv := v.IsAGlobalValue(); !gv.IsNil() {
		return e.cloneGlobalValue(gv)
	}
	dstTy := e.cloneType(v.Type())
	if v.IsNull() || !v.IsAConstantAggregateZero().IsNil() {
		return llvm.ConstNull(dstTy)
	}
	if !v.IsAUndefValue().IsNil() {
		return llvm.Undef(dstTy)
	}
	if !v.IsAConstantInt().IsNil() {
		return llvm.ConstInt(dstTy, v.ZExtValue(), false)
	}
	if !v.IsAConstantFP().IsNil() {
		return llvm.ConstFloatFromBits(dstTy, v.FloatBits())
	}
	if v.IsConstantString() {
		return e.dst.Context().ConstString(v.ConstGetAsString(), false)
	}
	if !v.IsAConstantStruct().IsNil() {
		clone := constStructOfType(dstTy, e.cloneOperands(v))
		e.values[v] = clone
		return clone
	}
	if !v.IsAConstantArray().IsNil() {
		clone := llvm.ConstArray(dstTy.ElementType(), e.cloneOperands(v))
		e.values[v] = clone
		return clone
	}
	if !v.IsAConstantVector().IsNil() {
		clone := llvm.ConstVector(e.cloneOperands(v), false)
		e.values[v] = clone
		return clone
	}
	if !v.IsAConstantExpr().IsNil() {
		return e.cloneConstExpr(v, dstTy)
	}
	panic(fmt.Sprintf("dcepass: unsupported constant %s", v.String()))
}

func (e *overrideEmitter) cloneConstExpr(v llvm.Value, dstTy llvm.Type) llvm.Value {
	ops := e.cloneOperands(v)
	var clone llvm.Value
	switch v.Opcode() {
	case llvm.GetElementPtr:
		clone = llvm.ConstGEP(e.cloneType(v.GEPSourceElementType()), ops[0], ops[1:])
	case llvm.BitCast:
		clone = llvm.ConstBitCast(ops[0], dstTy)
	case llvm.IntToPtr:
		clone = llvm.ConstIntToPtr(ops[0], dstTy)
	case llvm.PtrToInt:
		clone = llvm.ConstPtrToInt(ops[0], dstTy)
	case llvm.Trunc:
		clone = llvm.ConstTrunc(ops[0], dstTy)
	case llvm.Add:
		clone = llvm.ConstAdd(ops[0], ops[1])
	case llvm.Sub:
		clone = llvm.ConstSub(ops[0], ops[1])
	case llvm.Xor:
		clone = llvm.ConstXor(ops[0], ops[1])
	default:
		panic(fmt.Sprintf("dcepass: unsupported constant expression %s", v.String()))
	}
	e.values[v] = clone
	return clone
}

// cloneType re-interns an LLVM type in the destination module's Context.
// Returning a source-context type is invalid even when its printed spelling is
// identical; LLVM otherwise permits malformed mixed-context constants that
// fail verification and can crash while either Context is disposed.
func (e *overrideEmitter) cloneType(src llvm.Type) llvm.Type {
	if dst, ok := e.types[src]; ok {
		return dst
	}
	ctx := e.dst.Context()
	var dst llvm.Type
	switch src.TypeKind() {
	case llvm.VoidTypeKind:
		dst = ctx.VoidType()
	case llvm.FloatTypeKind:
		dst = ctx.FloatType()
	case llvm.DoubleTypeKind:
		dst = ctx.DoubleType()
	case llvm.X86_FP80TypeKind:
		dst = ctx.X86FP80Type()
	case llvm.FP128TypeKind:
		dst = ctx.FP128Type()
	case llvm.PPC_FP128TypeKind:
		dst = ctx.PPCFP128Type()
	case llvm.LabelTypeKind:
		dst = ctx.LabelType()
	case llvm.IntegerTypeKind:
		dst = ctx.IntType(src.IntTypeWidth())
	case llvm.FunctionTypeKind:
		params := src.ParamTypes()
		for i := range params {
			params[i] = e.cloneType(params[i])
		}
		dst = llvm.FunctionType(e.cloneType(src.ReturnType()), params, src.IsFunctionVarArg())
	case llvm.StructTypeKind:
		name := src.StructName()
		if name != "" {
			dst = e.dst.GetTypeByName(name)
			if dst.IsNil() {
				dst = ctx.StructCreateNamed(name)
			}
			e.types[src] = dst
			if dst.IsStructOpaque() && !src.IsStructOpaque() {
				dst.StructSetBody(e.cloneTypes(src.StructElementTypes()), src.IsStructPacked())
			}
			return dst
		}
		dst = ctx.StructType(e.cloneTypes(src.StructElementTypes()), src.IsStructPacked())
	case llvm.ArrayTypeKind:
		dst = llvm.ArrayType(e.cloneType(src.ElementType()), src.ArrayLength())
	case llvm.PointerTypeKind:
		// LLVM uses opaque pointers; the element type only selects the Context.
		dst = llvm.PointerType(ctx.Int8Type(), src.PointerAddressSpace())
	case llvm.VectorTypeKind:
		dst = llvm.VectorType(e.cloneType(src.ElementType()), src.VectorSize())
	case llvm.MetadataTypeKind:
		dst = ctx.MetadataType()
	case llvm.TokenTypeKind:
		dst = ctx.TokenType()
	default:
		panic(fmt.Sprintf("dcepass: unsupported LLVM type kind %d", src.TypeKind()))
	}
	e.types[src] = dst
	return dst
}

func (e *overrideEmitter) cloneTypes(src []llvm.Type) []llvm.Type {
	dst := make([]llvm.Type, len(src))
	for i := range src {
		dst[i] = e.cloneType(src[i])
	}
	return dst
}

func (e *overrideEmitter) cloneOperands(v llvm.Value) []llvm.Value {
	ops := make([]llvm.Value, v.OperandsCount())
	for i := range ops {
		ops[i] = e.cloneConst(v.Operand(i))
	}
	return ops
}

func (e *overrideEmitter) cloneGlobalValue(v llvm.Value) llvm.Value {
	// Rebind a source-module function reference to a declaration in dst. The
	// function body remains in its package object and resolves by name at link
	// time; the override initializer only needs a destination-owned reference
	// with the same function type.
	if fn := v.IsAFunction(); !fn.IsNil() {
		dstFn := e.dst.NamedFunction(fn.Name())
		if dstFn.IsNil() {
			dstFn = llvm.AddFunction(e.dst, fn.Name(), e.cloneType(fn.GlobalValueType()))
		}
		e.values[v] = dstFn
		return dstFn
	}
	if gv := v.IsAGlobalVariable(); !gv.IsNil() {
		return e.cloneGlobalVariable(gv)
	}
	panic("dcepass: unsupported global value")
}

func (e *overrideEmitter) cloneGlobalVariable(src llvm.Value) llvm.Value {
	if mapped, ok := e.values[src]; ok {
		return mapped
	}
	name := src.Name()
	if name != "" && !isLocalLinkage(src.Linkage()) {
		dst := e.dst.NamedGlobal(name)
		if dst.IsNil() {
			dst = llvm.AddGlobal(e.dst, e.cloneType(src.GlobalValueType()), name)
			dst.SetLinkage(llvm.ExternalLinkage)
		}
		e.values[src] = dst
		return dst
	}

	dst := llvm.AddGlobal(e.dst, e.cloneType(src.GlobalValueType()), "")
	e.values[src] = dst
	copyGlobalAttrs(dst, src)
	dst.SetLinkage(src.Linkage())
	dst.SetGlobalConstant(src.IsGlobalConstant())
	if init := src.Initializer(); !init.IsNil() {
		dst.SetInitializer(e.cloneConst(init))
	}
	return dst
}

func methodArray(init llvm.Value) (llvm.Value, llvm.Type, bool) {
	if init.IsNil() || init.OperandsCount() == 0 {
		return llvm.Value{}, llvm.Type{}, false
	}
	methodsVal := init.Operand(init.OperandsCount() - 1)
	if methodsVal.Type().TypeKind() != llvm.ArrayTypeKind {
		return llvm.Value{}, llvm.Type{}, false
	}
	elemTy := methodsVal.Type().ElementType()
	if elemTy.TypeKind() != llvm.StructTypeKind || elemTy.StructElementTypesCount() != 4 {
		return llvm.Value{}, llvm.Type{}, false
	}
	if elemTy.StructName() != abiMethodTypeName {
		return llvm.Value{}, llvm.Type{}, false
	}
	return methodsVal, elemTy, true
}

func liveSlotSet(slots []int) map[int]bool {
	out := make(map[int]bool, len(slots))
	for _, slot := range slots {
		out[slot] = true
	}
	return out
}

func copyGlobalAttrs(dst, src llvm.Value) {
	dst.SetVisibility(src.Visibility())
	dst.SetThreadLocal(src.IsThreadLocal())
	if align := src.Alignment(); align > 0 {
		dst.SetAlignment(align)
	}
}

func isLocalLinkage(linkage llvm.Linkage) bool {
	return linkage == llvm.PrivateLinkage || linkage == llvm.InternalLinkage
}

func constStructOfType(typ llvm.Type, fields []llvm.Value) llvm.Value {
	// LLVMConstNamedStruct accepts both identified and literal struct types. Use
	// the exact cloned type: LLVMConstStruct(InContext) may manufacture another
	// structurally identical literal type that is not pointer-identical to a
	// global's declared value type, which the verifier correctly rejects.
	return llvm.ConstNamedStruct(typ, fields)
}

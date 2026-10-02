package ssa

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/xgo-dev/llvm"
)

// SIMDOp identifies semantic operations independently of Go syntax tokens.
type SIMDOp uint8

const (
	SIMDAdd SIMDOp = iota
	SIMDSub
	SIMDAnd
	SIMDOr
	SIMDXor
	SIMDExtractLane
	SIMDInsertLane
	SIMDUnimplemented
	SIMDLoad
	SIMDStore
	SIMDBroadcast
	SIMDSplatLane0
	SIMDMul
	SIMDDiv
	SIMDAndNot
	SIMDOrNot
	SIMDNot
	SIMDNeg
	SIMDAbs
	SIMDSqrt
	SIMDCeil
	SIMDFloor
	SIMDTrunc
	SIMDRound
	SIMDBitcast
	SIMDEqual
	SIMDNotEqual
	SIMDLess
	SIMDLessEqual
	SIMDGreater
	SIMDGreaterEqual
	SIMDToMask
	SIMDBitSelect
	SIMDBitSelectNot
	SIMDBlend
	SIMDMaskFromBits
	SIMDMaskToBits
	SIMDShiftAllLeft
	SIMDShiftAllRight
	SIMDShiftLeft
	SIMDShiftRight
	SIMDShift
	SIMDAddSaturated
	SIMDSubSaturated
	SIMDMin
	SIMDMax
)

// SIMDNumericShape validates the official numeric aggregate representation.
// Keep storage knowledge here, separate from operation selection and features.
func SIMDNumericShape(typ types.Type) (*types.Array, bool) {
	lanes, ok := SIMDVectorShape(typ)
	return lanes, ok && !strings.HasPrefix(types.Unalias(typ).(*types.Named).Obj().Name(), "Mask")
}

// SIMDMaskShape recognizes the four SIMD128 masks. Masks use canonical zero
// or all-one integer lanes, preserving their public Go storage representation.
func SIMDMaskShape(typ types.Type) (*types.Array, bool) {
	lanes, ok := SIMDVectorShape(typ)
	return lanes, ok && strings.HasPrefix(types.Unalias(typ).(*types.Named).Obj().Name(), "Mask")
}

// SIMDVectorShape validates numeric and mask SIMD128 storage.
func SIMDVectorShape(typ types.Type) (*types.Array, bool) {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != "simd/archsimd" {
		return nil, false
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok || st.NumFields() != 2 {
		return nil, false
	}
	lanes, ok := st.Field(1).Type().(*types.Array)
	if !ok {
		return nil, false
	}
	elem, ok := lanes.Elem().(*types.Basic)
	if !ok {
		return nil, false
	}
	var bits int64
	switch elem.Kind() {
	case types.Int8, types.Uint8:
		bits = 8
	case types.Int16, types.Uint16:
		bits = 16
	case types.Int32, types.Uint32, types.Float32:
		bits = 32
	case types.Int64, types.Uint64, types.Float64:
		bits = 64
	default:
		return nil, false
	}
	name := elem.Name()
	expected := fmt.Sprintf("%s%sx%d", strings.ToUpper(name[:1]), name[1:], lanes.Len())
	if strings.HasPrefix(named.Obj().Name(), "Mask") && elem.Info()&types.IsInteger != 0 && elem.Info()&types.IsUnsigned == 0 {
		expected = fmt.Sprintf("Mask%dx%d", bits, lanes.Len())
	}
	if named.Obj().Name() != expected || lanes.Len()*bits != 128 {
		return nil, false
	}
	tag, ok := st.Field(0).Type().(*types.Named)
	if !ok || tag.Obj().Pkg() != named.Obj().Pkg() || tag.Obj().Name() != "v128" {
		return nil, false
	}
	if (&types.StdSizes{WordSize: 8, MaxAlign: 8}).Sizeof(tag) != 0 {
		return nil, false
	}
	return lanes, true
}

func simdLanes(typ types.Type) *types.Array {
	lanes, ok := SIMDVectorShape(typ)
	if !ok {
		panic("unsupported SIMD numeric storage: " + typ.String())
	}
	return lanes
}

// SIMD lowers a validated source operation. LLVM legalizes these operations
// for the native baseline; wasm needs the SIMD128 feature. The result type is
// explicit for loads and conversions whose result differs from their operands.
func (b Builder) SIMD(op SIMDOp, result Type, args ...Expr) Expr {
	if op == SIMDUnimplemented {
		if len(args) != 1 || !types.Identical(args[0].RawType(), types.Typ[types.String]) {
			panic("SIMD fallback requires an intrinsic name string")
		}
		return b.Call(b.Pkg.rtFunc("PanicSIMDUnimplemented"), args...)
	}
	b.simdFeatures(op)
	if op == SIMDRound && b.Prog.Target().GOARCH == "amd64" {
		return b.simdRoundEven(args[0])
	}
	if name, ok := simdFloatUnary[op]; ok {
		v := b.impl.CreateIntrinsic(result.ll, llvm.LookupIntrinsicID(name), []llvm.Value{args[0].impl}, "")
		return Expr{v, result}
	}
	switch op {
	case SIMDShiftAllLeft, SIMDShiftAllRight, SIMDShiftLeft, SIMDShiftRight, SIMDShift:
		return b.simdShift(op, args[0], args[1])
	case SIMDMin, SIMDMax:
		if simdLanes(args[0].RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsFloat != 0 {
			return b.simdFloatMinMax(op, args[0], args[1])
		}
		return b.simdIntegerIntrinsic(op, args[0], args[1])
	case SIMDAddSaturated, SIMDSubSaturated:
		return b.simdIntegerIntrinsic(op, args[0], args[1])
	case SIMDMaskFromBits:
		n := int(simdLanes(result.RawType()).Len())
		bits := b.impl.CreateTrunc(args[0].impl, b.Prog.ctx.IntType(n), "")
		mask := b.impl.CreateBitCast(bits, llvm.VectorType(b.Prog.Bool().ll, n), "")
		return Expr{b.impl.CreateSExt(mask, result.ll, ""), result}
	case SIMDMaskToBits:
		n := int(simdLanes(args[0].RawType()).Len())
		mask := llvm.CreateICmp(b.impl, llvm.IntNE, args[0].impl, llvm.ConstNull(args[0].ll))
		bits := b.impl.CreateBitCast(mask, b.Prog.ctx.IntType(n), "")
		return Expr{b.impl.CreateZExt(bits, result.ll, ""), result}
	case SIMDEqual, SIMDNotEqual, SIMDLess, SIMDLessEqual, SIMDGreater, SIMDGreaterEqual:
		return b.simdCompare(op, result, args[0], args[1])
	case SIMDToMask:
		cond := llvm.CreateICmp(b.impl, llvm.IntNE, args[0].impl, llvm.ConstNull(args[0].ll))
		return Expr{b.impl.CreateSExt(cond, result.ll, ""), result}
	case SIMDBitSelect, SIMDBitSelectNot, SIMDBlend:
		x, y, mask := args[0].impl, args[1].impl, args[2].impl
		if op == SIMDBlend {
			cond := llvm.CreateICmp(b.impl, llvm.IntSLT, mask, llvm.ConstNull(mask.Type()))
			return Expr{b.impl.CreateSelect(cond, y, x, ""), result}
		}
		if op == SIMDBitSelectNot {
			x, y = y, x
		}
		v := b.impl.CreateOr(b.impl.CreateAnd(x, mask, ""), b.impl.CreateAnd(y, b.impl.CreateNot(mask, ""), ""), "")
		return Expr{v, result}
	case SIMDBitcast:
		return Expr{b.impl.CreateBitCast(args[0].impl, result.ll, ""), result}
	case SIMDNeg, SIMDNot, SIMDAbs:
		return b.simdUnary(op, args[0])
	case SIMDLoad:
		ptr := args[0]
		b.AssertNilDeref(ptr)
		v := llvm.CreateLoad(b.impl, result.ll, ptr.impl)
		v.SetAlignment(int(b.Prog.AlignOf(b.Prog.Elem(ptr.Type))))
		return Expr{v, result}
	case SIMDStore:
		ptr := args[1]
		b.AssertNilDeref(ptr)
		v := b.impl.CreateStore(args[0].impl, ptr.impl)
		v.SetAlignment(int(b.Prog.AlignOf(b.Prog.Elem(ptr.Type))))
		return Expr{v, b.Prog.Void()}
	case SIMDBroadcast:
		v := b.impl.CreateInsertElement(llvm.Undef(result.ll), args[0].impl, llvm.ConstInt(b.Prog.tyInt32(), 0, false), "")
		return b.simdSplatLane0(Expr{v, result})
	case SIMDSplatLane0:
		return b.simdSplatLane0(args[0])
	case SIMDExtractLane:
		return b.simdGetElem(args[0], args[1])
	case SIMDInsertLane:
		return b.simdSetElem(args[0], args[1], args[2])
	case SIMDAdd, SIMDSub, SIMDMul, SIMDDiv, SIMDAnd, SIMDOr, SIMDXor, SIMDAndNot, SIMDOrNot:
		return b.simdBinary(op, args[0], args[1])
	default:
		panic("unsupported SIMD operation")
	}
}

func (b Builder) simdCompare(op SIMDOp, result Type, x, y Expr) Expr {
	info := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()
	var cond llvm.Value
	if info&types.IsFloat != 0 {
		pred := map[SIMDOp]llvm.FloatPredicate{
			SIMDEqual: llvm.FloatOEQ, SIMDNotEqual: llvm.FloatUNE,
			SIMDLess: llvm.FloatOLT, SIMDLessEqual: llvm.FloatOLE,
			SIMDGreater: llvm.FloatOGT, SIMDGreaterEqual: llvm.FloatOGE,
		}[op]
		cond = b.impl.CreateFCmp(pred, x.impl, y.impl, "")
	} else {
		pred := map[SIMDOp]llvm.IntPredicate{
			SIMDEqual: llvm.IntEQ, SIMDNotEqual: llvm.IntNE,
			SIMDLess: llvm.IntSLT, SIMDLessEqual: llvm.IntSLE,
			SIMDGreater: llvm.IntSGT, SIMDGreaterEqual: llvm.IntSGE,
		}[op]
		if info&types.IsUnsigned != 0 {
			switch pred {
			case llvm.IntSLT:
				pred = llvm.IntULT
			case llvm.IntSLE:
				pred = llvm.IntULE
			case llvm.IntSGT:
				pred = llvm.IntUGT
			case llvm.IntSGE:
				pred = llvm.IntUGE
			}
		}
		cond = llvm.CreateICmp(b.impl, pred, x.impl, y.impl)
	}
	return Expr{b.impl.CreateSExt(cond, result.ll, ""), result}
}

var simdFloatUnary = map[SIMDOp]string{
	SIMDSqrt: "llvm.sqrt", SIMDCeil: "llvm.ceil", SIMDFloor: "llvm.floor",
	SIMDTrunc: "llvm.trunc", SIMDRound: "llvm.roundeven",
}

func (b Builder) simdFeatures(op SIMDOp) {
	b.requireSIMDFeatures()
}

func (b Builder) requireSIMDFeatures() {
	if b.Prog.Target().GOARCH == "wasm" {
		features := "+simd128"
		for _, attr := range b.Func.impl.GetFunctionAttributes() {
			if attr.IsString() && attr.GetStringKind() == "target-features" {
				features = attr.GetStringValue()
				if !strings.Contains(","+features+",", ",+simd128,") {
					features += ",+simd128"
				}
			}
		}
		b.Func.impl.AddFunctionAttr(b.Prog.ctx.CreateStringAttribute("target-features", features))
	}
}

// SIMD values stay vectors. Aggregate conversion is restricted to Go storage
// and source-level access to the underlying fields inside archsimd.
func (b Builder) simdFromStorage(value llvm.Value, typ Type) llvm.Value {
	lanes := simdLanes(typ.RawType())
	array := b.impl.CreateExtractValue(value, 1, "")
	vec := llvm.Undef(typ.ll)
	for i := 0; i < int(lanes.Len()); i++ {
		lane := b.impl.CreateExtractValue(array, i, "")
		vec = b.impl.CreateInsertElement(vec, lane, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
	}
	return vec
}

func (b Builder) simdToStorage(value llvm.Value, typ Type) llvm.Value {
	storage := b.Prog.storageType(typ)
	lanes := simdLanes(typ.RawType())
	array := llvm.Undef(b.Prog.rawType(lanes).ll)
	for i := 0; i < int(lanes.Len()); i++ {
		lane := b.impl.CreateExtractElement(value, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
		array = b.impl.CreateInsertValue(array, lane, i, "")
	}
	return b.impl.CreateInsertValue(llvm.ConstNull(storage), array, 1, "")
}

// simdBinary uses the actual element type for integer and floating arithmetic.
func (b Builder) simdBinary(op SIMDOp, x, y Expr) Expr {
	a, c := x.impl, y.impl
	floating := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsFloat != 0
	var v llvm.Value
	switch op {
	case SIMDAdd:
		if floating {
			v = b.impl.CreateFAdd(a, c, "")
		} else {
			v = b.impl.CreateAdd(a, c, "")
		}
	case SIMDSub:
		if floating {
			v = b.impl.CreateFSub(a, c, "")
		} else {
			v = b.impl.CreateSub(a, c, "")
		}
	case SIMDMul:
		if floating {
			v = b.impl.CreateFMul(a, c, "")
		} else {
			v = b.impl.CreateMul(a, c, "")
		}
	case SIMDDiv:
		v = b.impl.CreateFDiv(a, c, "")
	case SIMDAndNot:
		v = b.impl.CreateAnd(a, b.impl.CreateNot(c, ""), "")
	case SIMDOrNot:
		v = b.impl.CreateOr(a, b.impl.CreateNot(c, ""), "")
	case SIMDAnd:
		v = b.impl.CreateAnd(a, c, "")
	case SIMDOr:
		v = b.impl.CreateOr(a, c, "")
	case SIMDXor:
		v = b.impl.CreateXor(a, c, "")
	default:
		panic("invalid SIMD128 binary operation")
	}
	return Expr{v, x.Type}
}

func (b Builder) simd128Index(x, index Expr) llvm.Value {
	lanes := simdLanes(x.RawType()).Len()
	if c := index.impl.IsAConstantInt(); !c.IsNil() && c.ZExtValue() < uint64(lanes) {
		return llvm.ConstInt(b.Prog.tyInt32(), c.ZExtValue(), false)
	}
	// Check even when ordinary slice bounds checks are disabled: an invalid
	// immediate is an intrinsic error and must never become LLVM poison.
	bad := Expr{llvm.CreateICmp(b.impl, llvm.IntUGE, index.impl, llvm.ConstInt(index.ll, uint64(lanes), false)), b.Prog.Bool()}
	blocks := b.Func.MakeBlocks(2)
	b.If(bad, blocks[0], blocks[1])
	b.SetBlockEx(blocks[0], AtEnd, false)
	b.Call(b.Pkg.rtFunc("PanicSIMDImmediate"))
	b.Unreachable()
	b.SetBlockEx(blocks[1], AtEnd, false)
	b.blk.last = blocks[1].last
	return b.impl.CreateZExt(index.impl, b.Prog.tyInt32(), "")
}

// Lane indices are uint8 at the Go API boundary. Valid constants need no
// runtime check; dynamic and out-of-range indices retain the panic check.
func (b Builder) simdGetElem(x, index Expr) Expr {
	i := b.simd128Index(x, index)
	v := b.impl.CreateExtractElement(x.impl, i, "")
	elem := simdLanes(x.RawType()).Elem()
	return Expr{v, b.Prog.toType(elem)}
}

func (b Builder) simdSetElem(x, index, value Expr) Expr {
	i := b.simd128Index(x, index)
	v := b.impl.CreateInsertElement(x.impl, value.impl, i, "")
	return Expr{v, x.Type}
}

// A vector ABI also needs SIMD when a function only forwards values, without
// invoking an intrinsic. Inspect the completed IR so wrappers and reflection
// bridges receive the same feature requirement as arithmetic functions.
func (b Builder) finishSIMDFeatures() {
	if b.Prog.Target().GOARCH != "wasm" {
		return
	}
	if llvmTypeHasVector(b.Func.ll) {
		b.requireSIMDFeatures()
		return
	}
	for block := b.Func.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if llvmTypeHasVector(inst.Type()) {
				b.requireSIMDFeatures()
				return
			}
			for i := 0; i < inst.OperandsCount(); i++ {
				if llvmTypeHasVector(inst.Operand(i).Type()) {
					b.requireSIMDFeatures()
					return
				}
			}
		}
	}
}

func llvmTypeHasVector(t llvm.Type) bool {
	switch t.TypeKind() {
	case llvm.VectorTypeKind:
		return true
	case llvm.ArrayTypeKind:
		return llvmTypeHasVector(t.ElementType())
	case llvm.StructTypeKind:
		for _, elem := range t.StructElementTypes() {
			if llvmTypeHasVector(elem) {
				return true
			}
		}
	case llvm.FunctionTypeKind:
		if llvmTypeHasVector(t.ReturnType()) {
			return true
		}
		for _, param := range t.ParamTypes() {
			if llvmTypeHasVector(param) {
				return true
			}
		}
	}
	return false
}

func (b Builder) simdSplatLane0(x Expr) Expr {
	mask := make([]llvm.Value, simdLanes(x.RawType()).Len())
	for i := range mask {
		mask[i] = llvm.ConstInt(b.Prog.tyInt32(), 0, false)
	}
	return Expr{b.impl.CreateShuffleVector(x.impl, llvm.Undef(x.ll), llvm.ConstVector(mask, false), ""), x.Type}
}

func (b Builder) simdUnary(op SIMDOp, x Expr) Expr {
	floating := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsFloat != 0
	var v llvm.Value
	switch op {
	case SIMDNeg:
		if floating {
			v = llvm.CreateFNeg(b.impl, x.impl)
		} else {
			v = llvm.CreateNeg(b.impl, x.impl)
		}
	case SIMDNot:
		v = b.impl.CreateNot(x.impl, "")
	case SIMDAbs:
		if floating {
			v = b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID("llvm.fabs"), []llvm.Value{x.impl}, "")
		} else {
			// Signed minimum keeps its two's-complement bit pattern, never poison.
			v = b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID("llvm.abs"), []llvm.Value{x.impl, llvm.ConstInt(b.Prog.Bool().ll, 0, false)}, "")
		}
	default:
		panic("invalid SIMD unary operation")
	}
	return Expr{v, x.Type}
}

// SSE2 has no rounding instruction. LLVM's roundeven fallback calls a C23
// libm function that is unavailable on some supported systems. Round the
// significand with integer vectors instead, independently of the FP mode.
func (b Builder) simdRoundEven(x Expr) Expr {
	lanes := simdLanes(x.RawType())
	width, fraction, bias := 32, uint64(23), uint64(127)
	if lanes.Elem().Underlying().(*types.Basic).Kind() == types.Float64 {
		width, fraction, bias = 64, 52, 1023
	}
	integer := b.Prog.ctx.IntType(width)
	vector := llvm.VectorType(integer, int(lanes.Len()))
	constant := func(value uint64) llvm.Value {
		values := make([]llvm.Value, lanes.Len())
		for i := range values {
			values[i] = llvm.ConstInt(integer, value, false)
		}
		return llvm.ConstVector(values, false)
	}
	bits := b.impl.CreateBitCast(x.impl, vector, "")
	sign := b.impl.CreateAnd(bits, constant(uint64(1)<<(width-1)), "")
	magnitude := b.impl.CreateAnd(bits, constant((uint64(1)<<(width-1))-1), "")
	exponent := b.impl.CreateLShr(magnitude, constant(fraction), "")
	small := llvm.CreateICmp(b.impl, llvm.IntULT, exponent, constant(bias))
	hasFraction := llvm.CreateICmp(b.impl, llvm.IntULT, exponent, constant(bias+fraction))
	valid := b.impl.CreateAnd(b.impl.CreateNot(small, ""), hasFraction, "")
	// Every shift is in range even in lanes discarded by the final select.
	e := b.impl.CreateSelect(valid, b.impl.CreateSub(exponent, constant(bias), ""), constant(0), "")
	odd := b.impl.CreateAnd(b.impl.CreateLShr(bits, b.impl.CreateSub(constant(fraction), e, ""), ""), constant(1), "")
	increment := b.impl.CreateLShr(b.impl.CreateAdd(constant((uint64(1)<<(fraction-1))-1), odd, ""), e, "")
	mask := b.impl.CreateLShr(constant((uint64(1)<<fraction)-1), e, "")
	rounded := b.impl.CreateAnd(b.impl.CreateAdd(bits, increment, ""), b.impl.CreateNot(mask, ""), "")
	greaterHalf := llvm.CreateICmp(b.impl, llvm.IntUGT, magnitude, constant((bias-1)<<fraction))
	smallResult := b.impl.CreateOr(sign, b.impl.CreateSelect(greaterHalf, constant(bias<<fraction), constant(0), ""), "")
	result := b.impl.CreateSelect(small, smallResult, b.impl.CreateSelect(valid, rounded, bits, ""), "")
	return Expr{b.impl.CreateBitCast(result, x.ll, ""), x.Type}
}

func (b Builder) simdFloatMinMax(op SIMDOp, x, y Expr) Expr {
	if b.Prog.Target().GOARCH == "amd64" {
		// MINPS/MAXPS return the second operand for unordered or equal lanes,
		// including opposite signed zeroes. Keep the operand order intact.
		pred := llvm.FloatOLT
		if op == SIMDMax {
			pred = llvm.FloatOGT
		}
		cond := b.impl.CreateFCmp(pred, x.impl, y.impl, "")
		return Expr{b.impl.CreateSelect(cond, x.impl, y.impl, ""), x.Type}
	}
	name := "llvm.minimum"
	if op == SIMDMax {
		name = "llvm.maximum"
	}
	value := b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID(name), []llvm.Value{x.impl, y.impl}, "")
	return Expr{value, x.Type}
}

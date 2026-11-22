package ssa

import (
	"fmt"
	"strconv"

	"github.com/xgo-dev/llvm"
)

// These operations have instruction-specific behavior, including the precision
// of approximate reciprocals and the handling of infinities in VREDUCE.
type x86SIMDFamily uint8

const (
	x86AES x86SIMDFamily = iota
	x86SHA
	x86GF
	x86Reciprocal
	x86Scale
	x86Round
	x86Residue
)

type x86SIMDInstruction struct {
	family x86SIMDFamily
	name   string
	mode   int
}

var x86SIMDInstructions = map[SIMDOp]x86SIMDInstruction{
	SIMDAESEncryptOneRound: {x86AES, "aesenc", 0}, SIMDAESEncryptLastRound: {x86AES, "aesenclast", 0},
	SIMDAESDecryptOneRound: {x86AES, "aesdec", 0}, SIMDAESDecryptLastRound: {x86AES, "aesdeclast", 0},
	SIMDAESInvMixColumns: {x86AES, "aesimc", 0}, SIMDAESRoundKeyGenAssist: {x86AES, "aeskeygenassist", 0},
	SIMDSHA1FourRounds: {x86SHA, "sha1rnds4", 0}, SIMDSHA1NextE: {x86SHA, "sha1nexte", 0},
	SIMDSHA1Message1: {x86SHA, "sha1msg1", 0}, SIMDSHA1Message2: {x86SHA, "sha1msg2", 0},
	SIMDSHA256TwoRounds: {x86SHA, "sha256rnds2", 0}, SIMDSHA256Message1: {x86SHA, "sha256msg1", 0}, SIMDSHA256Message2: {x86SHA, "sha256msg2", 0},
	SIMDGaloisFieldMul: {x86GF, "vgf2p8mulb", 0}, SIMDGaloisFieldAffineTransform: {x86GF, "vgf2p8affineqb", 0}, SIMDGaloisFieldAffineTransformInverse: {x86GF, "vgf2p8affineinvqb", 0},
	SIMDReciprocal: {x86Reciprocal, "rcp", 0}, SIMDReciprocalSqrt: {x86Reciprocal, "rsqrt", 0}, SIMDScale: {x86Scale, "scalef", 0},
	SIMDRoundScaled: {x86Round, "rndscale", 0}, SIMDFloorScaled: {x86Round, "rndscale", 1}, SIMDCeilScaled: {x86Round, "rndscale", 2}, SIMDTruncScaled: {x86Round, "rndscale", 3},
	SIMDRoundScaledResidue: {x86Residue, "reduce", 0}, SIMDFloorScaledResidue: {x86Residue, "reduce", 1}, SIMDCeilScaledResidue: {x86Residue, "reduce", 2}, SIMDTruncScaledResidue: {x86Residue, "reduce", 3},
}

func SIMDX86Special(op SIMDOp) bool { _, ok := x86SIMDInstructions[op]; return ok }

// Keep the required predicate bits in sync with internal/cpu's publication.
func (ins x86SIMDInstruction) encoding(width int, single bool) (name, features string, required uint64) {
	suffix := "." + strconv.Itoa(width)
	switch ins.family {
	case x86AES:
		if width == 128 {
			return "aesni." + ins.name, "+aes,+avx", 8
		}
		if width == 256 {
			return "aesni." + ins.name + suffix, "+avx,+vaes", 16
		}
		return "aesni." + ins.name + suffix, "+avx512f,+vaes", 32
	case x86GF:
		features = "+gfni,+avx"
		if width == 256 {
			features += ",+avx2"
		}
		if width == 512 {
			features += ",+avx512f,+avx512bw"
		}
		return ins.name + suffix, features, 64
	case x86SHA:
		return ins.name, "+sha,+sse2", 128
	}
	precision := ".pd"
	if single {
		precision = ".ps"
	}
	features = "+avx512f,+avx512vl"
	if ins.family == x86Reciprocal {
		if single && width == 128 {
			return "sse." + ins.name + ".ps", "+sse", 4
		}
		if single && width == 256 {
			return "avx." + ins.name + ".ps.256", "+avx", 4
		}
		return "avx512." + ins.name + "14" + precision + suffix, features, 256
	}
	if ins.family == x86Residue {
		features += ",+avx512dq"
	}
	return "avx512.mask." + ins.name + precision + suffix, features, 256
}

// XMM-sized inputs and results use the same register class on both sides of
// the feature check. Keep wider values in caller-owned storage so the leaf
// never depends on agreement about optional YMM or ZMM call carriers.
func (b Builder) simdX86Special(op SIMDOp, result Type, args []Expr) Expr {
	ins := x86SIMDInstructions[op]
	width := int(b.Prog.td.TypeSizeInBits(result.ll))
	name, features, required := ins.encoding(width, result.ll.ElementType().TypeKind() == llvm.FloatTypeKind)
	b.requireX86SIMD(required)
	ctx := b.Prog.ctx
	ptr := b.Prog.VoidPtr().ll
	mod := b.Pkg.Module()
	intrinsic := mod.IntrinsicDeclaration(llvm.LookupIntrinsicID("llvm.x86."+name), nil)
	it := intrinsic.GlobalValueType()
	// Win64 passes XMM inputs indirectly under its ordinary calling convention.
	registers := width == 128 && b.Prog.Target().GOOS != "windows"
	ret := ctx.VoidType()
	var slots []llvm.Value
	var params []llvm.Type
	if registers {
		ret = it.ReturnType()
	} else {
		slots = append(slots, llvm.CreateAlloca(b.Func.entryAllocaBuilder(), result.ll))
		params = append(params, ptr)
	}
	fixed := -1
	symbol := fmt.Sprintf("__llgo_simd_%s_%d", name, ins.mode)
	for _, arg := range args {
		switch {
		case arg.ll.TypeKind() != llvm.VectorTypeKind:
			if k := arg.impl.IsAConstantInt(); !k.IsNil() && (ins.family == x86Round || ins.family == x86Residue || ins.family == x86SHA) {
				mask := uint64(15)
				if ins.family == x86SHA {
					mask = 3
				}
				fixed = int(k.ZExtValue() & mask)
				symbol += "_imm" + strconv.Itoa(fixed)
				continue
			}
			slots = append(slots, arg.impl)
			params = append(params, arg.ll)
		case registers:
			slots = append(slots, b.impl.CreateBitCast(arg.impl, ret, ""))
			params = append(params, ret)
		default:
			storage := llvm.CreateAlloca(b.Func.entryAllocaBuilder(), arg.ll)
			b.impl.CreateStore(arg.impl, storage)
			slots = append(slots, storage)
			params = append(params, ptr)
		}
	}
	ft := llvm.FunctionType(ret, params, false)
	leaf := mod.NamedFunction(symbol)
	if leaf.IsNil() {
		leaf = llvm.AddFunction(mod, symbol, ft)
		leaf.SetLinkage(llvm.InternalLinkage)
		leaf.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features))
		leaf.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(ctx.AddBasicBlock(leaf, "entry"))
		var vectors []llvm.Value
		var immediate llvm.Value
		first := 0
		if !registers {
			first = 1
		}
		for i := first; i < len(params); i++ {
			param := leaf.Param(i)
			switch params[i].TypeKind() {
			case llvm.PointerTypeKind:
				v := builder.CreateLoad(it.ReturnType(), param, "")
				v.SetAlignment(1)
				vectors = append(vectors, v)
			case llvm.VectorTypeKind:
				vectors = append(vectors, param)
			default:
				immediate = param
			}
		}
		emit := func(k int) {
			operands := append([]llvm.Value(nil), vectors...)
			types := it.ParamTypes()
			if fixed >= 0 || !immediate.IsNil() {
				if ins.family == x86Round || ins.family == x86Residue {
					k = k<<4 | ins.mode
				}
				operands = append(operands, llvm.ConstInt(types[len(operands)], uint64(k), false))
			}
			if ins.family == x86Round || ins.family == x86Residue || ins.family == x86Scale || ins.family == x86Reciprocal && len(types) > len(operands) {
				// All lanes enabled, with the instruction's ordinary MXCSR mode.
				operands = append(operands, llvm.ConstNull(it.ReturnType()), llvm.ConstAllOnes(types[len(operands)+1]))
				if len(operands) < len(types) {
					operands = append(operands, llvm.ConstInt(types[len(operands)], 4, false))
				}
			}
			v := builder.CreateCall(it, intrinsic, operands, "")
			// GF affine XORs the same byte into every result. AES key
			// generation XORs Rcon into the low byte of words one and three.
			// Apply variable constants after an instruction with immediate zero.
			if !immediate.IsNil() && ins.family == x86GF {
				one := builder.CreateInsertElement(llvm.Undef(v.Type()), immediate, llvm.ConstNull(ctx.Int32Type()), "")
				mask := llvm.ConstNull(llvm.VectorType(ctx.Int32Type(), v.Type().VectorSize()))
				v = builder.CreateXor(v, builder.CreateShuffleVector(one, llvm.Undef(v.Type()), mask, ""), "")
			}
			if op == SIMDAESRoundKeyGenAssist {
				words := llvm.VectorType(ctx.Int32Type(), 4)
				rcon := builder.CreateZExt(immediate, ctx.Int32Type(), "")
				pattern := llvm.ConstNull(words)
				for _, lane := range []uint64{1, 3} {
					pattern = builder.CreateInsertElement(pattern, rcon, llvm.ConstInt(ctx.Int32Type(), lane, false), "")
				}
				v = builder.CreateXor(v, builder.CreateBitCast(pattern, v.Type(), ""), "")
			}
			if registers {
				builder.CreateRet(v)
				return
			}
			store := builder.CreateStore(v, leaf.Param(0))
			store.SetAlignment(1)
			builder.CreateRetVoid()
		}
		if fixed >= 0 {
			emit(fixed)
		} else if immediate.IsNil() || ins.family == x86GF || op == SIMDAESRoundKeyGenAssist {
			emit(0)
		} else {
			cases := 16
			if ins.family == x86SHA {
				cases = 4
			}
			index := builder.CreateAnd(immediate, llvm.ConstInt(immediate.Type(), uint64(cases-1), false), "")
			def := ctx.AddBasicBlock(leaf, "imm0")
			sw := builder.CreateSwitch(index, def, cases-1)
			builder.SetInsertPointAtEnd(def)
			emit(0)
			for k := 1; k < cases; k++ {
				bb := ctx.AddBasicBlock(leaf, "imm"+strconv.Itoa(k))
				sw.AddCase(llvm.ConstInt(index.Type(), uint64(k), false), bb)
				builder.SetInsertPointAtEnd(bb)
				emit(k)
			}
		}
	}
	call := b.impl.CreateCall(ft, leaf, slots, "")
	if registers {
		return Expr{b.impl.CreateBitCast(call, result.ll, ""), result}
	}
	return Expr{b.impl.CreateLoad(result.ll, slots[0], ""), result}
}

// Keep a missing feature recoverable, but avoid a runtime call on every
// successful round of a SHA/AES kernel. CPUFeatures is initialized once by
// internal/cpu after applying GODEBUG; it must not be a compile-time constant.
func (b Builder) requireX86SIMD(bits uint64) {
	t := b.Prog.Uint64()
	g := b.Pkg.NewVarEx(PkgRuntime+".CPUFeatures", b.Prog.Pointer(t))
	published := b.impl.CreateLoad(t.ll, g.impl, "")
	required := llvm.ConstInt(t.ll, bits, false)
	masked := b.impl.CreateAnd(published, required, "")
	missing := llvm.CreateICmp(b.impl, llvm.IntNE, masked, required)
	logical, entry := b.blk, b.impl.GetInsertBlock()
	blocks := b.Func.MakeBlocks(2)
	b.If(Expr{missing, b.Prog.Bool()}, blocks[0], blocks[1])
	b.SetBlock(blocks[0])
	b.Call(b.Pkg.rtFunc("PanicSIMDX86Feature"))
	b.Unreachable()
	b.SetBlockEx(blocks[1], AtEnd, false)
	b.blk = logical
	if logical.last == entry {
		logical.last = blocks[1].last
	}
}

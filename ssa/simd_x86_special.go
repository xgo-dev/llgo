package ssa

import (
	"fmt"
	"strconv"

	"github.com/xgo-dev/llgo/internal/llvmfmv"
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

func (ins x86SIMDInstruction) encoding(width int, single bool) (name, features string) {
	suffix := "." + strconv.Itoa(width)
	switch ins.family {
	case x86AES:
		if width == 128 {
			return "aesni." + ins.name, "+aes,+avx"
		}
		if width == 256 {
			return "aesni." + ins.name + suffix, "+avx,+vaes"
		}
		return "aesni." + ins.name + suffix, "+avx512f,+vaes"
	case x86GF:
		features = "+gfni,+avx"
		if width == 256 {
			features += ",+avx2"
		}
		if width == 512 {
			features += ",+avx512f,+avx512bw"
		}
		return ins.name + suffix, features
	case x86SHA:
		return ins.name, "+sha,+sse2"
	}
	precision := ".pd"
	if single {
		precision = ".ps"
	}
	features = "+avx512f"
	if width < 512 {
		features += ",+avx512vl"
	}
	if ins.family == x86Reciprocal {
		if single && width == 128 {
			return "sse." + ins.name + ".ps", "+sse"
		}
		if single && width == 256 {
			return "avx." + ins.name + ".ps.256", "+avx"
		}
		return "avx512." + ins.name + "14" + precision + suffix, features
	}
	if ins.family == x86Residue {
		features += ",+avx512dq"
	}
	return "avx512.mask." + ins.name + precision + suffix, features
}

// SHA1 rounds and scaled rounding encode their selectors as two or four
// immediate bits. Dynamic selectors need one instruction for each encoding.
func (ins x86SIMDInstruction) selectorCases() int {
	switch ins.family {
	case x86SHA:
		return 4
	case x86Round, x86Residue:
		return 16
	}
	return 0
}

// Keep the vector signature visible to FMV so the calling body and the
// instruction use the same register width. LLVM may inline the instruction
// when the body already has its target features.
func (b Builder) simdX86Special(op SIMDOp, result Type, args []Expr) Expr {
	ins := x86SIMDInstructions[op]
	width := int(b.Prog.td.TypeSizeInBits(result.ll))
	name, features := ins.encoding(width, result.ll.ElementType().TypeKind() == llvm.FloatTypeKind)
	ctx := b.Prog.ctx
	mod := b.Pkg.Module()
	intrinsic := mod.IntrinsicDeclaration(llvm.LookupIntrinsicID("llvm.x86."+name), nil)
	it := intrinsic.GlobalValueType()
	ret := it.ReturnType()
	var operands []llvm.Value
	var params []llvm.Type
	cases := ins.selectorCases()
	fixed := -1
	symbol := fmt.Sprintf("__llgo_simd_%s_%d", name, ins.mode)
	for _, arg := range args {
		v := arg.impl
		if arg.ll.TypeKind() == llvm.VectorTypeKind {
			// The AES and GFNI intrinsics choose their own vector lane type.
			v = b.impl.CreateBitCast(v, ret, "")
		} else if k := v.IsAConstantInt(); cases != 0 && !k.IsNil() {
			fixed = int(k.ZExtValue() & uint64(cases-1))
			symbol += "_imm" + strconv.Itoa(fixed)
			continue
		}
		operands = append(operands, v)
		params = append(params, v.Type())
	}
	ft := llvm.FunctionType(ret, params, false)
	leaf := mod.NamedFunction(symbol)
	if leaf.IsNil() {
		leaf = llvm.AddFunction(mod, symbol, ft)
		leaf.SetLinkage(llvm.InternalLinkage)
		leaf.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features))
		llvmfmv.SetX86VectorCallConvention(mod, leaf)
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(ctx.AddBasicBlock(leaf, "entry"))
		var vectors []llvm.Value
		var immediate llvm.Value
		for i, t := range params {
			if t.TypeKind() == llvm.VectorTypeKind {
				vectors = append(vectors, leaf.Param(i))
			} else {
				immediate = leaf.Param(i)
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
			builder.CreateRet(v)
		}
		if immediate.IsNil() || cases == 0 {
			emit(max(fixed, 0))
		} else {
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
	call := b.impl.CreateCall(ft, leaf, operands, "")
	call.SetInstructionCallConv(leaf.FunctionCallConv())
	return Expr{b.impl.CreateBitCast(call, result.ll, ""), result}
}

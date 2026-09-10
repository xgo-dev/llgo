package ssa

import (
	"go/token"
	"strings"

	"github.com/xgo-dev/llvm"
)

func (b Builder) floatMinMax(op token.Token, a, c llvm.Value) llvm.Value {
	// Go propagates NaNs and orders -0 below +0. Neither compare/select nor
	// LLVM minnum/maxnum implements both rules.
	if b.Prog.Target().useFloatMinMaxIntrinsics() {
		name := "llvm.minimum"
		if op == token.GTR {
			name = "llvm.maximum"
		}
		return b.impl.CreateIntrinsic(a.Type(), llvm.LookupIntrinsicID(name), []llvm.Value{a, c}, "")
	}
	return b.floatMinMaxBits(op, a, c)
}

func (p *Target) useFloatMinMaxIntrinsics() bool {
	// Named targets may retarget the emitted IR in an external compiler with
	// different floating-point features. Keep their lowering portable.
	if p.Target != "" && p.WasmProfile == "" {
		return false
	}
	spec := p.Spec()
	switch strings.SplitN(spec.Triple, "-", 2)[0] {
	case "aarch64", "arm64", "x86_64", "wasm32", "wasm64":
		return true
	case "i386", "i686":
		return strings.Contains(spec.Features, "+sse2") && !strings.Contains(spec.Features, "+soft-float")
	}
	// LLVM 22 can fail to select minimum.f64 on ARM32 without ARMv8 FP,
	// or emit C23 fminimum/fmaximum libcalls on soft-float targets. Those
	// functions are not available in all supported C libraries.
	return false
}

// floatMinMaxBits orders IEEE float32/float64 encodings without floating-point
// instructions or libcalls. Positive encodings sort upwards, negative ones
// downwards; flipping the sign bit or complementing a negative encoding gives
// unsigned keys with -0 immediately below +0. NaNs are handled separately.
func (b Builder) floatMinMaxBits(op token.Token, a, c llvm.Value) llvm.Value {
	width, infinity, ty := 64, uint64(0x7ff0000000000000), b.Prog.tyInt64()
	if a.Type().TypeKind() == llvm.FloatTypeKind {
		width, infinity, ty = 32, 0x7f800000, b.Prog.tyInt32()
	}
	sign := uint64(1) << (width - 1)
	ua := b.impl.CreateBitCast(a, ty, "")
	uc := b.impl.CreateBitCast(c, ty, "")
	key := func(u llvm.Value) llvm.Value {
		mask := b.impl.CreateAShr(u, llvm.ConstInt(ty, uint64(width-1), false), "")
		mask = b.impl.CreateOr(mask, llvm.ConstInt(ty, sign, false), "")
		return b.impl.CreateXor(u, mask, "")
	}
	pred := llvm.IntULT
	if op == token.GTR {
		pred = llvm.IntUGT
	}
	result := b.impl.CreateSelect(b.impl.CreateICmp(pred, key(ua), key(uc), ""), ua, uc, "")
	isNaN := func(u llvm.Value) llvm.Value {
		abs := b.impl.CreateAnd(u, llvm.ConstInt(ty, sign-1, false), "")
		return b.impl.CreateICmp(llvm.IntUGT, abs, llvm.ConstInt(ty, infinity, false), "")
	}
	result = b.impl.CreateSelect(isNaN(uc), uc, result, "")
	result = b.impl.CreateSelect(isNaN(ua), ua, result, "")
	return b.impl.CreateBitCast(result, a.Type(), "")
}

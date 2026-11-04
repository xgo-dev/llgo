package ssa

import (
	"go/types"
	"strings"

	"github.com/xgo-dev/llvm"
)

// AESOp is the private interface used by the AES source patch. ARM rounds
// consume a key before SubBytes; the other targets consume it after the round.
type AESOp uint8

const (
	AESEncryptRound AESOp = iota
	AESEncryptLast
	AESDecryptRound
	AESDecryptLast
	AESInverseMixKey
	AESXORKey
	AESXORBlock
)

// AESAlloca reserves fixed-size, pointer-free AES state in the entry block.
// Explicit vector storage and alignment permit SROA while making counter-word
// accesses valid on strict-alignment targets as well.
func (b Builder) AESAlloca(size int) Expr {
	if size != 16 && size != 128 {
		panic("aesAlloca: expected 16 or 128 bytes")
	}
	ptr := llvm.CreateAlloca(b.Func.entryAllocaBuilder(), llvm.ArrayType(b.aesBytes(), size/16))
	ptr.SetAlignment(16)
	return Expr{ptr, b.Prog.VoidPtr()}
}

func (b Builder) requireAESFeatures() {
	var required, cpu string
	switch b.Prog.Target().GOARCH {
	case "amd64", "386":
		required = "+aes,+sse,+sse2,-soft-float"
	case "arm64":
		required = "+aes,+neon"
	case "arm":
		required, cpu = "+armv8-a,+neon,+aes,+crypto,-soft-float", "cortex-a53"
	case "ppc64", "ppc64le":
		required, cpu = "+altivec,+vsx,+crypto", "pwr8"
	case "riscv64":
		required = "+zkne,+zknd"
	default:
		panic("AES intrinsic on unsupported architecture")
	}
	features := b.Prog.Target().Spec().Features
	for _, attr := range b.Func.impl.GetFunctionAttributes() {
		if attr.IsString() && attr.GetStringKind() == "target-features" {
			features = attr.GetStringValue()
		}
	}
	for _, feature := range strings.Split(required, ",") {
		var kept []string
		for _, old := range strings.Split(features, ",") {
			name := strings.TrimLeft(old, "+-")
			if b.Prog.Target().GOARCH == "arm" && strings.HasPrefix(name, "armv") && name != "armv8-a" {
				continue // the guarded function targets ARMv8, not the baseline ISA
			}
			if name != "" && name != strings.TrimLeft(feature, "+-") {
				kept = append(kept, old)
			}
		}
		features = strings.Join(append(kept, feature), ",")
	}
	b.Func.impl.AddFunctionAttr(b.Prog.ctx.CreateStringAttribute("target-features", features))
	if cpu != "" {
		b.Func.impl.AddFunctionAttr(b.Prog.ctx.CreateStringAttribute("target-cpu", cpu))
	}
	// CPU dispatch lives in the caller. Keep optional instructions out of the
	// portable path even when full LTO can see both bodies.
	b.Func.impl.AddFunctionAttr(b.Prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
}

func (b Builder) aesIntrinsic(name string, result llvm.Type, args ...llvm.Value) llvm.Value {
	return b.impl.CreateIntrinsic(result, llvm.LookupIntrinsicID(name), args, "")
}

func (b Builder) aesBytes() llvm.Type {
	return llvm.VectorType(b.Prog.ctx.Int8Type(), 16)
}

func (b Builder) aesShuffle(v llvm.Value, indices ...int) llvm.Value {
	mask := make([]llvm.Value, len(indices))
	for i, index := range indices {
		mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
	}
	return b.impl.CreateShuffleVector(v, llvm.Undef(v.Type()), llvm.ConstVector(mask, false), "")
}

// The standard generic key schedule stores big-endian words, represented as
// native uint32 values. Hardware instructions consume the corresponding bytes.
func (b Builder) aesKeyBytes(v llvm.Value) llvm.Value {
	if b.Prog.Target().GOARCH == "ppc64" {
		return v
	}
	return b.aesShuffle(v, 3, 2, 1, 0, 7, 6, 5, 4, 11, 10, 9, 8, 15, 14, 13, 12)
}

func (b Builder) aesRISCV(name string, v llvm.Value) llvm.Value {
	words := llvm.VectorType(b.Prog.ctx.Int64Type(), 2)
	v = b.impl.CreateBitCast(v, words, "")
	lo := b.impl.CreateExtractElement(v, llvm.ConstInt(b.Prog.tyInt32(), 0, false), "")
	hi := b.impl.CreateExtractElement(v, llvm.ConstInt(b.Prog.tyInt32(), 1, false), "")
	result := llvm.Undef(words)
	var a, c llvm.Value
	if name == "aes64im" {
		a = b.aesIntrinsic("llvm.riscv."+name, lo.Type(), lo)
		c = b.aesIntrinsic("llvm.riscv."+name, hi.Type(), hi)
	} else {
		a = b.aesIntrinsic("llvm.riscv."+name, lo.Type(), lo, hi)
		c = b.aesIntrinsic("llvm.riscv."+name, hi.Type(), hi, lo)
	}
	result = b.impl.CreateInsertElement(result, a, llvm.ConstInt(b.Prog.tyInt32(), 0, false), "")
	result = b.impl.CreateInsertElement(result, c, llvm.ConstInt(b.Prog.tyInt32(), 1, false), "")
	return b.impl.CreateBitCast(result, b.aesBytes(), "")
}

func (b Builder) aesPPC(name string, a, c llvm.Value) llvm.Value {
	// Power's crypto instructions number the state in big-endian register
	// order, whereas LLVM's vector memory order follows the target endian.
	little := b.Prog.Target().GOARCH == "ppc64le"
	if little {
		a = b.aesShuffle(a, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)
		c = b.aesShuffle(c, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)
	}
	words := llvm.VectorType(b.Prog.ctx.Int64Type(), 2)
	a = b.impl.CreateBitCast(a, words, "")
	c = b.impl.CreateBitCast(c, words, "")
	v := b.aesIntrinsic("llvm.ppc.altivec.crypto."+name, words, a, c)
	v = b.impl.CreateBitCast(v, b.aesBytes(), "")
	if little {
		v = b.aesShuffle(v, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)
	}
	return v
}

// AESRound operates on unaligned byte buffers. Only the round-key operand (or
// both operands for InverseMixKey) uses the generic uint32 schedule layout.
func (b Builder) AESRound(op AESOp, dst, src, key Expr) Expr {
	if op > AESXORBlock {
		panic("aesRound: invalid operation")
	}
	for _, arg := range []Expr{dst, src, key} {
		if !types.Identical(arg.RawType(), types.NewPointer(types.Typ[types.Uint8])) {
			panic("aesRound: expected *byte operands")
		}
	}
	b.requireAESFeatures()
	vec := b.aesBytes()
	state := b.impl.CreateLoad(vec, src.impl, "")
	state.SetAlignment(1)
	if op == AESInverseMixKey {
		state = b.aesKeyBytes(state)
	}
	k := llvm.ConstNull(vec)
	if op != AESInverseMixKey {
		k = b.impl.CreateLoad(vec, key.impl, "")
		k.SetAlignment(1)
		if op != AESXORBlock {
			k = b.aesKeyBytes(k)
		}
	}
	var value llvm.Value
	if op == AESXORKey || op == AESXORBlock {
		value = b.impl.CreateXor(state, k, "")
	} else {
		switch b.Prog.Target().GOARCH {
		case "arm64", "arm":
			prefix := "llvm.aarch64.crypto."
			if b.Prog.Target().GOARCH == "arm" {
				prefix = "llvm.arm.neon."
			}
			switch op {
			case AESEncryptRound, AESEncryptLast:
				value = b.aesIntrinsic(prefix+"aese", vec, state, k)
				if op == AESEncryptRound {
					value = b.aesIntrinsic(prefix+"aesmc", vec, value)
				}
			case AESDecryptRound, AESDecryptLast:
				value = b.aesIntrinsic(prefix+"aesd", vec, state, k)
				if op == AESDecryptRound {
					value = b.aesIntrinsic(prefix+"aesimc", vec, value)
				}
			case AESInverseMixKey:
				value = b.aesIntrinsic(prefix+"aesimc", vec, state)
			}
		case "amd64", "386":
			words := llvm.VectorType(b.Prog.ctx.Int64Type(), 2)
			state = b.impl.CreateBitCast(state, words, "")
			k = b.impl.CreateBitCast(k, words, "")
			name := []string{"aesenc", "aesenclast", "aesdec", "aesdeclast", "aesimc"}[op]
			if op == AESInverseMixKey {
				value = b.aesIntrinsic("llvm.x86.aesni."+name, words, state)
			} else {
				value = b.aesIntrinsic("llvm.x86.aesni."+name, words, state, k)
			}
			value = b.impl.CreateBitCast(value, vec, "")
		case "ppc64", "ppc64le":
			switch op {
			case AESEncryptRound, AESEncryptLast, AESDecryptLast:
				name := map[AESOp]string{AESEncryptRound: "vcipher", AESEncryptLast: "vcipherlast", AESDecryptLast: "vncipherlast"}[op]
				value = b.aesPPC(name, state, k)
			case AESDecryptRound:
				// VNCIPHER mixes the round key before InvMixColumns. Our generic
				// schedule has already mixed it, so add it after a zero-key round.
				value = b.impl.CreateXor(b.aesPPC("vncipher", state, llvm.ConstNull(vec)), k, "")
			case AESInverseMixKey:
				value = b.aesPPC("vcipherlast", state, llvm.ConstNull(vec))
				value = b.aesPPC("vncipher", value, llvm.ConstNull(vec))
			}
		case "riscv64":
			name := []string{"aes64esm", "aes64es", "aes64dsm", "aes64ds", "aes64im"}[op]
			value = b.aesRISCV(name, state)
			if op != AESInverseMixKey {
				value = b.impl.CreateXor(value, k, "")
			}
		}
	}
	if op == AESInverseMixKey {
		value = b.aesKeyBytes(value)
	}
	store := b.impl.CreateStore(value, dst.impl)
	store.SetAlignment(1)
	return Expr{store, b.Prog.Void()}
}

// AESSubWord substitutes every byte without secret-indexed table accesses.
func (b Builder) AESSubWord(x Expr) Expr {
	if !types.Identical(x.RawType(), types.Typ[types.Uint32]) {
		panic("aesSubWord: expected uint32")
	}
	b.requireAESFeatures()
	if b.Prog.Target().GOARCH == "riscv64" {
		v := b.impl.CreateZExt(x.impl, b.Prog.ctx.Int64Type(), "")
		v = b.impl.CreateShl(v, llvm.ConstInt(v.Type(), 32, false), "")
		v = b.aesIntrinsic("llvm.riscv.aes64ks1i", v.Type(), v, llvm.ConstInt(b.Prog.tyInt32(), 10, false))
		return Expr{b.impl.CreateTrunc(v, x.ll, ""), x.Type}
	}
	words := llvm.VectorType(b.Prog.tyInt32(), 4)
	v := llvm.Undef(words)
	for i := 0; i < 4; i++ {
		v = b.impl.CreateInsertElement(v, x.impl, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
	}
	switch b.Prog.Target().GOARCH {
	case "amd64", "386":
		wide := llvm.VectorType(b.Prog.ctx.Int64Type(), 2)
		v = b.impl.CreateBitCast(v, wide, "")
		v = b.aesIntrinsic("llvm.x86.aesni.aeskeygenassist", wide, v, llvm.ConstInt(b.Prog.ctx.Int8Type(), 0, false))
	case "arm64", "arm":
		v = b.impl.CreateBitCast(v, b.aesBytes(), "")
		name := "llvm.aarch64.crypto.aese"
		if b.Prog.Target().GOARCH == "arm" {
			name = "llvm.arm.neon.aese"
		}
		v = b.aesIntrinsic(name, v.Type(), v, llvm.ConstNull(v.Type()))
	case "ppc64", "ppc64le":
		wide := llvm.VectorType(b.Prog.ctx.Int64Type(), 2)
		v = b.impl.CreateBitCast(v, wide, "")
		v = b.aesIntrinsic("llvm.ppc.altivec.crypto.vsbox", wide, v)
	}
	v = b.impl.CreateBitCast(v, words, "")
	return Expr{b.impl.CreateExtractElement(v, llvm.ConstInt(b.Prog.tyInt32(), 0, false), ""), x.Type}
}

package ssa

import (
	"go/types"
	"strings"

	"github.com/xgo-dev/llvm"
)

const mapsRuntimePackage = "github.com/goplus/llgo/runtime/internal/runtime/maps."

func (b Builder) callAMD64MapIntrinsic(fn Expr, sig *types.Signature, args []Expr) (Expr, bool) {
	if b.Prog.target.GOARCH != "amd64" || !strings.HasPrefix(fn.Name(), mapsRuntimePackage) {
		return Expr{}, false
	}

	name := strings.TrimPrefix(fn.Name(), mapsRuntimePackage)
	retType := b.Prog.retType(sig)
	ret := func(v llvm.Value) (Expr, bool) { return Expr{v, retType}, true }

	switch name {
	case "bitsetFirst":
		return ret(b.impl.CreateIntrinsic(args[0].impl.Type(), llvm.LookupIntrinsicID("llvm.cttz"), []llvm.Value{
			args[0].impl, llvm.ConstInt(b.Prog.ctx.Int1Type(), 0, false),
		}, ""))
	case "bitsetRemoveBelow":
		one := llvm.ConstInt(args[0].impl.Type(), 1, false)
		mask := b.impl.CreateSub(b.impl.CreateShl(one, args[1].impl, ""), one, "")
		return ret(b.impl.CreateAnd(args[0].impl, b.impl.CreateNot(mask, ""), ""))
	case "bitsetLowestSet":
		one := llvm.ConstInt(args[0].impl.Type(), 1, false)
		low := b.impl.CreateAnd(args[0].impl, one, "")
		return ret(b.impl.CreateICmp(llvm.IntEQ, low, one, ""))
	case "bitsetShiftOutLowest":
		one := llvm.ConstInt(args[0].impl.Type(), 1, false)
		return ret(b.impl.CreateLShr(args[0].impl, one, ""))
	case "ctrlGroupMatchH2":
		group := mapGroupBytes(b, args[0].impl)
		h := b.impl.CreateTrunc(args[1].impl, b.Prog.ctx.Int8Type(), "")
		return ret(mapByteMask(b, b.impl.CreateICmp(llvm.IntEQ, group, splatByte(b, h), ""), retType.ll))
	case "ctrlGroupMatchEmpty":
		group := mapGroupBytes(b, args[0].impl)
		empty := llvm.ConstInt(b.Prog.ctx.Int8Type(), 0x80, false)
		return ret(mapByteMask(b, b.impl.CreateICmp(llvm.IntEQ, group, splatByte(b, empty), ""), retType.ll))
	case "ctrlGroupMatchEmptyOrDeleted":
		group := mapGroupBytes(b, args[0].impl)
		zero := llvm.ConstNull(group.Type())
		return ret(mapByteMask(b, b.impl.CreateICmp(llvm.IntSLT, group, zero, ""), retType.ll))
	case "ctrlGroupMatchFull":
		group := mapGroupBytes(b, args[0].impl)
		zero := llvm.ConstNull(group.Type())
		return ret(mapByteMask(b, b.impl.CreateICmp(llvm.IntSGE, group, zero, ""), retType.ll))
	default:
		return Expr{}, false
	}
}

// callARM64MapIntrinsic lowers the portable Swiss-map bitset helpers at the
// call site. ARM64 uses one high bit per control byte (unlike AMD64's packed
// one-bit representation), so these operations intentionally mirror the
// generic implementations in runtime/internal/runtime/maps/group.go.
func (b Builder) callARM64MapIntrinsic(fn Expr, sig *types.Signature, args []Expr) (Expr, bool) {
	if b.Prog.target.GOARCH != "arm64" || !strings.HasPrefix(fn.Name(), mapsRuntimePackage) {
		return Expr{}, false
	}

	name := strings.TrimPrefix(fn.Name(), mapsRuntimePackage)
	retType := b.Prog.retType(sig)
	ret := func(v llvm.Value) (Expr, bool) { return Expr{v, retType}, true }
	u64 := b.Prog.ctx.Int64Type()
	one := llvm.ConstInt(u64, 1, false)
	const lsb uint64 = 0x0101010101010101
	const msb uint64 = 0x8080808080808080

	switch name {
	case "bitsetFirst":
		zeros := b.impl.CreateIntrinsic(args[0].impl.Type(), llvm.LookupIntrinsicID("llvm.cttz"), []llvm.Value{
			args[0].impl, llvm.ConstInt(b.Prog.ctx.Int1Type(), 0, false),
		}, "")
		return ret(b.impl.CreateLShr(zeros, llvm.ConstInt(args[0].impl.Type(), 3, false), ""))
	case "bitsetRemoveBelow":
		shift := b.impl.CreateMul(args[1].impl, llvm.ConstInt(args[1].impl.Type(), 8, false), "")
		mask := b.impl.CreateSub(b.impl.CreateShl(one, shift, ""), one, "")
		return ret(b.impl.CreateAnd(args[0].impl, b.impl.CreateNot(mask, ""), ""))
	case "bitsetLowestSet":
		mask := llvm.ConstInt(u64, 0x80, false)
		return ret(b.impl.CreateICmp(llvm.IntNE, b.impl.CreateAnd(args[0].impl, mask, ""), llvm.ConstNull(u64), ""))
	case "bitsetShiftOutLowest":
		return ret(b.impl.CreateLShr(args[0].impl, llvm.ConstInt(u64, 8, false), ""))
	case "ctrlGroupMatchH2":
		h := b.impl.CreateZExt(args[1].impl, u64, "")
		v := b.impl.CreateXor(args[0].impl, b.impl.CreateMul(llvm.ConstInt(u64, lsb, false), h, ""), "")
		matches := b.impl.CreateAnd(b.impl.CreateSub(v, llvm.ConstInt(u64, lsb, false), ""), b.impl.CreateNot(v, ""), "")
		return ret(b.impl.CreateAnd(matches, llvm.ConstInt(u64, msb, false), ""))
	case "ctrlGroupMatchEmpty":
		v := args[0].impl
		return ret(b.impl.CreateAnd(v, b.impl.CreateNot(b.impl.CreateShl(v, llvm.ConstInt(u64, 6, false), ""), ""), ""))
	case "ctrlGroupMatchEmptyOrDeleted":
		return ret(b.impl.CreateAnd(args[0].impl, llvm.ConstInt(u64, msb, false), ""))
	case "ctrlGroupMatchFull":
		return ret(b.impl.CreateAnd(b.impl.CreateNot(args[0].impl, ""), llvm.ConstInt(u64, msb, false), ""))
	default:
		return Expr{}, false
	}
}

func mapGroupBytes(b Builder, group llvm.Value) llvm.Value {
	return b.impl.CreateBitCast(group, llvm.VectorType(b.Prog.ctx.Int8Type(), 8), "")
}

func splatByte(b Builder, value llvm.Value) llvm.Value {
	vecType := llvm.VectorType(b.Prog.ctx.Int8Type(), 8)
	vec := b.impl.CreateInsertElement(llvm.Undef(vecType), value, llvm.ConstInt(b.Prog.ctx.Int32Type(), 0, false), "")
	indices := make([]llvm.Value, 8)
	for i := range indices {
		indices[i] = llvm.ConstInt(b.Prog.ctx.Int32Type(), 0, false)
	}
	return b.impl.CreateShuffleVector(vec, llvm.Undef(vecType), llvm.ConstVector(indices, false), "")
}

func mapByteMask(b Builder, matches llvm.Value, retType llvm.Type) llvm.Value {
	return b.impl.CreateZExt(b.impl.CreateBitCast(matches, b.Prog.ctx.Int8Type(), ""), retType, "")
}

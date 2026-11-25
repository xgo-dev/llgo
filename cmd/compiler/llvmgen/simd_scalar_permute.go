package llvmgen

import "github.com/xgo-dev/llvm"

// simdPermuteScalars implements PSHUFD/PSHUFHW/PSHUFLW within 128-bit groups.
// A packed byte selects four lanes; word shuffles preserve the other half of
// each group. Every packed selector is valid, with no cross-group selection.
func (b Builder) simdPermuteScalars(op SIMDOp, x, selector Expr) Expr {
	n := x.ll.VectorSize()
	group := 128 / x.ll.ElementType().IntTypeWidth()
	start := 0
	if op == SIMDPermuteScalarsHi || op == SIMDPermuteScalarsHiGrouped {
		start = 4
	}
	lane := func(i, selected int) int {
		if i%group < start || i%group >= start+4 {
			return i
		}
		return i/group*group + start + selected
	}
	shuffle := func(mask []llvm.Value) llvm.Value {
		return b.impl.CreateShuffleVector(x.impl, llvm.Undef(x.ll), llvm.ConstVector(mask, false), "")
	}
	if c := selector.impl.IsAConstantInt(); !c.IsNil() {
		mask := make([]llvm.Value, n)
		for i := range mask {
			selected := int(c.ZExtValue()>>uint(2*(i%4))) & 3
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(lane(i, selected)), false)
		}
		return Expr{shuffle(mask), x.Type}
	}

	// Use vector masks to choose between four fixed shuffles. Variable-index
	// extractelement would spill vectors and reload their lanes on baseline x86.
	var candidates [4]llvm.Value
	for selected := range candidates {
		mask := make([]llvm.Value, n)
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(lane(i, selected)), false)
		}
		candidates[selected] = shuffle(mask)
	}
	packed := b.impl.CreateZExt(selector.impl, x.ll.ElementType(), "")
	packed = b.impl.CreateInsertElement(llvm.Undef(x.ll), packed, llvm.ConstInt(b.Prog.tyInt32(), 0, false), "")
	packed = b.simdSplatLane0(Expr{packed, x.Type}).impl
	var bits [2]llvm.Value
	for bit := range bits {
		mask := make([]llvm.Value, n)
		for i := range mask {
			mask[i] = llvm.ConstInt(x.ll.ElementType(), 1<<uint(2*(i%4)+bit), false)
		}
		masked := b.impl.CreateAnd(packed, llvm.ConstVector(mask, false), "")
		bits[bit] = llvm.CreateICmp(b.impl, llvm.IntNE, masked, llvm.ConstNull(x.ll))
	}
	lo := b.impl.CreateSelect(bits[0], candidates[1], candidates[0], "")
	hi := b.impl.CreateSelect(bits[0], candidates[3], candidates[2], "")
	return Expr{b.impl.CreateSelect(bits[1], hi, lo, ""), x.Type}
}

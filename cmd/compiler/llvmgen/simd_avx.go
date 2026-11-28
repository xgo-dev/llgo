package llvmgen

import "github.com/xgo-dev/llvm"

// Keep the AVX target feature on a leaf, so baseline callers remain legal
// and LLVM models the upper-register clobbers across the call.
func (b Builder) simdClearAVXUpperBits() Expr {
	ctx := b.Prog.ctx
	functionType := llvm.FunctionType(ctx.VoidType(), nil, false)
	mod := b.Pkg.Module()
	const name = "__llgo_simd_vzeroupper"
	leaf := mod.NamedFunction(name)
	if leaf.IsNil() {
		leaf = llvm.AddFunction(mod, name, functionType)
		leaf.SetLinkage(llvm.InternalLinkage)
		leaf.AddFunctionAttr(ctx.CreateStringAttribute("target-features", "+avx"))
		leaf.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
		builder := ctx.NewBuilder()
		defer builder.Dispose()
		builder.SetInsertPointAtEnd(ctx.AddBasicBlock(leaf, "entry"))
		builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.x86.avx.vzeroupper"), nil, "")
		builder.CreateRetVoid()
	}
	enabled := b.Call(b.Pkg.rtFunc("SIMDHasAVX"))
	logical, entry := b.blk, b.impl.GetInsertBlock()
	blocks := b.Func.MakeBlocks(2)
	b.If(enabled, blocks[0], blocks[1])
	b.SetBlock(blocks[0])
	b.impl.CreateCall(functionType, leaf, nil, "")
	b.Jump(blocks[1])
	b.SetBlockEx(blocks[1], AtEnd, false)
	b.blk = logical
	if logical.last == entry {
		logical.last = blocks[1].last
	}
	return Expr{}
}

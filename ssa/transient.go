package ssa

import (
	"go/types"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

func (b Builder) stringStackBuffer() Expr {
	return b.uninitializedStackSlot(b.Prog.Type(types.NewArray(types.Typ[types.Byte], llabi.SmallStackBufferSize), GoDecl))
}

// Reserve one fixed slot in the entry frame. The caller must initialize all
// observable bytes before a read; buffer helpers and receiver adapters do so.
func (b Builder) uninitializedStackSlot(elem Type) Expr {
	entry := b.Func.entryAllocaBuilder()
	value := llvm.CreateAlloca(entry, b.Prog.storageType(elem))
	b.Prog.requireStorageAlignment(value, elem)
	return Expr{value, b.Prog.Pointer(elem)}
}

// ReadOnlyStringBytes is valid only with a frontend proof that the resulting
// slice neither escapes nor can be mutated, including through called helpers.
func (b Builder) ReadOnlyStringBytes(t Type, x Expr) Expr {
	return Expr{b.InlineCall(b.Pkg.rtFunc("StringToBytesReadOnly"), x).impl, t}
}

// ReadOnlyBytesString requires an immediately consumed string temporary with
// no source mutation between conversion and its sole comparison/lookup use.
func (b Builder) ReadOnlyBytesString(t Type, x Expr) Expr {
	return Expr{b.InlineCall(b.Pkg.rtFunc("StringFromBytesReadOnly"), x).impl, t}
}

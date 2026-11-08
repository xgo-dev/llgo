package ssa

import (
	"go/types"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

// SetEscapeStackBytes reserves the Go-level plan before helper lowering starts.
func (f Function) SetEscapeStackBytes(size uint64) {
	f.escapeStackBytes = size
}

func (f Function) reserveEscapeStack(typ Type) bool {
	size, alignment := f.Prog.SizeOf(typ), f.Prog.AlignOf(typ)
	if f.Prog.GCRootsEnabled() {
		size += f.Prog.StackRootBytes(typ)
	}
	if alignment == 0 || alignment&(alignment-1) != 0 || f.escapeStackBytes > llabi.MaxEscapeStackBytes {
		return false
	}
	padding := -f.escapeStackBytes & (alignment - 1)
	remaining := llabi.MaxEscapeStackBytes - f.escapeStackBytes
	if padding > remaining || size > remaining-padding {
		return false
	}
	f.escapeStackBytes += padding + size
	return true
}

func (b Builder) stringStackBuffer() Expr {
	return b.uninitializedStackSlot(b.Prog.Type(types.NewArray(types.Typ[types.Byte], llabi.SmallStackBufferSize), GoDecl))
}

// Reserve one fixed slot in the entry frame. The caller must initialize all
// observable bytes before a read; buffer helpers and receiver adapters do so.
func (b Builder) uninitializedStackSlot(elem Type) Expr {
	entry := b.Func.entryAllocaBuilder()
	value := llvm.CreateAlloca(entry, b.Prog.storageType(elem))
	b.Prog.requireStorageAlignment(value, elem)
	b.registerStackRoots(value, elem)
	return Expr{value, b.Prog.Pointer(elem)}
}

// ReadOnlyStringBytes is valid only with a frontend proof that the resulting
// slice neither escapes nor can be mutated, including through called helpers.
func (b Builder) ReadOnlyStringBytes(t Type, x Expr) Expr {
	return Expr{b.InlineCall(b.Pkg.rtFunc("StringToBytesReadOnly"), x).impl, t}
}

// ReadOnlyBytesString requires a temporary string with no source mutation on
// any path from the conversion to its final consumer.
func (b Builder) ReadOnlyBytesString(t Type, x Expr) Expr {
	return Expr{b.InlineCall(b.Pkg.rtFunc("StringFromBytesReadOnly"), x).impl, t}
}

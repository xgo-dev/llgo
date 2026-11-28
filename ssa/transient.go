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
	size := f.Prog.SizeOf(typ) + f.Prog.StackRootBytes(typ)
	alignment := f.Prog.AlignOf(typ)
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

func (p Program) stringConcatTable(n int) Type {
	return p.Type(types.NewArray(types.Typ[types.String], int64(n)), GoDecl)
}

// ReserveStringConcatTable charges the operand table and its GC roots during
// concat planning. Lowering helpers share the remaining stack budget. Escape
// analysis already accounts for the result buffer at the final addition.
func (f Function) ReserveStringConcatTable(n int) bool {
	return f.reserveEscapeStack(f.Prog.stringConcatTable(n))
}

// StringConcat concatenates at least two parts and preserves the Go result type.
// Three or more parts require a successful ReserveStringConcatTable.
// useStackBuffer enables a separately budgeted result buffer only for a result
// proven not to escape its invocation or loop iteration. Larger results still
// allocate on the heap.
func (b Builder) StringConcat(parts []Expr, resultType Type, useStackBuffer bool) Expr {
	if len(parts) == 2 {
		return b.stringCat(parts[0], parts[1], resultType, useStackBuffer)
	}
	p := b.Prog
	table := b.uninitializedStackSlot(p.stringConcatTable(len(parts)))
	data := Expr{table.impl, p.Pointer(p.String())}
	for i, part := range parts {
		b.Store(b.Advance(data, p.Val(i)), part)
	}
	var buffer Expr
	if useStackBuffer {
		buffer = b.stringStackBuffer()
	} else {
		buffer = p.Nil(p.VoidPtr())
	}
	result := b.InlineCall(b.Pkg.rtFunc("StringConcat"), data, p.Val(len(parts)), buffer)
	if p.GCRootsEnabled() {
		// A registered table lives until function return. Release the input
		// references now that the result owns its bytes.
		b.zeroinit(table, p.IntVal(p.SizeOf(p.Elem(table.Type)), p.Uintptr()))
	}
	return Expr{result.impl, resultType}
}

func (b Builder) stringCat(x, y Expr, resultType Type, useStackBuffer bool) Expr {
	if useStackBuffer {
		return Expr{b.InlineCall(b.Pkg.rtFunc("StringCatBuffer"), x, y, b.stringStackBuffer()).impl, resultType}
	}
	return Expr{b.InlineCall(b.Pkg.rtFunc("StringCat"), x, y).impl, resultType}
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

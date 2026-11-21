package ssa

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWasmHeaderStoragePolicy(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	text, word := prog.String().ll, prog.ctx.Int64Type()
	cell := prog.ctx.StructType([]llvm.Type{llvm.ArrayType(word, 250), text}, false)
	for _, tc := range []struct {
		name    string
		typ     llvm.Type
		storage bool
	}{
		{"string", text, false},
		{"slice", prog.Slice(prog.Byte()).ll, false},
		{"empty_array", llvm.ArrayType(text, 0), false},
		{"single_header", llvm.ArrayType(text, 1), false},
		// Two Go strings exceed four physical Wasm pointer words, even
		// though their size is only four of the wider Go storage words.
		{"two_headers", llvm.ArrayType(text, 2), true},
		{"small_struct", prog.ctx.StructType([]llvm.Type{text, text}, false), false},
		{"single_sparse_element", llvm.ArrayType(cell, 1), false},
		{"sparse_array_4032", llvm.ArrayType(cell, 2), true},
		{"large_struct", prog.ctx.StructType([]llvm.Type{llvm.ArrayType(word, 510), text}, false), true},
		{"many_headers", prog.ctx.StructType([]llvm.Type{llvm.ArrayType(text, 9)}, false), true},
		{"no_headers", llvm.ArrayType(word, 1024), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := prog.wasmHeaderCount(tc.typ, maxInlineWasmHeaders+1)
			if got := prog.needsWasmHeaderStorage(tc.typ, count); got != tc.storage {
				t.Errorf("header storage = %v for %s (%d bytes), want %v", got, tc.typ, prog.td.TypeAllocSize(tc.typ), tc.storage)
			}
		})
	}
}

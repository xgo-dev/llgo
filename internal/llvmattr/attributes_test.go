package llvmattr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestEncodingsMatchLLVM(t *testing.T) {
	const ir = `
declare void @none() memory(none)
declare void @read() memory(read)
declare void @argread(ptr) memory(argmem: read)
declare void @argwrite(ptr) memory(argmem: write)
declare ptr @single(i64) allocsize(0)
declare ptr @product(ptr, ptr, i64, i64) allocsize(2, 3)
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "attrs.ll")
	if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Dispose()
	for _, tc := range []struct {
		name, attr string
		value      uint64
	}{
		{"none", "memory", MemoryNone}, {"read", "memory", MemoryRead},
		{"argread", "memory", MemoryArgRead}, {"argwrite", "memory", MemoryArgWrite},
		{"single", "allocsize", AllocSize(0, AllocSizeNoCount)},
		{"product", "allocsize", AllocSize(2, 3)},
	} {
		attr := mod.NamedFunction(tc.name).GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(tc.attr))
		if attr.IsNil() || attr.GetEnumValue() != tc.value {
			t.Errorf("%s encoding = %v, want %v", tc.name, attr.GetEnumValue(), tc.value)
		}
	}
	for _, tc := range []struct {
		name           string
		element, count uint32
	}{
		{"single", 0, AllocSizeNoCount}, {"product", 2, 3},
	} {
		attr := mod.NamedFunction(tc.name).GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("allocsize"))
		element, count := AllocSizeArgs(attr.GetEnumValue())
		if element != tc.element || count != tc.count {
			t.Errorf("%s decoded indices = %d, %d", tc.name, element, count)
		}
	}
}

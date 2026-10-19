//go:build !llgo

package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMD256StorageAndNativeBoundary(t *testing.T) {
	pkg := types.NewPackage("simd/archsimd", "archsimd")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "v256", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "tag", types.NewArray(types.NewSignatureType(nil, nil, nil, nil, nil, false), 0), false),
	}, nil), nil)
	vector := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Float32x8", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "tag", marker, false),
		types.NewField(token.NoPos, pkg, "vals", types.NewArray(types.Typ[types.Float32], 8), false),
	}, nil), nil)
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	typ := prog.Type(vector, GoDecl)
	if typ.ll.TypeKind() != llvm.VectorTypeKind || typ.ll.VectorSize() != 8 || prog.SizeOf(typ) != 32 || prog.AlignOf(typ) != 8 {
		t.Fatal("wide value or Go storage shape")
	}
	record := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "prefix", types.Typ[types.Byte], false),
		types.NewField(token.NoPos, nil, "value", vector, false),
		types.NewField(token.NoPos, nil, "suffix", types.Typ[types.Byte], false),
	}, nil)
	storage := prog.Type(record, GoDecl)
	if prog.OffsetOf(storage, 1) != 8 || prog.OffsetOf(storage, 2) != 40 || prog.SizeOf(storage) != 48 {
		t.Fatal("native vector alignment leaked into Go aggregate")
	}
	for _, param := range []types.Type{vector, record, types.NewArray(vector, 2)} {
		func() {
			defer func() {
				value := recover()
				if value == nil || !strings.Contains(value.(string), "native C call boundaries") {
					t.Fatalf("native boundary diagnostic: %v", value)
				}
			}()
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", param)), nil, false)
			prog.FuncDecl(sig, CDecl)
		}()
	}
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewParam(token.NoPos, nil, "x", types.NewPointer(vector))), nil, false)
	prog.FuncDecl(sig, CDecl) // An explicit memory interface remains available.
}

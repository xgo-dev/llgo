package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestSourceAndStorageLayouts(t *testing.T) {
	const source = `package layout
type Empty struct{}
type Scalar uint64
type ScalarArray [2]uint64
type GoFunc func()
//llgo:cdecl
type CFunc func()
type CFuncAlias = CFunc
//llgo:cdecl
type NativeAliasFunc = func()
type NativeArrays struct { A [2]CFunc; B [2]CFuncAlias; C [2]NativeAliasFunc; Z byte }
//llgo:cdecl
type Native struct { B byte; F func(); P *byte; X float64; E Empty }
type NativeCopy Native
type NativeAlias = Native
type GoPointer *byte
//llgo:cdecl
type NativeGoFields struct { F GoFunc; P GoPointer; A [2]GoFunc }
//llgo:cdecl
type NativeAnonymous struct { S struct { X byte; E struct{} } }
type GoAnonymous struct { S struct { X byte; E struct{} } }
type Go struct { B byte; F func(); P *byte; X float64; E Empty }
//llgo:packed
type Packed struct { B byte; X uint32; F CFunc; C CFunc; Z byte }
//llgo:pack 2
type Pack2 struct { B byte; X uint32; Y uint16 }
//llgo:packed
type PackedNested struct { B byte; S struct { B byte; X uint64 }; Z byte }
type NoPad struct {
 X uint64
 //llgo:no-padding
 E Empty
}
type Pad struct {
 //llgo:padding
 E Empty
 B byte
}
type Pair[A, B any] struct {
 //llgo:padding-if
 A A
 //llgo:padding-if
 B B
}
type Same = Pair[Empty, Empty]
type Nested struct { B byte; P [2]Packed; N [2]NoPad; S [3]Same }
`
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten", WasmProfile: "j64", WasmProvider: "emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "layout.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := types.NewPackage("example/layout", "layout")
			if err := prog.ParseLayoutSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			conf := types.Config{Sizes: sizes}
			if err := types.NewChecker(&conf, fset, pkg, nil).Files([]*ast.File{file}); err != nil {
				t.Fatal(err)
			}
			if target.GOOS == "windows" && target.GOARCH == "386" {
				if got := prog.AlignOf(prog.Type(types.Typ[types.Float64], CDecl)); got != 8 {
					t.Fatalf("native double alignment = %d, want 8", got)
				}
			}
			for _, name := range []string{"Scalar", "ScalarArray", "GoFunc", "CFunc", "NativeArrays", "Native", "NativeCopy", "NativeAlias", "NativeGoFields", "NativeAnonymous", "GoAnonymous", "Go", "Packed", "Pack2", "PackedNested", "NoPad", "Pad", "Same", "Nested"} {
				t.Run(name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(name).Type()
					stored := prog.Type(raw, GoDecl)
					if got, want := int64(prog.SizeOf(stored)), sizes.Sizeof(raw); got != want {
						t.Errorf("LLVM size %d != source %d", got, want)
					}
					if got, want := int64(prog.AlignOf(stored)), sizes.Alignof(raw); got != want {
						t.Errorf("LLVM alignment %d != source %d", got, want)
					}
					if st, ok := raw.Underlying().(*types.Struct); ok {
						fields := make([]*types.Var, st.NumFields())
						for i := range fields {
							fields[i] = st.Field(i)
						}
						for i, offset := range sizes.Offsetsof(fields) {
							if got := int64(prog.OffsetOf(stored, i)); got != offset {
								t.Errorf("field %d LLVM offset %d != source %d", i, got, offset)
							}
						}
					}
				})
			}
		})
	}
}

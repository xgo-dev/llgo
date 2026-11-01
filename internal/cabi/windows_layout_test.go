package cabi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestWindows386DeclaredLayoutSurvivesABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	const source = `package layout
type Tail struct { A, B *byte; E struct{} }
type Pad struct {
 //llgo:padding
 E struct{}
 X uint32
}
//llgo:packed
type Packed struct { B byte; X uint32; Z byte }
//llgo:pack 2
type Pack2 struct { B byte; X uint32; Y uint16 }
type Nested [2]Packed
//llgo:cdecl
//llgo:packed
type Native struct { B byte; X uint32; Z byte }
type Ordinary struct { B byte; X uint64 }
`
	for _, triple := range []string{"i686-pc-windows-msvc", "i686-w64-windows-gnu"} {
		t.Run(triple, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "386", LLVMTarget: triple})
			defer prog.Dispose()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "layout.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := types.NewPackage("example/layout", "layout")
			if err := prog.ParseLayoutSyntax(fset, pkg, []*ast.File{file}, nil); err != nil {
				t.Fatal(err)
			}
			conf := types.Config{Sizes: prog.TypeSizes(types.SizesFor("gc", "386"))}
			if err := types.NewChecker(&conf, fset, pkg, nil).Files([]*ast.File{file}); err != nil {
				t.Fatal(err)
			}
			compiled := prog.NewPackage("layout", "example/layout")
			mod := compiled.Module()
			tr := NewTransformer(prog, triple, "", false)
			for _, tc := range []struct {
				name                   string
				storage, native, align int
			}{
				{"Tail", 12, 12, 4}, {"Pad", 8, 8, 4}, {"Packed", 6, 6, 1},
				{"Pack2", 8, 8, 2}, {"Nested", 12, 12, 1}, {"Native", 6, 6, 1},
				// Preserve the existing Go/386-to-native scalar alignment rule.
				{"Ordinary", 12, 16, 8},
			} {
				t.Run(tc.name, func(t *testing.T) {
					typ := pkg.Scope().Lookup(tc.name).Type()
					arg := types.NewVar(token.NoPos, nil, "", typ)
					fn := compiled.NewFunc(tc.name, types.NewSignatureType(nil, nil, nil, types.NewTuple(arg), types.NewTuple(arg), false), llssa.GoDecl)
					body := fn.MakeBody(1)
					defer body.Dispose()
					body.Return(fn.Param(0))
					body.EndBuild()
					ft := mod.NamedFunction(tc.name).GlobalValueType()
					stored := ft.ParamTypes()[0]
					if got := tr.Sizeof(stored); got != tc.storage {
						t.Fatalf("storage size = %d, want %d", got, tc.storage)
					}
					for _, index := range []int{0, 1} {
						info := tr.GetTypeInfo(mod.Context(), ft, stored, index)
						if info.Size != tc.native || info.Align != tc.align {
							t.Fatalf("ABI index %d: size/alignment = %d/%d, want %d/%d; type %s", index, info.Size, info.Align, tc.native, tc.align, info.nativeType())
						}
					}
				})
			}
			tr.TransformModule(pkg.Path(), mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

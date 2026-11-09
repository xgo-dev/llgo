package ssa

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestHostLayoutPreservesLegacySourceQueries(t *testing.T) {
	Initialize(InitAll)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"}, {GOOS: "windows", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64", LLVMTarget: "wasm64-unknown-emscripten"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			sizes := prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
			setTestRuntime(t, prog)
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "legacy.go", `package legacy
import "structs"
import "unsafe"
type H struct { _ structs.HostLayout; B byte }
type Control struct { B byte; Fn func(); Tail byte }
type Nested struct { H H; Fn func(); Tail byte }
type Direct struct { _ structs.HostLayout; B byte; Fn func(); Tail byte }
type Alias = struct { _ structs.HostLayout; B byte; Fn func(); Tail byte }
type GoRecord struct { _ structs.HostLayout; B byte; Fn func(); Tail byte }
const ControlSize = unsafe.Sizeof(Control{})
const NestedSize = unsafe.Sizeof(Nested{})
const DirectSize = unsafe.Sizeof(Direct{})
const AliasSize = unsafe.Sizeof(Alias{})
`, 0)
			if err != nil {
				t.Fatal(err)
			}
			// Match the source loader: register type directives and original
			// field positions before go/types evaluates unsafe.Sizeof constants.
			ast.Inspect(file, func(node ast.Node) bool {
				decl, ok := node.(*ast.TypeSpec)
				if !ok || decl.Name.Name == "H" || decl.Name.Name == "GoRecord" {
					return true
				}
				prog.SetTypeCallConv("legacy."+decl.Name.Name, CDecl)
				var positions []token.Pos
				for _, field := range decl.Type.(*ast.StructType).Fields.List {
					for _, name := range field.Names {
						positions = append(positions, name.Pos())
					}
				}
				prog.SetNativeStructFields(positions)
				return true
			})
			pkg, err := (&types.Config{Importer: importer.Default(), Sizes: sizes}).Check("legacy", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			control := pkg.Scope().Lookup("Control").Type()
			wantSize := sizes.Sizeof(control)
			wantOffsets := sizes.Offsetsof(structFields(control))
			for _, name := range []string{"Nested", "Direct", "Alias"} {
				t.Run(name, func(t *testing.T) {
					raw := pkg.Scope().Lookup(name).Type()
					if got := sizes.Sizeof(raw); got != wantSize {
						t.Fatalf("size=%d, legacy control=%d", got, wantSize)
					}
					if got := pkg.Scope().Lookup(name + "Size").(*types.Const).Val(); got.ExactString() != pkg.Scope().Lookup("ControlSize").(*types.Const).Val().ExactString() {
						t.Fatalf("checker constant=%s, legacy control=%d", got, wantSize)
					}
					if got := sizes.Sizeof(types.NewArray(raw, 2)); got != 2*wantSize {
						t.Fatalf("array size=%d, want %d", got, 2*wantSize)
					}
					offsets := sizes.Offsetsof(structFields(raw))
					if offsets[len(offsets)-1] != wantOffsets[len(wantOffsets)-1] {
						t.Fatalf("tail offset changed: %v, legacy control=%v", offsets, wantOffsets)
					}
				})
			}
			// Marker-only Go records must still account for their closure
			// environment, even in a package containing legacy declarations.
			goRecord := pkg.Scope().Lookup("GoRecord").Type()
			word := int64(prog.GoWordSize())
			checkHostLayout(t, prog, sizes, goRecord, 4*word, word, []int64{0, 0, word, 3 * word})
		})
	}
}

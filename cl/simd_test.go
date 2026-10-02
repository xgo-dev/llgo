package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDOperationIdentity(t *testing.T) {
	const source = `package archsimd
 type v128 struct { _ [0]func() }
 type Float32x4 struct { tag v128; vals [4]float32 }
 func (x Float32x4) Add(y Float32x4) Float32x4
 func Sum(x, y Float32x4) Float32x4
 `
	// A test-only package-function registration exercises the same signature
	// family without adding an operation to the supported official API.
	key := simdKey{"", "Sum"}
	simdOperations[key] = simdOperation{llssa.SIMDAdd, simdBinary, 0}
	defer delete(simdOperations, key)
	for _, tc := range []struct {
		name, path, arch, source string
		want                     bool
	}{
		{"method", "simd/archsimd", "amd64", source, true},
		{"package function", "simd/archsimd", "wasm", source, true},
		{"other package", "example/archsimd", "amd64", source, false},
		{"unsupported target", "simd/archsimd", "386", source, false},
		{"wrong argument", "simd/archsimd", "arm64", strings.Replace(source, "Add(y Float32x4)", "Add(y int)", 1), false},
		{"wrong result", "simd/archsimd", "arm64", strings.Replace(source, "Add(y Float32x4) Float32x4", "Add(y Float32x4) float32", 1), false},
		{"unknown method", "simd/archsimd", "amd64", strings.ReplaceAll(source, "Add", "Unknown"), false},
		{"missing result", "simd/archsimd", "amd64", strings.Replace(source, "Add(y Float32x4) Float32x4", "Add(y Float32x4)", 1), false},
		{"variadic argument", "simd/archsimd", "amd64", strings.Replace(source, "Add(y Float32x4)", "Add(y ...Float32x4)", 1), false},
		{"missing argument", "simd/archsimd", "amd64", strings.Replace(source, "Add(y Float32x4)", "Add()", 1), false},
		{"missing marker", "simd/archsimd", "amd64", strings.Replace(source, "tag v128; ", "", 1), false},
		{"nonarray storage", "simd/archsimd", "amd64", strings.Replace(source, "[4]float32", "float32", 1), false},
		{"nonnumeric lanes", "simd/archsimd", "amd64", strings.Replace(source, "[4]float32", "[4]bool", 1), false},
		{"aggregate lanes", "simd/archsimd", "amd64", strings.Replace(source, "[4]float32", "[4]struct{}", 1), false},

		{"wrong lanes", "simd/archsimd", "arm64", strings.ReplaceAll(source, "[4]float32", "[2]float32"), false},
		{"nonzero marker", "simd/archsimd", "arm64", strings.ReplaceAll(source, "type v128 struct { _ [0]func() }", "type v128 struct{x int}"), false},
		{"wrong marker", "simd/archsimd", "arm64", strings.ReplaceAll(source, "tag v128", "tag int"), false},
		{"float bitwise", "simd/archsimd", "arm64", strings.ReplaceAll(source, "Add", "And"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "simd.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check(tc.path, fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssa.NewProgram(fs, 0)
			prog.CreatePackage(pkg, nil, nil, true)
			var obj *types.Func
			if tc.name == "package function" {
				obj = pkg.Scope().Lookup("Sum").(*types.Func)
			} else {
				obj = pkg.Scope().Lookup("Float32x4").Type().(*types.Named).Method(0)
			}
			desc, ok := lookupSIMD(prog.FuncValue(obj), tc.arch)
			if ok != tc.want {
				t.Fatalf("lookup = %v, want %v", ok, tc.want)
			}
			if ok && desc.op != llssa.SIMDAdd {
				t.Fatalf("unexpected operation %v", desc.op)
			}
		})
	}
}

func TestSIMDFallbackDeclarations(t *testing.T) {
	const source = `package archsimd
 type Mask struct{ bits uint16 }
 func Missing()
 func (Mask) Not() Mask
 func Helper() int { return 7 }
 type v128 struct { _ [0]func() }
 type Float32x4 struct { tag v128; vals [4]float32 }
 func (x Float32x4) Add(y Float32x4) Float32x4 { return x }
 `
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "simd.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fs, types.NewPackage("simd/archsimd", "archsimd"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	mask := pkg.Pkg.Scope().Lookup("Mask").Type().(*types.Named)
	vector := pkg.Pkg.Scope().Lookup("Float32x4").Type().(*types.Named)
	for _, tc := range []struct {
		fn   *ssa.Function
		want bool
	}{
		{pkg.Func("Missing"), true},
		{pkg.Prog.FuncValue(mask.Method(0)), true},
		{pkg.Func("Helper"), false},
		{pkg.Prog.FuncValue(vector.Method(0)), false},
	} {
		op, ok := lookupSIMD(tc.fn, "amd64")
		if ok != tc.want || ok && op.op != llssa.SIMDUnimplemented {
			t.Errorf("%s: fallback=%v op=%v", tc.fn, ok, op.op)
		}
	}
}

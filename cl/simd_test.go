package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestSIMDOriginalEntriesAcrossPackages(t *testing.T) {
	dep, root := buildCallerFrameSSAProgram(t, "simd/archsimd", `package archsimd
type v128 struct { _ [0]func() }
type Float32x4 struct { tag v128; vals [4]float32 }
func (x Float32x4) Add(y Float32x4) Float32x4
func LoadFloat32x4Array(x *[4]float32) Float32x4
`, "example.com/root", `package root
import "simd/archsimd"
var Add = archsimd.Float32x4.Add
var Load = archsimd.LoadFloat32x4Array
`)
	vector := dep.Pkg.Scope().Lookup("Float32x4").Type().(*types.Named)
	method := dep.Prog.FuncValue(vector.Method(0))
	names := []string{
		funcName(dep.Pkg, method, false),
		funcName(dep.Pkg, dep.Func("LoadFloat32x4Array"), false),
	}
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			owner, err := NewPackage(prog, dep, nil)
			if err != nil {
				t.Fatal(err)
			}
			consumer, err := NewPackage(prog, root, nil)
			if err != nil {
				t.Fatal(err)
			}
			// SIMD emits entries in its defining package even without taking
			// their values there. Importers synthesize the same original names;
			// both definitions need COMDAT on COFF to coalesce at native link.
			for i, pkg := range []llssa.Package{owner, consumer} {
				entries := names
				linkage := llvm.ExternalLinkage
				if i == 1 {
					// Method expressions have their own SSA thunk; the package
					// function value materializes the shared entry directly.
					entries = names[1:]
					linkage = llvm.LinkOnceAnyLinkage
				}
				for _, name := range entries {
					entry := mustNamedFunction(t, pkg.Module(), name)
					if entry.IsDeclaration() || entry.Linkage() != linkage {
						t.Fatalf("SIMD entry must share its original name and linkage:\n%s", entry.String())
					}
					if target.GOOS == "windows" && !strings.Contains(entry.String(), "comdat") {
						t.Fatalf("SIMD entry lacks COFF COMDAT:\n%s", entry.String())
					}
				}
				if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
			}
			// External aliases may be the only callers of the defining entry.
			// Keep its original symbol through independent package optimization.
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := owner.Module().RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			for _, name := range names {
				if entry := mustNamedFunction(t, owner.Module(), name); entry.IsDeclaration() || entry.Linkage() != llvm.ExternalLinkage {
					t.Fatal("optimized SIMD entry did not retain its original definition")
				}
			}
			if err := llvm.LinkModules(owner.Module(), consumer.Module()); err != nil {
				t.Fatal(err)
			}
			for _, name := range names {
				mustNamedFunction(t, owner.Module(), name)
			}
			if err := llvm.VerifyModule(owner.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSIMDFMVQueryIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, path, recv, signature string
		want                        string
	}{
		{"official", "simd/archsimd", "X86Features", "AVX2() bool", "x86.avx2"},
		{"other package", "example/archsimd", "X86Features", "AVX2() bool", ""},
		{"other receiver", "simd/archsimd", "OtherFeatures", "AVX2() bool", ""},
		{"pointer receiver", "simd/archsimd", "*X86Features", "AVX2() bool", ""},
		{"other query", "simd/archsimd", "X86Features", "AVX() bool", ""},
		{"argument", "simd/archsimd", "X86Features", "AVX2(int) bool", ""},
		{"result", "simd/archsimd", "X86Features", "AVX2() int", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "cpu.go", "package archsimd; type X86Features struct{}; type OtherFeatures struct{}; func ("+tc.recv+") "+tc.signature, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check(tc.path, fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssa.NewProgram(fs, 0)
			prog.CreatePackage(pkg, nil, nil, true)
			named := pkg.Scope().Lookup(strings.TrimPrefix(tc.recv, "*")).Type().(*types.Named)
			if got := simdCPUQuery(prog.FuncValue(named.Method(0))); got != tc.want {
				t.Fatalf("query = %q, want %q", got, tc.want)
			}
		})
	}
}

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

func TestSIMDBridgeOperationSignatures(t *testing.T) {
	const declarations = `package archsimd
type v128 struct { _ [0]func() }
type Float32x4 struct { tag v128; vals [4]float32 }
type Float64x2 struct { tag v128; vals [2]float64 }
type Uint32x4 struct { tag v128; vals [4]uint32 }
type Uint64x2 struct { tag v128; vals [2]uint64 }
type Int64x2 struct { tag v128; vals [2]int64 }
type Mask32x4 struct { tag v128; vals [4]int32 }
type Mask64x2 struct { tag v128; vals [2]int64 }
`
	for _, tc := range []struct {
		name, receiver, method string
		want                   llssa.SIMDOp
	}{
		{"masked store", "Float32x4", "StoreArrayMasked(*[4]float32, Mask32x4)", llssa.SIMDStoreMasked},
		{"missing mask", "Float32x4", "StoreArrayMasked(*[4]float32)", llssa.SIMDUnimplemented},
		{"numeric mask", "Float32x4", "StoreArrayMasked(*[4]float32, Float32x4)", llssa.SIMDUnimplemented},
		{"different mask lanes", "Float32x4", "StoreArrayMasked(*[4]float32, Mask64x2)", llssa.SIMDUnimplemented},
		{"different memory shape", "Float32x4", "StoreArrayMasked(*[8]float32, Mask32x4)", llssa.SIMDUnimplemented},
		{"widen", "Uint32x4", "MulWidenLo(Uint32x4) Uint64x2", llssa.SIMDMulWidenLo},
		{"missing widened result", "Uint32x4", "MulWidenLo(Uint32x4)", llssa.SIMDUnimplemented},
		{"nonvector widened result", "Uint32x4", "MulWidenLo(Uint32x4) uint64", llssa.SIMDUnimplemented},
		{"unchanged lane width", "Uint32x4", "MulWidenLo(Uint32x4) Uint32x4", llssa.SIMDUnimplemented},
		{"floating widened result", "Uint32x4", "MulWidenLo(Uint32x4) Float64x2", llssa.SIMDUnimplemented},
		{"changed signedness", "Uint32x4", "MulWidenLo(Uint32x4) Int64x2", llssa.SIMDUnimplemented},
		{"carryless selection", "Uint64x2", "carrylessMultiply(uint8, Uint64x2) Uint64x2", llssa.SIMDCarrylessMultiply},
		{"wrong selection type", "Uint64x2", "carrylessMultiply(uint64, Uint64x2) Uint64x2", llssa.SIMDUnimplemented},
		{"wrong polynomial width", "Uint32x4", "carrylessMultiply(uint8, Uint32x4) Uint32x4", llssa.SIMDUnimplemented},
		{"carryless low", "Uint64x2", "carrylessMultiplyWidenLo(Uint64x2) Uint64x2", llssa.SIMDCarrylessMultiplyLo},
		{"integer FMA", "Uint32x4", "MulAdd(Uint32x4, Uint32x4) Uint32x4", llssa.SIMDUnimplemented},
		{"floating average", "Float32x4", "Average(Float32x4) Float32x4", llssa.SIMDUnimplemented},
		{"rotate count", "Uint32x4", "RotateAllLeft(uint64) Uint32x4", llssa.SIMDRotateAllLeft},
		{"wrong rotate count", "Uint32x4", "RotateAllLeft(uint32) Uint32x4", llssa.SIMDUnimplemented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, "simd.go", declarations+"func (x "+tc.receiver+") "+tc.method, 0)
			if err != nil {
				t.Fatal(err)
			}
			pkg, err := new(types.Config).Check("simd/archsimd", fs, []*ast.File{file}, nil)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssa.NewProgram(fs, 0)
			prog.CreatePackage(pkg, nil, nil, true)
			method := pkg.Scope().Lookup(tc.receiver).Type().(*types.Named).Method(0)
			op, ok := lookupSIMD(prog.FuncValue(method), "amd64")
			wantOK := tc.want != llssa.SIMDUnimplemented
			if ok != wantOK || ok && op.op != tc.want {
				t.Fatalf("operation = %v (%v), want %v", op.op, ok, tc.want)
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

func TestSIMDLookupOrZeroTargets(t *testing.T) {
	const source = `package archsimd
 type v128 struct { _ [0]func() }
 type Int8x16 struct { tag v128; vals [16]int8 }
 func (x Int8x16) LookupOrZero(y Int8x16) Int8x16
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
	vector := pkg.Pkg.Scope().Lookup("Int8x16").Type().(*types.Named)
	fn := pkg.Prog.FuncValue(vector.Method(0))
	for _, arch := range []string{"arm64", "wasm", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			want := llssa.SIMDLookupOrZero
			if arch == "amd64" {
				want = llssa.SIMDUnimplemented
			}
			op, ok := lookupSIMD(fn, arch)
			if !ok || op.op != want {
				t.Fatalf("lookup = (%v, %v), want %v", op.op, ok, want)
			}
		})
	}
}

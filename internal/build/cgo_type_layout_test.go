package build

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	gobuild "go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

type cLayoutTarget struct{ os, arch, triple, profile string }

func cLayoutTargets() []cLayoutTarget {
	return []cLayoutTarget{
		{"linux", "amd64", "x86_64-unknown-linux-gnu", ""},
		{"linux", "386", "i386-unknown-linux-gnu", ""},
		{"linux", "arm64", "aarch64-unknown-linux-gnu", ""},
		{"linux", "arm", "armv7-unknown-linux-gnueabihf", ""},
		{"darwin", "amd64", "x86_64-apple-darwin", ""},
		{"darwin", "arm64", "arm64-apple-darwin", ""},
		{"windows", "amd64", "x86_64-pc-windows-msvc", ""},
		{"windows", "386", "i686-pc-windows-msvc", ""},
		{"windows", "arm64", "aarch64-pc-windows-msvc", ""},
		{"js", "wasm", "wasm32-unknown-emscripten", "j32"},
		{"js", "wasm", "wasm64-unknown-emscripten", "j64"},
		{"wasip1", "wasm", "wasm32-unknown-wasip1", "w32"},
	}
}

// Cache immutable IR bytes, not LLVM objects: each caller still owns its
// context/module and disposes them through t.Cleanup.
var cLayoutOracleCache = struct {
	sync.Mutex
	ir map[string][]byte
}{ir: make(map[string][]byte)}

// Use the same compiler constructor and argument merging as LLGo C preambles.
func cLayoutOracle(t *testing.T, target cLayoutTarget) llvm.Module {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	ctx := &context{buildConf: &Config{}, commands: commandEnv{environ: os.Environ()},
		crossCompile: crosscompile.Export{CC: clang, CCFLAGS: []string{"--target=" + target.triple}}}
	cmd := ctx.compiler()
	args := []string{"-std=c11", "-S", "-emit-llvm", "testdata/typelayout/oracle.c"}
	// Include the compiler and effective flags (including CFLAGS/CCFLAGS),
	// while excluding the caller-specific output path.
	key := clang + "\x00" + strings.Join(cmd.CompileArguments(args...), "\x00")
	ir := func() []byte {
		cLayoutOracleCache.Lock()
		defer cLayoutOracleCache.Unlock()
		if ir, ok := cLayoutOracleCache.ir[key]; ok {
			return ir
		}
		out := filepath.Join(t.TempDir(), "compiled-oracle.ll")
		var diagnostics bytes.Buffer
		cmd.Stderr = &diagnostics
		if err := cmd.Compile(append(slices.Clone(args), "-o", out)...); err != nil {
			t.Fatalf("C oracle %s: %v\n%s", target.triple, err, diagnostics.String())
		}
		ir, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		cLayoutOracleCache.ir[key] = ir
		return ir
	}()
	out := filepath.Join(t.TempDir(), "oracle.ll")
	if err := os.WriteFile(out, ir, 0600); err != nil {
		t.Fatal(err)
	}
	llvmctx := llvm.NewContext()
	t.Cleanup(llvmctx.Dispose)
	buf, err := llvm.NewMemoryBufferFromFile(out)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := llvmctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	return mod
}

func cLayoutConstant(t *testing.T, mod llvm.Module, name string) uint64 {
	t.Helper()
	g := mod.NamedGlobal(name)
	if g.IsNil() || g.Initializer().IsNil() {
		t.Fatalf("C oracle missing %s", name)
	}
	return g.Initializer().ZExtValue()
}

func cgoLayoutTypeNames(t *testing.T, mod llvm.Module) []string {
	t.Helper()
	names := []string{}
	for g := mod.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
		// LongDouble is a C-only oracle entry; cmd/cgo cannot represent it.
		if strings.HasPrefix(g.Name(), "value_") && g.Name() != "value_LongDouble" {
			names = append(names, strings.TrimPrefix(g.Name(), "value_"))
		}
	}
	if want := int(cLayoutConstant(t, mod, "audit_type_count")); want == 0 || len(names) != want {
		t.Fatalf("oracle has %d cgo types, AUDIT_TYPES in types.h defines %d", len(names), want)
	}
	return names
}

// Check the C compiler's final IR against its constant expressions, rather than
// importing sizeof into Go and letting go/types fold both sides of the test.
func TestCTypeLLVMLayout(t *testing.T) {
	for _, target := range cLayoutTargets() {
		t.Run(target.triple, func(t *testing.T) {
			mod := cLayoutOracle(t, target)
			td := llvm.NewTargetData(mod.DataLayout())
			defer td.Dispose()
			count := 0
			for g := mod.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
				if !strings.HasPrefix(g.Name(), "value_") {
					continue
				}
				name := strings.TrimPrefix(g.Name(), "value_")
				count++
				typ := g.GlobalValueType()
				if got, want := td.TypeAllocSize(typ), cLayoutConstant(t, mod, "size_"+name); got != want {
					t.Errorf("%s LLVM size=%d, C sizeof=%d", name, got, want)
				}
				if got, want := uint64(td.ABITypeAlignment(typ)), cLayoutConstant(t, mod, "align_"+name); got != want {
					t.Errorf("%s LLVM alignment=%d, C alignof=%d", name, got, want)
				}
				if got, want := td.TypeAllocSize(mod.NamedGlobal("array_"+name).GlobalValueType()), 2*td.TypeAllocSize(typ); got != want {
					t.Errorf("%s array=%d, want %d", name, got, want)
				}
			}
			// AUDIT_TYPES plus the C-only LongDouble entry in oracle.c.
			if want := int(cLayoutConstant(t, mod, "audit_type_count")) + 1; count != want {
				t.Fatalf("checked %d C globals, want %d (AUDIT_TYPES plus LongDouble)", count, want)
			}
		})
	}
}

func cgoLayoutTypes(t *testing.T, target cLayoutTarget, cc string, flags ...string) *types.Package {
	t.Helper()
	dir := t.TempDir()
	source, err := filepath.Abs("testdata/typelayout/cgo.go")
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{"tool", "cgo", "-godefs", "-objdir", dir, "--", "-I" + filepath.Dir(source)}, flags...)
	args = append(args, source)
	cmd := exec.Command("go", args...)
	// cmd/cgo parses CC as a quoted command; LLVM installations on Windows
	// commonly live under Program Files. Quoted fields preserve backslashes.
	cmd.Env = withEnv(os.Environ(), "CC=\""+cc+"\"", "GOOS="+target.os, "GOARCH="+target.arch)
	output, err := cmd.Output()
	if err != nil {
		var failure *exec.ExitError
		if errors.As(err, &failure) {
			t.Fatalf("cgo %s: %v\n%s", target.triple, err, failure.Stderr)
		}
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "cgo.go", output, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", target.arch)}
	pkg, err := conf.Check("layout", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("cgo Go declarations: %v\n%s", err, output)
	}
	return pkg
}

// Compare each preprocessed C type's actual LLGo global against an independent
// C compilation. The return value permits a negative ABI mismatch test.
func cgoLayoutDifferences(t *testing.T, target cLayoutTarget, goPkg *types.Package, mod llvm.Module, names []string) []string {
	t.Helper()
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(&llssa.Target{GOOS: target.os, GOARCH: target.arch, LLVMTarget: target.triple, WasmProfile: target.profile})
	defer prog.Dispose()
	prog.TypeSizes(types.SizesFor("gc", target.arch))
	pkg := prog.NewPackage("layout", "layout")
	td := prog.TargetData()
	diffs := []string{}
	if names == nil {
		names = cgoLayoutTypeNames(t, mod)
	}
	for _, name := range names {
		obj, ok := goPkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			diffs = append(diffs, "missing Go type "+name)
			continue
		}
		cGlobal := mod.NamedGlobal("value_" + name)
		if cGlobal.IsNil() {
			t.Fatalf("C oracle missing value_%s", name)
		}
		// Named C records retain C field alignment (e.g. Windows i686 double).
		prog.SetTypeCallConv("layout."+name, llssa.CDecl)
		pkg.NewVar("layout."+name, types.NewPointer(obj.Type()), llssa.CDecl)
		physical := pkg.Module().NamedGlobal("layout." + name).GlobalValueType()
		got, want := td.TypeAllocSize(physical), cLayoutConstant(t, mod, "size_"+name)
		if got != want {
			diffs = append(diffs, fmt.Sprintf("%s: LLVM %s size=%d, C sizeof=%d", name, physical, got, want))
		}
		// Clang stores _Bool as i8, whereas LLGo uses i1 with a one-byte
		// allocation. Both representations have the same memory footprint.
		if typ := cGlobal.GlobalValueType(); name != "Bool" && name != "Record" && physical.String() != typ.String() {
			diffs = append(diffs, fmt.Sprintf("%s: LLVM type=%s, C LLVM type=%s", name, physical, typ))
		}
		pkg.NewVar("layout.array_"+name, types.NewPointer(types.NewArray(obj.Type(), 2)), llssa.CDecl)
		array := pkg.Module().NamedGlobal("layout.array_" + name).GlobalValueType()
		if size := td.TypeAllocSize(array); size != 2*want {
			diffs = append(diffs, fmt.Sprintf("%s: LLVM array bytes=%d, C array bytes=%d", name, size, 2*want))
		}
		if name == "Record" {
			st := obj.Type().Underlying().(*types.Struct)
			for i := range st.NumFields() {
				field := st.Field(i).Name()
				if field != "D" && field != "Tail" {
					continue
				}
				if got, want := td.ElementOffset(physical, i), cLayoutConstant(t, mod, "offset_Record_"+field); got != want {
					diffs = append(diffs, fmt.Sprintf("Record.%s: LLVM offset=%d, C offset=%d", field, got, want))
				}
			}
		}
	}
	return diffs
}

// Supplement the cross-target godefs checks with the full, unmodified cgo
// frontend path on the native CI target. Inspect globals before linking.
func TestCgoPackageLLVMLayout(t *testing.T) {
	conf := NewDefaultConf(ModeGen)
	pkgs, err := Do([]string{"./testdata/typelayout"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	target := cLayoutTarget{conf.Goos, conf.Goarch, pkg.Prog.Target().Spec().Triple, ""}
	oracle := cLayoutOracle(t, target)
	td := pkg.Prog.TargetData()
	for _, name := range cgoLayoutTypeNames(t, oracle) {
		g := oracle.NamedGlobal("value_" + name)
		got := pkg.Module().NamedGlobal("main.Value" + name)
		if got.IsNil() {
			t.Fatalf("missing cgo global Value%s", name)
		}
		physical := got.GlobalValueType()
		if size, want := td.TypeAllocSize(physical), cLayoutConstant(t, oracle, "size_"+name); size != want {
			t.Errorf("%s: cgo -> LLVM %s size=%d, C sizeof=%d", name, physical, size, want)
		}
		if typ := g.GlobalValueType(); name != "Bool" && name != "Record" && physical.String() != typ.String() {
			t.Errorf("%s: cgo -> LLVM type=%s, C LLVM type=%s", name, physical, typ)
		}
	}
}

func TestCgoTypesLLVMLayout(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range cLayoutTargets() {
		// cmd/cgo has no wasm ptrSize entry; the C oracle and C binding tests still
		// check all three hosted wasm profiles without pretending cgo supports them.
		t.Run(target.triple, func(t *testing.T) {
			if target.arch == "wasm" {
				t.Skip("cmd/cgo does not support GOARCH=wasm; C and runtime binding IR tested separately")
			}
			pkg := cgoLayoutTypes(t, target, clang, "--target="+target.triple)
			if diffs := cgoLayoutDifferences(t, target, pkg, cLayoutOracle(t, target), nil); len(diffs) != 0 {
				t.Fatal(strings.Join(diffs, "\n"))
			}
		})
	}
}

func TestCgoCompilerLayoutMismatch(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	target := cLayoutTarget{"linux", "amd64", "x86_64-unknown-linux-gnu", ""}
	pkg := cgoLayoutTypes(t, target, clang, "--target="+target.triple, "-fshort-enums")
	diffs := cgoLayoutDifferences(t, target, pkg, cLayoutOracle(t, target), nil)
	if !slices.ContainsFunc(diffs, func(s string) bool { return strings.HasPrefix(s, "Enum:") }) {
		t.Fatalf("missed incompatible cgo compiler options: %v", diffs)
	}
}

func TestCgoGCCAndLLGoClangLayout(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux CI supplies a distinct GCC frontend")
	}
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range cLayoutTargets() {
		if target.os == runtime.GOOS && target.arch == runtime.GOARCH {
			pkg := cgoLayoutTypes(t, target, gcc)
			if diffs := cgoLayoutDifferences(t, target, pkg, cLayoutOracle(t, target), nil); len(diffs) != 0 {
				t.Fatal(strings.Join(diffs, "\n"))
			}
			return
		}
	}
	t.Fatalf("no native GCC target for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func TestRuntimeCTypeLLVMLayout(t *testing.T) {
	for _, target := range cLayoutTargets() {
		t.Run(target.triple, func(t *testing.T) {
			dir := filepath.Join("..", "..", "runtime", "internal", "clite")
			ctx := gobuild.Default
			ctx.GOOS = target.os
			ctx.GOARCH = target.arch
			ctx.CgoEnabled = false
			if target.profile == "j64" {
				ctx.BuildTags = []string{"llgo.wasm.emscripten.memory64"}
			}
			selected, err := ctx.ImportDir(dir, 0)
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			files := []*ast.File{}
			for _, name := range selected.GoFiles {
				f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, f)
			}
			conf := types.Config{Importer: importer.Default(), IgnoreFuncBodies: true, Sizes: types.SizesFor("gc", target.arch)}
			pkg, err := conf.Check("layout", fset, files, nil)
			if err != nil {
				t.Fatal(err)
			}
			// Explicitly require the runtime aliases represented by the C
			// fixture, so a rename/removal reports the missing type by name.
			names := []string{"Char", "Int", "Uint", "Long", "Ulong", "LongLong", "UlongLong",
				"SizeT", "SsizeT", "IntptrT", "UintptrT", "Float", "Double", "Pointer"}
			if diffs := cgoLayoutDifferences(t, target, pkg, cLayoutOracle(t, target), names); len(diffs) != 0 {
				t.Fatal(strings.Join(diffs, "\n"))
			}
		})
	}
}

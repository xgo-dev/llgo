//go:build !llgo

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/optlevel"
	llssa "github.com/goplus/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestFrontendOptions(t *testing.T) {
	oldDebug := enableDbg
	oldDebugSymbols := enableDbgSyms
	oldTrace := enableCallTracing
	oldExportRename := enableExportRename
	t.Cleanup(func() {
		enableDbg = oldDebug
		enableDbgSyms = oldDebugSymbols
		enableCallTracing = oldTrace
		enableExportRename = oldExportRename
	})

	EnableDebug(true)
	EnableDbgSyms(true)
	EnableTrace(true)
	EnableExportRename(true)
	t.Setenv("LLGO_SHADOW_STACK", "1")

	wantLegacy := Options{
		Debug:        true,
		DebugSymbols: true,
		Trace:        true,
		ExportRename: true,
		ShadowStack:  true,
	}
	if got := (&context{}).frontendOptions(); got != wantLegacy {
		t.Fatalf("frontendOptions() = %+v, want legacy options %+v", got, wantLegacy)
	}
	if got := (*context)(nil).frontendOptions(); got != wantLegacy {
		t.Fatalf("nil frontendOptions() = %+v, want legacy options %+v", got, wantLegacy)
	}

	wantExplicit := Options{Trace: true}
	ctx := &context{options: wantExplicit, optionsSet: true}
	if got := ctx.frontendOptions(); got != wantExplicit {
		t.Fatalf("frontendOptions() = %+v, want explicit options %+v", got, wantExplicit)
	}
}

func compileDebugSource(t *testing.T, source string, level optlevel.Level) llssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "debug_compile.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset,
		types.NewPackage("debugcompile", "debugcompile"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug,
	)
	if err != nil {
		t.Fatal(err)
	}

	prog := newLLSSAProgForTarget(t, &llssa.Target{
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		OptLevel: level,
	})
	t.Cleanup(prog.Dispose)
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{
		Debug:        true,
		DebugSymbols: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestCompileDebugMetadata(t *testing.T) {
	const source = `package debugcompile

type item struct { value int }

func inspect(items [2]item, seed int) int {
	x := seed + 1
	var local [1]item
	if x > 0 {
		items[0].value = x
		local[0].value = x
	}
	return items[0].value + local[0].value
}

var anonymous = func(seed int) int {
	value := seed + 1
	return value
}
`
	pkg := compileDebugSource(t, source, optlevel.O0)
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("debug module is invalid: %v\n%s", err, pkg.Module().String())
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"#dbg_declare",
		"#dbg_value",
		"DILexicalBlock",
		`name: "items", arg: 1`,
		`name: "seed", arg: 2`,
		`name: "x"`,
		`name: "local"`,
		`name: "value"`,
		"isOptimized: false",
	} {
		if !strings.Contains(ir, want) {
			t.Errorf("debug module is missing %q:\n%s", want, ir)
		}
	}
}

func TestCompileNestedRangeFuncDeferDebugMetadata(t *testing.T) {
	const source = `package debugcompile

func outer(yield func(int) bool) { _ = yield(1) }

func inner(base int) func(func(int) bool) {
	return func(yield func(int) bool) { _ = yield(base + 10) }
}

func f() {
	for i := range outer {
		defer func() { _ = i }()
		for j := range inner(i) {
			defer func(v int) { _ = v }(j)
		}
	}
}
`
	pkg := compileDebugSource(t, source, optlevel.O2)
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("nested range defer debug module is invalid: %v\n%s", err, pkg.Module().String())
	}
}

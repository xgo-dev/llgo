//go:build !llgo

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
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
		OptLevel: optlevel.O0,
	})
	defer prog.Dispose()
	pkg, _, err := newPackageEx(prog, nil, nil, nil, ssaPkg, []*ast.File{file}, nil, false, Options{
		Debug:        true,
		DebugSymbols: true,
	})
	if err != nil {
		t.Fatal(err)
	}
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

func TestTypeAssertDebugDiagnostics(t *testing.T) {
	const source = `package foo

type methoder interface { M() }

func concrete(v any) int {
	return v.(int)
}

func dynamic(v any) methoder {
	return v.(methoder)
}
`
	ssaPkg, _, files := buildGoSSAPkg(t, source)
	prog := newLLSSAProg(t)
	defer prog.Dispose()

	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderr
	SetDebug(DbgFlagTypeAssert | DbgFlagNoErrorColumn)
	defer func() {
		SetDebug(0)
		os.Stderr = oldStderr
	}()

	if _, err := NewPackage(prog, ssaPkg, files); err != nil {
		t.Fatal(err)
	}
	os.Stderr = oldStderr
	if err := stderr.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"foo.go:6: type assertion inlined\n",
		"foo.go:10: type assertion not inlined\n",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("stderr does not contain %q:\n%s", want, got)
		}
	}
}

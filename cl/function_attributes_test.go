//go:build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	abiLower "github.com/xgo-dev/llgo/internal/abi"
	"github.com/xgo-dev/llgo/internal/cabi"
	"github.com/xgo-dev/llgo/internal/typepatch"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func checkFunctionAttributes(t *testing.T, fn llvm.Value, cold, noreturn bool) {
	t.Helper()
	if fn.IsNil() {
		t.Fatal("missing LLVM function")
	}
	for name, want := range map[string]bool{"cold": cold, "noreturn": noreturn} {
		if got := !fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil(); got != want {
			t.Errorf("%s: %s = %v, want %v\n%s", fn.Name(), name, got, want, fn.String())
		}
	}
}

func TestFunctionAttributesDefinitions(t *testing.T) {
	const source = `package attr
// llgo:cold
//llgo:cold
func Rare() {}
//llgo:noreturn
func Stop() { for {} }
//llgo:cold
//llgo:noreturn
func Generic[T any](x T) { for {} }
type T struct{}
//llgo:cold
func (T) Method() {}
//llgo:noreturn
func (*T) Stop() { for {} }
func Plain() {}
func Use() { Generic(1) }
`
	goPkg, _, files := buildGoSSAPkg(t, source)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg, err := NewPackage(prog, goPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		cold, noreturn bool
	}{
		{"Rare", true, false}, {"Stop", false, true}, {"Generic[int]", true, true},
		{"T.Method", true, false}, {"(*T).Stop", false, true}, {"Plain", false, false},
	} {
		checkFunctionAttributes(t, pkg.Module().NamedFunction("attr."+tc.name), tc.cold, tc.noreturn)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionAttributeDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{"//llgo:cold\nvar x int", "requires a named function"},
		{"func F() {\n//llgo:noreturn\nvar x int; _ = x\n}", "requires a named function"},
		{"//llgo:cold(true)\nfunc F() {}", "takes no arguments"},
		{"//llgo:noreturn cold\nfunc F() {}", "takes no arguments"},
		{"//llgo:result nonnull\nfunc F() *int { return nil }", "not yet supported"},
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "bad.go", "package p\n"+tc.source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		prog := newLLSSAProg(t)
		err = ParsePkgSyntax(prog, fset, types.NewPackage("p", "p"), []*ast.File{file})
		prog.Dispose()
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "bad.go:") {
			t.Errorf("%s: got %v, want %q with source position", tc.source, err, tc.want)
		}
	}
}

// The dependency has no Go SSA syntax in the caller. Only the shared source
// index can supply its properties; no backend LLVM object is shared.
func TestFunctionAttributesImportedDeclaration(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(name, text string) *ast.File {
		file, err := parser.ParseFile(fset, name, text, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	depFile := parse("dep.go", "package dep\n//llgo:cold\n//llgo:noreturn\nfunc Stop()\n")
	dep, err := (&types.Config{}).Check("dep", fset, []*ast.File{depFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	callerFile := parse("caller.go", "package caller\nimport \"dep\"\nvar Sink int\nfunc Call() { dep.Stop(); Sink = 1 }\n")
	info := newLocalityTypeInfo()
	caller, err := (&types.Config{Importer: importerFunc(func(string) (*types.Package, error) { return dep, nil })}).Check("caller", fset, []*ast.File{callerFile}, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	goProg.CreatePackage(dep, nil, nil, true)
	goPkg := goProg.CreatePackage(caller, []*ast.File{callerFile}, info, true)
	goPkg.Build()
	coordinator := newLLSSAProg(t)
	defer coordinator.Dispose()
	options := Options{}
	if err := ParsePkgSyntaxWithOptions(coordinator, fset, dep, []*ast.File{depFile}, options); err != nil {
		t.Fatal(err)
	}
	coordinator.PackageDirectives(dep).BindScope(dep)
	if err := ParsePkgSyntax(coordinator, fset, caller, []*ast.File{callerFile}); err != nil {
		t.Fatal(err)
	}
	options.PreloadedSyntax = true
	for i := 0; i < 2; i++ {
		backend := coordinator.NewBackendProgram()
		defer backend.Dispose()
		pkg, _, err := NewPackageExWithEmbedMetaOptions(backend, nil, nil, nil, goPkg, []*ast.File{callerFile}, nil, false, options)
		if err != nil {
			t.Fatal(err)
		}
		checkFunctionAttributes(t, pkg.Module().NamedFunction("dep.Stop"), true, true)
		if !pkg.Module().NamedFunction("dep.Stop").GetEnumFunctionAttribute(llvm.AttributeKindID("nounwind")).IsNil() {
			t.Fatal("noreturn must not imply nounwind")
		}
		passes := llvm.NewPassBuilderOptions()
		passes.SetVerifyEach(true)
		err = pkg.Module().RunPasses("function(simplifycfg)", backend.TargetMachine(), passes)
		passes.Dispose()
		if err != nil {
			t.Fatal(err)
		}
		body := pkg.Module().NamedFunction("caller.Call").String()
		if !strings.Contains(body, "unreachable") || strings.Contains(body, "store ") {
			t.Fatalf("noreturn did not remove the normal continuation:\n%s", body)
		}
	}
}

func TestFunctionAttributesPackagePatch(t *testing.T) {
	for _, linkname := range []string{"", "patchedF"} {
		for _, preloaded := range []bool{false, true} {
			fset := token.NewFileSet()
			goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
			build := func(path, source string) (*ssa.Package, *ast.File) {
				file, err := parser.ParseFile(fset, path+".go", "package p\n"+source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				info := newLocalityTypeInfo()
				owner, err := (&types.Config{}).Check(path, fset, []*ast.File{file}, info)
				if err != nil {
					t.Fatal(err)
				}
				pkg := goProg.CreatePackage(owner, []*ast.File{file}, info, true)
				pkg.Build()
				return pkg, file
			}
			original, originalFile := build("p", "//llgo:noreturn\nfunc F() { for {} }\nfunc Call() { F() }")
			replacement := "//llgo:cold\nfunc F() {}"
			if linkname != "" {
				replacement = "//llgo:link F " + linkname + "\n" + replacement
			}
			alternate, alternateFile := build(abi.PatchPathPrefix+"p", replacement)
			patches := Patches{"p": {Alt: alternate, Types: typepatch.Clone(alternate.Pkg)}}
			files := []*ast.File{originalFile, alternateFile}
			prog := newLLSSAProg(t)
			defer prog.Dispose()
			options := Options{}
			if preloaded {
				if err := ParsePkgSyntaxWithOptions(prog, fset, patches["p"].Types, files, options); err != nil {
					t.Fatal(err)
				}
				options.PreloadedSyntax = true
			}
			pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, patches, nil, original, files, nil, false, options)
			if err != nil {
				t.Fatal(err)
			}
			symbol := linkname
			if symbol == "" {
				symbol = "p.F"
			}
			checkFunctionAttributes(t, pkg.Module().NamedFunction(symbol), true, false)
			caller := pkg.Module().NamedFunction("p.Call").String()
			if !strings.Contains(caller, "@"+symbol+"(") || strings.Contains(caller, "unreachable") {
				t.Fatalf("call did not use the replacement symbol and attributes:\n%s", caller)
			}
		}
	}

}
func TestFunctionAttributesDoNotMergeSourceAliases(t *testing.T) {
	goPkg, _, _ := buildGoSSAPkg(t, `package p
import _ "unsafe"
//go:linkname F shared
//llgo:cold
func F() {}
//go:linkname G shared
func G() {}
`)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	for _, name := range []string{"F", "G"} {
		prog.Directives().Function(goPkg.Func(name).Syntax().(*ast.FuncDecl))
	}
	ctx := &context{prog: prog}
	f, g := ctx.sourceFunction(goPkg.Func("F")).Decl, ctx.sourceFunction(goPkg.Func("G")).Decl
	if !f.Cold || g.Cold || f.NoReturn || g.NoReturn {
		t.Fatal("source properties crossed linkname aliases")
	}
}

func TestFunctionAttributesSurviveABIConversion(t *testing.T) {
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOARCH, func(t *testing.T) {
			goPkg, _, files := buildGoSSAPkg(t, `package attr
//llgo:cold
//llgo:noreturn
func Stop(value [65537]byte) [65537]byte
`)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, goPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			mod.SetDataLayout(prog.DataLayout())
			mod.SetTarget(target.Spec().Triple)
			td := llvm.NewTargetData(prog.DataLayout())
			defer td.Dispose()
			abiLower.LowerLargeAggregates(td, mod, abiLower.AggregateLoweringConfig{})
			checkFunctionAttributes(t, mod.NamedFunction("attr.Stop"), true, true)
			cabi.NewTransformer(prog, target.Spec().Triple, "", false).TransformModule("attr", mod)
			checkFunctionAttributes(t, mod.NamedFunction("attr.Stop"), true, true)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFunctionAttributesEmbeddedInterfaceMethod(t *testing.T) {
	goPkg, _, files := buildGoSSAPkg(t, `package attr
type T struct { interfaceValue }
type interfaceValue = interface { Stop() }
func Box(value T) any { return value }
`)
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	pkg, err := NewPackage(prog, goPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionAttributesAliasReceiver(t *testing.T) {
	for _, tc := range []struct{ name, alias, receiver, symbol string }{
		{"value", "T", "Alias", "T.Stop"},
		{"pointer_alias", "*T", "Alias", "(*T).Stop"},
		{"pointer_to_alias", "T", "*Alias", "(*T).Stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			goPkg, _, files := buildGoSSAPkg(t, `package attr
type T struct{}
type Alias = `+tc.alias+`
//llgo:cold
//llgo:noreturn
func (`+tc.receiver+`) Stop() { for {} }
func Box(value *T) any { return value }
`)
			prog := newLLSSAProg(t)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, goPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			checkFunctionAttributes(t, pkg.Module().NamedFunction("attr."+tc.symbol), true, true)
			// The backend also looks up methods without their Go SSA syntax.
			typ := goPkg.Pkg.Scope().Lookup("T").Type()
			method := types.NewMethodSet(types.NewPointer(typ)).Lookup(goPkg.Pkg, "Stop").Obj().(*types.Func)
			properties := prog.FunctionDeclaration(goPkg.Pkg, method, nil)
			if properties == nil || !properties.Cold || !properties.NoReturn {
				t.Fatal("alias receiver lost source attributes")
			}

			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

//go:build !llgo

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goembed"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestFunctionPropertiesSurviveCommentRemoval(t *testing.T) {
	const source = `package p
//go:noinline
//go:nosplit
func marked() {}
func plain() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "properties.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := newLocalityTypeInfo()
	imp := importer.Default()
	pkg, err := (&types.Config{Importer: imp}).Check("example.com/p", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgramEx(t, nil, imp)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, fset, pkg, files); err != nil {
		t.Fatal(err)
	}
	prog.PackageDirectives(pkg).Bind(info)
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()
	// Clear the syntax before either analysis or lowering; neither is allowed to
	// rediscover directives from the comments.
	for _, d := range file.Decls {
		d.(*ast.FuncDecl).Doc = nil
	}
	file.Comments = nil
	tracking := NewCallerTracking(prog.Directives())
	tracking.Precompute([]*ssa.Package{ssaPkg})
	prog.Directives().Freeze()
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, tracking, nil, nil, ssaPkg, files, goembed.VarMap{}, false, Options{PreloadedSyntax: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	marked := compiled.Module().NamedFunction("example.com/p.marked").String()
	if !strings.Contains(marked, "noinline") || !strings.Contains(compiled.String(), `"disable-tail-calls"="true"`) {
		t.Fatalf("missing properties:\n%s", marked)
	}
	if strings.Contains(compiled.Module().NamedFunction("example.com/p.plain").String(), "noinline") {
		t.Fatal("plain function inherited noinline")
	}
}

func TestLinknameRecordsKeepPackageIdentity(t *testing.T) {
	prog := ssatest.NewProgramEx(t, nil, importer.Default())
	defer prog.Dispose()
	for _, target := range []string{"C.first", "C.second"} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "variant.go", "package p\n//llgo:link F "+target+"\nfunc F() {}\n", parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg := types.NewPackage("example.com/p", "p")
		if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
			t.Fatal(err)
		}
		// Verify the earlier variant even after the global compatibility index has
		// been overwritten by another package with exactly the same path.
		defer func() {
			if got, ok := prog.LinknameFor(pkg, nil, "example.com/p.F"); !ok || got != target {
				t.Errorf("variant %s = %s, %v", target, got, ok)
			}
		}()
	}
}

func TestLinknameRecordsSelectPatchReplacement(t *testing.T) {
	prog := ssatest.NewProgramEx(t, nil, importer.Default())
	defer prog.Dispose()
	fset := token.NewFileSet()
	parse := func(source string) *ast.File {
		file, err := parser.ParseFile(fset, "patch.go", source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	original := parse("package p\nfunc F() {}\n")
	replacement := parse("package p\n//llgo:link F C.replacement\nfunc F() {}\n")
	info := newLocalityTypeInfo()
	pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{original}, info)
	if err != nil {
		t.Fatal(err)
	}
	patch := types.NewPackage(pkg.Path(), pkg.Name())
	if err := ParsePkgSyntax(prog, fset, patch, []*ast.File{original, replacement}); err != nil {
		t.Fatal(err)
	}
	prog.PackageDirectives(patch).Bind(info)
	ctx := &context{patches: Patches{pkg.Path(): {Types: patch}}}
	if got, ok := prog.LinknameFor(ctx.directivePackage(pkg), pkg.Scope().Lookup("F"), pkg.Path()+".F"); !ok || got != "C.replacement" {
		t.Fatalf("patched link = %q, %v", got, ok)
	}
}

func TestFunctionPropertiesUseGenericOrigin(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generic.go", `package p
//go:noinline
func F[T any](v T) T { return v }
func Use() int { return F(1) }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	info.Instances = make(map[*ast.Ident]types.Instance)
	pkg, err := new(types.Config).Check("example.com/p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgramEx(t, nil, importer.Default())
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	prog.PackageDirectives(pkg).Bind(info)
	goProg := ssa.NewProgram(fset, ssa.InstantiateGenerics|ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()
	prog.Directives().Freeze()
	ctx := &context{prog: prog}
	for _, block := range ssaPkg.Func("Use").Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(*ssa.Call); ok {
				fn := call.Common().StaticCallee()
				source := ctx.sourceFunction(fn)
				if fn == nil || fn.Origin() == nil || source.Decl == nil || !source.Decl.NoInline {
					t.Fatalf("generic instance lost source property: %v", fn)
				}
				if source.SSA != fn || source.SSA.Signature.Params().At(0).Type() != types.Typ[types.Int] {
					t.Fatal("generic source association lost the instantiated signature")
				}
				if source.Decl != ctx.sourceFunction(fn.Origin()).Decl {
					t.Fatal("generic instance did not retain the origin declaration")
				}
				return
			}
		}
	}
	t.Fatal("missing generic call")
}

func TestStandalonePropertiesPreparedWithoutFiles(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "dependency.go", "package dep\n//go:uintptrescapes\nfunc F(p uintptr) {}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	pkg, err := new(types.Config).Check("example.com/dep", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, []*ast.File{file}, info, true)
	ssaPkg.Build()
	prog := ssatest.NewProgramEx(t, nil, importer.Default())
	defer prog.Dispose()
	ctx := &context{prog: prog, goProg: goProg, fset: fset}
	ctx.prepareImportSources()
	file.Decls[0].(*ast.FuncDecl).Doc = nil
	prog.Directives().Freeze()
	if !ctx.sourceFunction(ssaPkg.Func("F")).Decl.UintptrEscapes {
		t.Fatal("standalone dependency lost prepared uintptr property")
	}
}

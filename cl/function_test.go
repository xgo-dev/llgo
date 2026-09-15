package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestFunctionWrappersKeepAliasedDeclarationsIndependent(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "aliases.go", `package p
import _ "unsafe"
//go:linkname Env shared
//llgo:env
func Env() {}
//go:linkname Plain shared
func Plain() {}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	files := []*ast.File{file}
	owner, err := (&types.Config{Importer: importer.Default()}).Check("p", fs, files, info)
	if err != nil {
		t.Fatal(err)
	}
	coordinator := newLLSSAProg(t)
	defer coordinator.Dispose()
	if err = ParsePkgSyntax(coordinator, fs, owner, files); err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fs, ssa.SanityCheckFunctions)
	for _, dep := range owner.Imports() {
		goProg.CreatePackage(dep, nil, nil, true)
	}
	goPkg := goProg.CreatePackage(owner, files, info, true)
	goPkg.Build()
	for i := 0; i < 2; i++ {
		backend := coordinator.NewBackendProgram()
		defer backend.Dispose()
		ctx := &context{prog: backend, pkg: backend.NewPackage("p", "p"), fset: fs, goTyps: owner, goProg: goProg, goPkg: goPkg}
		env := ctx.function(goPkg.Func("Env"))
		plain := ctx.function(goPkg.Func("Plain"))
		if env == plain || env.declaration == plain.declaration {
			t.Fatal("linkname aliases share a source object")
		}
		if !env.declaration.HasExplicitEnv() || plain.declaration.HasExplicitEnv() {
			t.Fatal("env crossed source declarations")
		}
		for _, fn := range []*aFunction{env, plain} {
			if _, name, _ := ctx.funcName(fn); name != "shared" {
				t.Fatalf("linkname = %q", name)
			}
		}
		if _, _, kind := ctx.compileFuncDecl(ctx.pkg, env); kind != goFunc {
			t.Fatal("env function not compiled")
		}
		func() {
			defer func() {
				err := recover()
				if err == nil || !strings.Contains(err.(string), "conflicting closure environment ABI") {
					t.Fatalf("expected an ABI conflict, got %v", err)
				}
			}()
			ctx.compileFuncDecl(ctx.pkg, plain)
		}()
		if plain.declaration.HasExplicitEnv() {
			t.Fatal("ABI validation mutated source metadata")
		}
	}
}

func TestFunctionWrapperBindsGenericOriginAfterExportOnlyPreload(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "generic.go", `package p
//llgo:env
func private[T any](v T) {}
func Use() { private(1) }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	// Export-only preload can omit private functions and use a different
	// types.Package from the later full type check.
	if err = ParsePkgSyntax(prog, fs, types.NewPackage("p", "p"), files); err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	owner, err := (&types.Config{}).Check("p", fs, files, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fs, ssa.SanityCheckFunctions)
	goPkg := goProg.CreatePackage(owner, files, info, true)
	goPkg.Build()
	ctx := &context{prog: prog, fset: fs, goTyps: owner, goProg: goProg, goPkg: goPkg}
	origin := ctx.function(goPkg.Func("private"))
	for _, block := range goPkg.Func("Use").Blocks {
		for _, instruction := range block.Instrs {
			if call, ok := instruction.(*ssa.Call); ok {
				instance := ctx.function(call.Common().StaticCallee())
				if instance == origin || instance.declaration == nil || instance.declaration != origin.declaration || !instance.declaration.HasExplicitEnv() {
					t.Fatal("generic instance lost its source declaration")
				}
				return
			}
		}
	}
	t.Fatal("missing generic call")
}

func TestFunctionWrapperUsesPatchedImportDeclaration(t *testing.T) {
	fs := token.NewFileSet()
	parse := func(name, source string) *ast.File {
		file, err := parser.ParseFile(fs, name, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	originalFile := parse("original.go", "package p\nfunc F()\n")
	patchFile := parse("patch.go", "package p\n//go:linkname F llgo.unreachable\nfunc F()\n")
	original, err := (&types.Config{}).Check("p", fs, []*ast.File{originalFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := (&types.Config{}).Check("p", fs, []*ast.File{patchFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	if err = ParsePkgSyntax(prog, fs, original, []*ast.File{originalFile}); err != nil {
		t.Fatal(err)
	}
	if err = ParsePkgSyntax(prog, fs, patched, []*ast.File{originalFile, patchFile}); err != nil {
		t.Fatal(err)
	}
	BindPackageFunctionDeclarations(prog, original, patched, fs, []*ast.File{patchFile})
	originalDecl := prog.SourceFunctionDeclaration(original, fs, "p.F", originalFile.Decls[0].Pos())
	if _, ok := originalDecl.Linkname(); ok {
		t.Fatal("patch mutated the original declaration")
	}
	if originalDecl.Effective() == originalDecl {
		t.Fatal("missing explicit package replacement")
	}

	callerFile := parse("caller.go", "package caller\nimport \"p\"\nfunc Call() { p.F() }\n")
	info := newLocalityTypeInfo()
	caller, err := (&types.Config{Importer: importerFunc(func(string) (*types.Package, error) { return original, nil })}).Check("caller", fs, []*ast.File{callerFile}, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fs, ssa.SanityCheckFunctions)
	goProg.CreatePackage(original, nil, nil, true)
	goPkg := goProg.CreatePackage(caller, []*ast.File{callerFile}, info, true)
	goPkg.Build()
	backend := prog.NewBackendProgram()
	defer backend.Dispose()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, nil, nil, nil, goPkg, []*ast.File{callerFile}, nil, false, Options{PreloadedSyntax: true})
	if err != nil {
		t.Fatal(err)
	}
	body := compiled.Module().NamedFunction("caller.Call").String()
	if !strings.Contains(body, "unreachable") || strings.Contains(body, "@p.F") {
		t.Fatalf("import did not use the patched intrinsic:\n%s", body)
	}
}

func TestFunctionDeclarationKeepsPendingNoInterface(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "method.go", "package p\ntype T int\nfunc (T) M() {}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	defer prog.Dispose()
	prog.SetNoInterfaceMethod("p.T.M")
	pending := prog.NamedFunctionDeclaration("p.T.M")
	owner := types.NewPackage("p", "p")
	if err = ParsePkgSyntax(prog, fs, owner, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	decl := prog.SourceFunctionDeclaration(owner, fs, "p.T.M", file.Decls[1].Pos())
	if decl != pending || !decl.NoInterface() {
		t.Fatal("syntax preparation lost the pending nointerface directive")
	}
}

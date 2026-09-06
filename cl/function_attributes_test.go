package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestSourceFunctionAttributes(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package p
type T struct { x int }
//llgo:attribute param(p) returned
//llgo:attribute result(0) nonnull
func F(p *int) *int { return p }
//llgo:attribute receiver readonly captures(none)
//llgo:attribute memory(argmem: read)
func (p *T) Read() int { return p.x }
//llgo:attribute result(0) nonnegative
func Count() int { return 5 }
//llgo:attribute param(p) returned
//llgo:attribute result(0) nonnull
func Generic[T any](p *T) *T { return p }
func UseGeneric(p *int) *int { return Generic(p) }
`)
	defer prog.Dispose()
	for _, want := range []string{`define nonnull ptr @"example.com/locality.F"(ptr returned`, "ptr readonly captures(none)", "range(i64 0, -9223372036854775808)", "llgo.source.attributes"} {
		if !strings.Contains(ir, want) {
			t.Errorf("missing %q:\n%s", want, ir)
		}
	}
	if !strings.Contains(ir, `Generic[int]"(ptr returned`) {
		t.Fatalf("generic instance lost source attributes:\n%s", ir)
	}
}

func TestSourceAttributePlacementDiagnostics(t *testing.T) {
	for _, body := range []string{
		"//llgo:attribute cold\nvar x int",
		"type T struct {\n//llgo:attribute cold\n x int\n}",
		"type I interface {\n//llgo:attribute cold\n M()\n}",
		"func F() {\n//llgo:attribute cold\n f := func() {}; f()\n}",
	} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "placement.go", "package p\n"+body, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := (&types.Config{}).Check("p", fset, []*ast.File{f}, nil)
		if err != nil {
			t.Fatal(err)
		}
		p := llssa.NewProgram(nil)
		err = ParsePkgSyntax(p, fset, pkg, []*ast.File{f})
		p.Dispose()
		if err == nil || !strings.Contains(err.Error(), "requires a named function") {
			t.Fatalf("%s: %v", body, err)
		}
	}
}

func TestSourceAttributesPreloadedAcrossBackendsAndLinknames(t *testing.T) {
	fset := token.NewFileSet()
	src := `package owner
import "unsafe"
//go:linkname Copy shared_copy
//llgo:attribute param(p) returned
//llgo:attribute result(0) nonnull
func Copy(p unsafe.Pointer) unsafe.Pointer { return p }
`
	f, err := parser.ParseFile(fset, "owner.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("owner", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	coordinator := llssa.NewProgram(nil)
	defer coordinator.Dispose()
	if err = ParsePkgSyntax(coordinator, fset, pkg, []*ast.File{f}); err != nil {
		t.Fatal(err)
	}
	sig := pkg.Scope().Lookup("Copy").Type().(*types.Signature)
	for _, name := range []string{"caller", "owner"} {
		backend := coordinator.NewBackendProgram()
		p := backend.NewPackage(name, name)
		p.NewFunc("shared_copy", sig, llssa.InGo)
		if !strings.Contains(p.String(), "declare nonnull ptr @shared_copy(ptr returned") {
			t.Fatalf("%s lost contract:\n%s", name, p.String())
		}
		if err = llvm.VerifyModule(p.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		backend.Dispose()
	}
}

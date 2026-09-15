package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

func TestFunctionAttributes(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package p
type T struct{}
//llgo:cold
// llgo:noreturn
//llgo:cold
func Fatal() { for {} }
//llgo:cold
func (*T) Rare() {}
//llgo:noreturn
func Generic[T any](v *T) { for {} }
func Use(p *int) { Generic(p) }
//export Exported
//llgo:cold
func Exported() {}
func Ordinary() {}
`)
	defer prog.Dispose()
	for _, test := range []struct {
		symbol         string
		cold, noreturn bool
	}{
		{"example.com/locality.Fatal", true, true},
		{"example.com/locality.(*T).Rare", true, false},
		{"example.com/locality.Generic[int]", false, true},
		{"Exported", true, false},
		{"example.com/locality.Ordinary", false, false},
	} {
		header := regexp.MustCompile(`(?m)^define .*@"?` + regexp.QuoteMeta(test.symbol) + `"?\(.*$`).FindString(ir)
		if header == "" {
			t.Fatalf("missing %s:\n%s", test.symbol, ir)
		}
		attrs := ""
		if group := regexp.MustCompile(`#\d+`).FindString(header); group != "" {
			attrs = regexp.MustCompile(`(?m)^attributes ` + group + ` = .*`).FindString(ir)
		}
		if strings.Contains(attrs, "cold") != test.cold || strings.Contains(attrs, "noreturn") != test.noreturn {
			t.Errorf("%s: %s", test.symbol, attrs)
		}
	}
}

func TestFunctionAttributesWithExportOnlyPreload(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "private.go", `package owner
//llgo:cold
//llgo:noreturn
func private[T any](p *T) { for {} }
func Use(p *int) { private(p) }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	coordinator := newLLSSAProg(t)
	defer coordinator.Dispose()
	// Export data may omit private functions. The source declaration must
	// survive before a different type-checking pass creates their objects.
	exported := types.NewPackage("owner", "owner")
	if err = ParsePkgSyntax(coordinator, fs, exported, files); err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	owner, err := (&types.Config{}).Check("owner", fs, files, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fs, ssa.SanityCheckFunctions)
	goPkg := goProg.CreatePackage(owner, files, info, true)
	goPkg.Build()
	backend := coordinator.NewBackendProgram()
	defer backend.Dispose()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(backend, nil, nil, nil, goPkg, files, nil, false, Options{PreloadedSyntax: true})
	if err != nil {
		t.Fatal(err)
	}
	fn := compiled.Module().NamedFunction("owner.private[int]")
	if fn.IsNil() {
		t.Fatal("missing private generic instance")
	}
	for _, name := range []string{"cold", "noreturn"} {
		if fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(name)).IsNil() {
			t.Fatalf("private generic instance lost %s: %s", name, fn.String())
		}
	}
	if err = llvm.VerifyModule(compiled.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionAttributeDiagnostics(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"//llgo:cold\nvar x int", "named function"},
		{"//llgo:cold\n\nfunc F() {}", "named function"},
		{"func F() {} //llgo:cold", "named function"},
		{"type T struct {\n//llgo:noreturn\n x int\n}", "named function"},
		{"type T interface {\n//llgo:cold\n M()\n}", "named function"},
		{"func F() {\n//llgo:cold\n f := func() {}; f()\n}", "named function"},
		{"//llgo:cold noreturn\nfunc F() {}", "takes no arguments"},
		{"// llgo:noreturn()\nfunc F() {}", "takes no arguments"},
		{"//llgo:cold(x)\nfunc F() {}", "takes no arguments"},
		{"//llgo:param(p) nonnull\nfunc F(p *int) {}", "not yet supported"},
		{"//llgo:result nonnull\nfunc F() *int { return nil }", "not yet supported"},
		{"type T int\n//llgo:receiver noalias\nfunc (T) F() {}", "not yet supported"},
	} {
		fs := token.NewFileSet()
		file, err := parser.ParseFile(fs, "attributes.go", "package p\n"+test.source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := (&types.Config{}).Check("p", fs, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prog := llssa.NewProgram(nil)
		err = ParsePkgSyntax(prog, fs, pkg, []*ast.File{file})
		prog.Dispose()
		if err == nil || !strings.Contains(err.Error(), test.want) || !strings.Contains(err.Error(), "attributes.go:") {
			t.Errorf("%s: %v, want %s at source location", test.source, err, test.want)
		}
	}
}

func TestFunctionAttributesAcrossBackendsAndLinknames(t *testing.T) {
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, "owner.go", `package owner
import _ "unsafe"
//go:linkname Stop shared_stop
//llgo:cold
func Stop()
//go:linkname Fatal shared_stop
//llgo:noreturn
func Fatal() { for {} }
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := (&types.Config{Importer: importer.Default()}).Check("owner", fs, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	coordinator := llssa.NewProgram(nil)
	defer coordinator.Dispose()
	if err = ParsePkgSyntax(coordinator, fs, owner, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	sig := owner.Scope().Lookup("Stop").Type().(*types.Signature)
	for _, name := range []string{"owner.Stop", "owner.Fatal", "shared_stop"} {
		backend := coordinator.NewBackendProgram()
		pkg := backend.NewPackage("caller", "caller")
		pkg.NewFunc(name, sig, llssa.InGo, backend.SourceFunctionAttributes(name))
		fn := pkg.Module().FirstFunction()
		for _, attr := range []string{"cold", "noreturn"} {
			if fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID(attr)).IsNil() {
				t.Errorf("%s lost %s", name, attr)
			}
		}
		if err = llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		backend.Dispose()
	}
}

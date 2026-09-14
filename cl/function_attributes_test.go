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

func TestFunctionAttributeDiagnostics(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"//llgo:cold\nvar x int", "named function"},
		{"type T struct {\n//llgo:noreturn\n x int\n}", "named function"},
		{"type T interface {\n//llgo:cold\n M()\n}", "named function"},
		{"func F() {\n//llgo:cold\n f := func() {}; f()\n}", "named function"},
		{"//llgo:cold noreturn\nfunc F() {}", "takes no arguments"},
		{"// llgo:noreturn()\nfunc F() {}", "takes no arguments"},
		{"//llgo:cold(x)\nfunc F() {}", "takes no arguments"},

		{"type T int\n//llgo:receiver noalias\nfunc (T) F() {}", "unsupported attribute"},
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
		pkg.NewFunc(name, sig, llssa.InGo)
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

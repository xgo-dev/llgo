package funcattrs

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func parseTest(t *testing.T, source string) ([]Attribute, *types.Signature, error) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "attributes.go", "package p\nimport \"unsafe\"\nvar _ unsafe.Pointer\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	decl := f.Decls[len(f.Decls)-1].(*ast.FuncDecl)
	attrs, err := Parse(fset, decl)
	return attrs, pkg.Scope().Lookup(decl.Name.Name).Type().(*types.Signature), err
}

func TestSourceSelectorsAndValidation(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:attribute param(b) readonly captures(none)
// llgo:attribute param(0) returned
//llgo:attribute result(p) nonnull
func F(a, b unsafe.Pointer) (p unsafe.Pointer) { return a }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 4 {
		t.Fatalf("attributes = %v", attrs)
	}
	for _, a := range attrs {
		if a.Name == "readonly" && (a.Target.Scope != Parameter || a.Target.Index != 1) {
			t.Fatalf("grouped source parameter = %v", a.Target)
		}
		if a.Name == "nonnull" && (a.Target.Scope != Result || a.Target.Index != 0) {
			t.Fatalf("named result = %v", a.Target)
		}
	}
}

func TestSourceAttributeDiagnostics(t *testing.T) {
	for _, tc := range []struct{ directive, signature, want string }{
		{"hot", "func F() {}", "unsupported attribute"},
		{"nonnull", "func F() {}", "not supported on function"},
		{"param(missing) readonly", "func F(p *int) {}", "unknown source value"},
		{"param(1) readonly", "func F(p *int) {}", "unknown source value"},
		{"param(-1) readonly", "func F(p *int) {}", "unknown source value"},
		{"receiver readonly", "func F() {}", "requires a method"},
		{"param(0) readonly", "func F(p string) {}", "requires a pointer"},
		{"result(0) nonnull", "func F() (*int,bool) { return nil,false }", "single scalar result"},
		{"result(0) nonnegative", "func F() *int { return nil }", "integer result"},
		{"param(0) returned", "func F(p *int) int { return 0 }", "compatible scalar"},
		{"memory(argmem: write)", "func F() {}", "unsupported memory effects"},
		{"param(0) captures(address)", "func F(p *int) {}", "unsupported capture effects"},
		{"param(0) readonly writeonly", "func F(p *int) {}", "conflicting"},
		{"memory(read) memory(argmem: read)", "func F() {}", "conflicting"},
		{"result(0) range(128, 256)", "func F() int8 { return 0 }", "does not fit"},
		{"result(0) range(3, 3)", "func F() int { return 0 }", "nonempty"},
		{"result(0) range(0, n)", "func F() int { return 0 }", "integer literals"},
		{"memory(read", "func F() {}", "unbalanced"},
		{"cold()", "func F() {}", "empty arguments"},
		{"param(0)", "func F(p int) {}", "expected an attribute"},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			a, s, err := parseTest(t, "//llgo:attribute "+tc.directive+"\n"+tc.signature)
			if err == nil {
				err = Validate(a, s, 64, false)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attributes.go:4:") {
				t.Fatalf("error = %v, want %s with source position", err, tc.want)
			}
		})
	}
}

func TestTargetSizedRanges(t *testing.T) {
	for _, bits := range []int{32, 64} {
		a, s, err := parseTest(t, "//llgo:attribute result(0) nonnegative\nfunc F() int { return 0 }")
		if err != nil {
			t.Fatal(err)
		}
		n, v, full, err := IntegerRange(a[0], s.Results().At(0).Type(), bits)
		if err != nil || full || n != bits || v != [2]uint64{0, uint64(1) << uint(bits-1)} {
			t.Fatalf("%d: %d %v %v %v", bits, n, v, full, err)
		}
	}
	a, s, err := parseTest(t, "//llgo:attribute result(0) range(-128, 128)\nfunc F() int8 { return 0 }")
	if err != nil {
		t.Fatal(err)
	}
	_, _, full, err := IntegerRange(a[0], s.Results().At(0).Type(), 64)
	if err != nil || !full {
		t.Fatalf("full signed domain = %v, %v", full, err)
	}
}

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
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	_, err = (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	decl := f.Decls[len(f.Decls)-1].(*ast.FuncDecl)
	attrs, err := Parse(fset, decl)
	return attrs, info.Defs[decl.Name].Type().(*types.Signature), err
}

func TestSourceMergeOwnsCanonicalRange(t *testing.T) {
	attrs, _, err := parseTest(t, `//llgo:param(v) range(0x0,0x10) range(0,16)
func F(v int) {}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 1 || attrs[0].Range.Lower.Sign() != 0 || attrs[0].Range.Upper.Int64() != 16 {
		t.Fatalf("range = %+v", attrs)
	}
	merged, err := Merge(attrs)
	if err != nil {
		t.Fatal(err)
	}
	attrs[0].Range.Upper.SetInt64(8)
	if merged[0].Range.Upper.Int64() != 16 {
		t.Fatal("Merge retained mutable source operands")
	}
}

func TestSourceWideRanges(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:param(v) range(0,18446744073709551616)
//llgo:result(0) nonnegative
func F(v uint64) uint64 { return v }`)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range attrs {
		n, _, full, err := IntegerRange(a, types.Typ[types.Uint64], 64)
		if err != nil || !full || n != 64 {
			t.Fatalf("%+v: %d %v %v", a, n, full, err)
		}
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}

}

func TestTargetSizedRanges(t *testing.T) {
	for _, bits := range []int{32, 64} {
		a, s, err := parseTest(t, "//llgo:result(0) nonnegative\nfunc F() int { return 0 }")
		if err != nil {
			t.Fatal(err)
		}
		n, v, full, err := IntegerRange(a[0], s.Results().At(0).Type(), bits)
		if err != nil || full || n != bits || v != [2]uint64{0, uint64(1) << uint(bits-1)} {
			t.Fatalf("%d: %d %v %v %v", bits, n, v, full, err)
		}
	}
	a, s, err := parseTest(t, "//llgo:result(0) range(-128, 128)\nfunc F() int8 { return 0 }")
	if err != nil {
		t.Fatal(err)
	}
	_, _, full, err := IntegerRange(a[0], s.Results().At(0).Type(), 64)
	if err != nil || !full {
		t.Fatalf("full signed domain = %v, %v", full, err)
	}
}

func TestSourceValueAttributes(t *testing.T) {
	for _, source := range []string{
		"//llgo:param(b) nonnull\n// llgo:result nonnull\nfunc F(a,b *int) *int { return b }",
		"//llgo:param( 0 ) nonnegative range(0,64)\n//llgo:result(0) range(0,64)\nfunc F(_ uint32) uint32 { return 0 }",
		"type T int\n//llgo:receiver nonnull\nfunc (*T) F() {}",
		"//llgo:result nonnull\n//llgo:result(out) nonnull\nfunc F() (out *int) { return nil }",
		"//llgo:result nonnegative\nfunc F() (_ uint32) { return 0 }",
	} {
		attrs, sig, err := parseTest(t, source)
		if err == nil {
			err = Validate(attrs, sig, 64, false)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceValueDiagnostics(t *testing.T) {
	for _, tc := range []struct{ directive, signature, want string }{
		{"param(missing) nonnull", "func F(p *int) {}", "unknown source value"},
		{"param(-1) nonnull", "func F(p *int) {}", "unknown source value"},
		{"param(_) nonnull", "func F(_ *int) {}", "unknown source value"},
		{"receiver nonnull", "func F() {}", "requires a method"},
		{"param(0) nonnull", "func F(p string) {}", "requires a pointer"},
		{"result", "func F() {}", "exactly one source result"},
		{"result", "func F() *int { return nil }", "expected an attribute"},
		{"param(0) nonnull)", "func F(p *int) {}", "unbalanced"},
		{"param(0) nonnull(", "func F(p *int) {}", "unbalanced"},
		{"param(0) (nonnull)", "func F(p *int) {}", "invalid attribute expression"},
		{"param(0) nonnull()", "func F(p *int) {}", "empty arguments"},
		{"param(0) nonnull(x)", "func F(p *int) {}", "invalid arguments"},
		{"result range(1)", "func F() int { return 0 }", "two integer literals"},
		{"result range(128,256)", "func F() int8 { return 0 }", "does not fit"},
		{"result range(3,3)", "func F() int { return 0 }", "nonempty"},
		{"result range(0,n)", "func F() int { return 0 }", "integer literals"},
		{"param(0).field(p) nonnull", "func F(p *int) {}", "unsupported selector suffix"},
		{"result nonnegative", "func F() *int { return nil }", "integer value"},
		{"result range(-10,0) nonnegative", "func F() int { return 0 }", "conflicting"},
		{"result range(0,5) range(0,6)", "func F() int { return 0 }", "conflicting"},
	} {
		attrs, sig, err := parseTest(t, "//llgo:"+tc.directive+"\n"+tc.signature)
		if err == nil {
			err = Validate(attrs, sig, 64, false)
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attributes.go:4:") {
			t.Errorf("%s: %v, want %s", tc.directive, err, tc.want)
		}
	}
}

func TestGenericValueAttributeValidation(t *testing.T) {
	attrs, sig, err := parseTest(t, "//llgo:param(v) nonnegative\nfunc F[T any](v T) {}")
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, true); err != nil {
		t.Fatal(err)
	}
	for _, typ := range []types.Type{types.Typ[types.Int], types.Typ[types.String]} {
		instance, err := types.Instantiate(nil, sig, []types.Type{typ}, true)
		if err != nil {
			t.Fatal(err)
		}
		err = Validate(attrs, instance.(*types.Signature), 64, false)
		if (err == nil) != (typ == types.Typ[types.Int]) {
			t.Fatalf("%s: %v", typ, err)
		}
	}
}

func TestSourceResultRelations(t *testing.T) {
	for _, tc := range []struct{ selector, signature, want string }{
		{"result sameas(p)", "func F(p *int) *int { return p }", ""},
		{"result(out) sameas(p) nonnull", "func F(p *int) (out *int,n int) { return p,0 }", ""},
		{"result sameas(p)", "func F(p int32) uint32 { return 0 }", "identical integer"},
		{"result sameas(p)", "func F(p bool) bool { return p }", "integer or pointer"},
		{"result sameas(0)", "func F(p int) int { return p }", "parameter name"},
		{"result sameas(missing)", "func F(p int) int { return p }", "unknown source value"},
		{"param(p) sameas(p)", "func F(p int) int { return p }", "not supported on parameter"},
	} {
		attrs, sig, err := parseTest(t, "//llgo:"+tc.selector+"\n"+tc.signature)
		if err == nil {
			err = Validate(attrs, sig, 64, false)
		}
		if tc.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.selector, err)
		}
	}
}

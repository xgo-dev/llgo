package funcattrs

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
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

func TestSourceSelectorsAndValidation(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:param(b) access(read) noalias
// llgo:result sameas(a)
//llgo:result(p) nonnull
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
		if a.Name == "access" && (a.Target.Scope != Parameter || a.Target.Index != 1 || a.Access != AccessRead) {
			t.Fatalf("grouped source parameter = %v", a.Target)
		}
		if a.Name == "nonnull" && (a.Target.Scope != Result || a.Target.Index != 0) {
			t.Fatalf("named result = %v", a.Target)
		}
	}
}

func TestSourceAttributeDiagnostics(t *testing.T) {
	for _, tc := range []struct{ directive, signature, want string }{
		{"cold noreturn", "func F() {}", "own //llgo: line"},
		{"cold(extra)", "func F() {}", "invalid arguments"},
		{"param(p) cold", "func F(p *int) {}", "not supported on parameter"},
		{"param(p) hot", "func F(p *int) {}", "unsupported attribute"},
		{"param(missing) nonnull", "func F(p *int) {}", "unknown source value"},
		{"param(1) nonnull", "func F(p *int) {}", "unknown source value"},
		{"param(-1) nonnull", "func F(p *int) {}", "unknown source value"},
		{"param(_) nonnull", "func F(_ *int) {}", "unknown source value"},
		{"receiver nonnull", "func F() {}", "requires a method"},
		{"param(0) access(read)", "func F(p string) {}", "requires a pointer"},
		{"result(0) nonnegative", "func F() *int { return nil }", "integer value"},
		{"result", "func F() {}", "exactly one source result"},
		{"result nonnull", "func F() (*int,*int) { return nil,nil }", "exactly one source result"},
		{"result", "func F() *int { return nil }", "expected an attribute"},
		{"param(0)", "func F(p int) {}", "expected an attribute"},
		{"param(0) nonnull)", "func F(p *int) {}", "unbalanced"},
		{"param(0) nonnull(", "func F(p *int) {}", "unbalanced"},
		{"param(0) (nonnull)", "func F(p *int) {}", "invalid attribute expression"},
		{"param(0) nonnull()", "func F(p *int) {}", "empty arguments"},
		{"param(0) noalias(x)", "func F(p *int) {}", "invalid arguments"},
		{"param(0) access", "func F(p *int) {}", "invalid arguments"},
		{"param(0) access(invalid)", "func F(p *int) {}", "unknown access mode"},
		{"param(0) access(read) access(write)", "func F(p *int) {}", "conflicting"},
		{"result(0) access(read)", "func F() *int { return nil }", "not supported on result"},
		{"result(0) range(1)", "func F() int { return 0 }", "two integer literals"},
		{"result(0) range(128,256)", "func F() int8 { return 0 }", "does not fit"},
		{"result(0) range(3,3)", "func F() int { return 0 }", "nonempty"},
		{"result(0) range(0,n)", "func F() int { return 0 }", "integer literals"},
		{"param(0).field(p) nonnull", "func F(p *int) {}", "unsupported selector suffix"},
		{"param(0).element(0) nonnull", "func F(p [2]*int) {}", "unsupported selector suffix"},
		{"param(0) sameas(s)", "func F(s int) int { return s }", "not supported on parameter"},
		{"result(0) sameas(0)", "func F(s int) int { return s }", "parameter name"},
		{"result(0) sameas(_)", "func F(_ int) int { return 0 }", "parameter name"},
		{"result(0) sameas(param(s))", "func F(s int) int { return s }", "parameter name"},
		{"result(0) sameas(missing)", "func F(s int) int { return s }", "unknown source value"},
		{"result(0) sameas(s)", "func F(s int32) uint32 { return 0 }", "identical integer source types"},
		{"result(0) sameas(s)", "func F(s bool) bool { return s }", "integer or pointer result"},
		{"result(0) range(-10,0) nonnegative", "func F() int { return 0 }", "conflicting range and nonnegative"},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			attrs, sig, err := parseTest(t, "//llgo:"+tc.directive+"\n"+tc.signature)
			if err == nil {
				err = Validate(attrs, sig, 64, false)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attributes.go:4:") {
				t.Fatalf("diagnostic = %v; want %q at source annotation", err, tc.want)
			}
		})
	}
}

func TestSourceAttributesRejectRemovedScope(t *testing.T) {
	for _, text := range []string{"memory(none)", "capture(none)", "nofree", "nosync", "nounwind", "willreturn", "param(p) capture(none)", "param(p) readonly", "param(p) returned", "param(p) align(16)", "result same_as(param(p))", "attr result(0) nonnull"} {
		_, _, err := parseTest(t, "//llgo:"+text+"\nfunc F(p *int) *int { return p }")
		if err == nil {
			t.Errorf("accepted removed attribute %s", text)
		}
	}
}

func TestSourceMultipleResults(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:param(p) nonnull access(read)
//llgo:param(n) range(-4,20) nonnegative
//llgo:result(out) nonnull sameas(p)
//llgo:result(count) range(0,32)
func F(p *int, n int32) (out *int, count int32) { return p,n }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 7 {
		t.Fatalf("attributes = %+v", attrs)
	}
	for _, a := range attrs {
		if a.Name == "sameas" && (a.Target.String() != "result(0)" || a.From.String() != "param(0)") {
			t.Fatalf("sameas = %+v", a)
		}
	}
}

func TestSourceReceiverAndUnnamedValues(t *testing.T) {
	attrs, sig, err := parseTest(t, `type Box struct{ P *int }
//llgo:receiver nonnull
//llgo:param(0) range(-4,4)
//llgo:result(1) nonnull
func (*Box) F(int) (bool,*Box) { return false,nil }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	for _, a := range attrs {
		if a.Target.Scope == Receiver {
			typ, err := ResolveTarget(sig, a.Target)
			if err != nil || !pointer(typ) {
				t.Fatalf("receiver %v: %v", typ, err)
			}
		}
	}
}

func TestSourceRepeatedAttributesAndResultShorthand(t *testing.T) {
	for _, signature := range []string{"func F(p unsafe.Pointer) *int8 { return (*int8)(p) }", "func F(p unsafe.Pointer) (out *int8) { return (*int8)(p) }", "func F(p unsafe.Pointer) (_ *int8) { return (*int8)(p) }"} {
		attrs, sig, err := parseTest(t, "// llgo:param(p) access(read) access(read) noalias\n//llgo:result sameas(p)\n//llgo:result(0) sameas(p)\n"+signature)
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(attrs, sig, 64, false); err != nil {
			t.Fatal(err)
		}
		if len(attrs) != 3 {
			t.Fatalf("duplicate annotations were not merged: %+v", attrs)
		}
	}
}

func TestSourcePublicContractModes(t *testing.T) {
	for _, mode := range []string{"none", "read", "write", "readwrite"} {
		attrs, sig, err := parseTest(t, "//llgo:param(0) access("+mode+")\nfunc F(p *int) {}")
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(attrs, sig, 64, false); err != nil {
			t.Fatal(err)
		}
	}
	attrs, sig, err := parseTest(t, "//llgo:cold\n// llgo:noreturn\nfunc F() { panic(0) }")
	if err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 2 {
		t.Fatalf("missing independent function attributes: %+v", attrs)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
}

func TestSourceGenericValueAndIntegerIdentity(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:param(v) range(0,128)
//llgo:result(0) sameas(v)
func F[T ~int8 | ~int16](v T) T { return v }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, true); err != nil {
		t.Fatal(err)
	}
	for _, typ := range []types.Type{types.Typ[types.Int8], types.Typ[types.Int16]} {
		instance, err := types.Instantiate(nil, sig, []types.Type{typ}, true)
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(attrs, instance.(*types.Signature), 64, false); err != nil {
			t.Fatal(err)
		}
	}
	attrs, sig, err = parseTest(t, `type Counter int
//llgo:result(0) sameas(v)
func F(v Counter) int { return int(v) }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err == nil {
		t.Fatal("distinct named integer types accepted")
	}
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

func TestSourceTypedOperandsJSONRoundTrip(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:param(p) nonnull access(read)
//llgo:param(n) range(0,18446744073709551616)
//llgo:result(1) sameas(p)
func F(p *int, n uint64) (bool,*int) { return false,p }`)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []Attribute
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(attrs, decoded) {
		t.Fatalf("typed operands changed after JSON round trip: %s", data)
	}
	if err = Validate(decoded, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	// Backend validation must consume the typed operands, including bounds
	// above uint64's maximum, independently of their original source spelling.
	for i := range decoded {
		decoded[i].Args = "invalid parser input"
	}
	if err = Validate(decoded, sig, 64, false); err != nil {
		t.Fatalf("validation reparsed source spelling: %v", err)
	}
	for _, a := range decoded {
		if a.Name == "range" {
			bits, _, full, err := IntegerRange(a, types.Typ[types.Uint64], 64)
			if err != nil || bits != 64 || !full {
				t.Fatalf("uint64 bounds lost: bits=%d full=%v err=%v", bits, full, err)
			}
		}
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

func TestSourceSelectorWhitespaceAndBlankInput(t *testing.T) {
	attrs, sig, err := parseTest(t, `// llgo:param( 0 ) nonnull
//llgo:param( value ) range( 0, 64 )
// llgo:result sameas( value )
func F(_ *int, value uint32) uint32 { return value }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 3 {
		t.Fatalf("attributes = %+v", attrs)
	}
	for _, a := range attrs {
		if a.Name == "sameas" && (a.From.Scope != Parameter || a.From.Index != 1) {
			t.Fatalf("wrong entry parameter: %+v", a)
		}
	}
}

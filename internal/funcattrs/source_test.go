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
	attrs, sig, err := parseTest(t, `//llgo:attr param(b) readonly captures(none)
// llgo:attr param(0) returned
//llgo:attr result(p) nonnull
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
		{"", "func F() {}", "expected an attribute"},
		{"memory(read))", "func F() {}", "unbalanced parentheses"},
		{"(cold)", "func F() {}", "invalid attribute expression"},
		{"cold(extra)", "func F() {}", "invalid arguments"},
		{"param(0) readonly(extra)", "func F(p *int) {}", "invalid arguments"},
		{"param(0) access", "func F(p *int) {}", "invalid arguments"},
		{"result(0) range(1)", "func F() int { return 0 }", "two integer literals"},
		{"result(0) same_as(missing)", "func F(p *int) *int { return p }", "same_as"},
		{"result(0) same_as(other(0))", "func F(p *int) *int { return p }", "same_as"},
		{"result(0) returned", "func F(p *int) *int { return p }", "returned requires an input"},
		{"hot", "func F() {}", "unsupported attribute"},
		{"nonnull", "func F() {}", "not supported on function"},
		{"param(missing) readonly", "func F(p *int) {}", "unknown source value"},
		{"param(1) readonly", "func F(p *int) {}", "unknown source value"},
		{"param(-1) readonly", "func F(p *int) {}", "unknown source value"},
		{"receiver readonly", "func F() {}", "requires a method"},
		{"param(0) readonly", "func F(p string) {}", "requires a pointer"},
		{"result(0) nonnegative", "func F() *int { return nil }", "integer value"},
		{"param(0) returned", "func F(p *int) int { return 0 }", "compatible pointer types"},
		{"memory(inaccessible: write)", "func F() {}", "unknown memory location"},
		{"param(0) captures(address)", "func F(p *int) {}", "unsupported capture effects"},
		{"param(0) readonly writeonly", "func F(p *int) {}", "conflicting"},
		{"memory(read) memory(argmem: read)", "func F() {}", "conflicting"},
		{"result(0) range(128, 256)", "func F() int8 { return 0 }", "does not fit"},
		{"result(0) range(3, 3)", "func F() int { return 0 }", "nonempty"},
		{"result(0) range(0, n)", "func F() int { return 0 }", "integer literals"},
		{"memory(read", "func F() {}", "unbalanced"},
		{"cold()", "func F() {}", "empty arguments"},
		{"param(0)", "func F(p int) {}", "expected an attribute"},
		{"noexternal", "func F() {}", "unsupported attribute"},
		{"param(0) align(0)", "func F(p *int) {}", "unsupported attribute"},
		{"param(0) align(3)", "func F(p *int) {}", "unsupported attribute"},
		{"param(0) align(1 6)", "func F(p *int) {}", "unsupported attribute"},
		{"param(0) range(1 2, 20)", "func F(p int) {}", "integer literals"},
		{"param(0) access(invalid)", "func F(p *int) {}", "unknown access mode"},
		{"result(0) access(read)", "func F() *int { return nil }", "not supported on result"},
		{"result(0) capture(none)", "func F() *int { return nil }", "not supported on result"},
		{"param(0).field(p) nonnull", "func F(s *struct{p *int}) {}", "unsupported selector suffix"},
		{"param(0).field(missing) nonnull", "func F(s struct{p *int}) {}", "unsupported selector suffix"},
		{"param(0).field(_) nonnull", "func F(s struct{_ *int}) {}", "unsupported selector suffix"},
		{"param(0).element(2) nonnull", "func F(s [2]*int) {}", "unsupported selector suffix"},
		{"param(0).element(-1) nonnull", "func F(s [2]*int) {}", "unsupported selector suffix"},
		{"param(0).element(i) nonnull", "func F(s [2]*int) {}", "unsupported selector suffix"},
		{"param(0).element(0) nonnull", "func F(s []*int) {}", "unsupported selector suffix"},
		{"param(0).deref(p) nonnull", "func F(s *int) {}", "unsupported selector suffix"},
		{"param(0) same_as(param(0))", "func F(s int) int { return s }", "not supported on parameter"},
		{"result(0) same_as(result(0))", "func F(s int) int { return s }", "requires an input"},
		{"result(0) same_as(param(0))", "func F(s int32) uint32 { return 0 }", "identical integer source types"},
		{"result(0) same_as(param(0))", "func F(s bool) bool { return s }", "integer or pointer result"},
		{"param(0) returned", "func F(s int) (int,int) { return s,s }", "exactly one source result"},
		{"result(0) range(-10,0) nonnegative", "func F() int { return 0 }", "conflicting range and nonnegative"},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			a, s, err := parseTest(t, "//llgo:attr "+tc.directive+"\n"+tc.signature)
			if err == nil {
				err = Validate(a, s, 64, false)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attributes.go:4:") {
				t.Fatalf("error = %v, want %s with source position", err, tc.want)
			}
		})
	}
}

func TestSourceAttributesExcludeCompilerManagedProperties(t *testing.T) {
	for _, attribute := range []string{"param(p) align(16)", "nofree", "nosync", "nounwind", "willreturn", "param(p).field(P) nonnull", "param(p).element(0) nonnull"} {
		t.Run(attribute, func(t *testing.T) {
			_, _, err := parseTest(t, "//llgo:attr "+attribute+"\nfunc F(p *int) {}")
			if err == nil || !strings.Contains(err.Error(), "unsupported") {
				t.Fatalf("source annotation accepted: %s, error = %v", attribute, err)
			}
		})
	}
}

func TestSourceMultipleResults(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:attr param(p) nonnull access(read) capture(results)
//llgo:attr param(n) range(-4,20) nonnegative
//llgo:attr result(out) nonnull same_as(param(p))
//llgo:attr result(count) range(0,32)
func F(p *int, n int32) (out *int, count int32) { return p,n }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 8 {
		t.Fatalf("attributes = %+v", attrs)
	}
	for _, a := range attrs {
		if a.Name == "same_as" && (a.Target.String() != "result(0)" || a.From.String() != "param(0)") {
			t.Fatalf("same_as = %+v", a)
		}
	}
}

func TestSourceReceiverAndUnnamedValues(t *testing.T) {
	attrs, sig, err := parseTest(t, `type Box struct{ P *int }
//llgo:attr receiver nonnull
//llgo:attr param(0) range(-4,4)
//llgo:attr result(1) same_as(receiver)
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

func TestSourceCanonicalAliases(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:attr memory(read, argmem:readwrite) memory(args:readwrite,other:read)
//llgo:attr param(p) readonly access(read) captures(ret: address, provenance) capture(results) returned
//llgo:attr result(0) same_as(param(0))
func F(p unsafe.Pointer) *int8 { return (*int8)(p) }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
	if len(attrs) != 4 {
		t.Fatalf("aliases did not deduplicate: %+v", attrs)
	}
	for _, a := range attrs {
		switch a.Name {
		case "memory":
			if a.Memory != (MemoryEffects{Args: AccessReadWrite, Other: AccessRead}) {
				t.Fatalf("effects = %+v", a.Memory)
			}
		case "access":
			if a.Access != AccessRead {
				t.Fatalf("access = %v", a.Access)
			}
		case "capture":
			if a.Capture != CaptureResults {
				t.Fatalf("capture = %v", a.Capture)
			}
		case "same_as":
			if a.From == nil || a.From.Index != 0 || a.Target.Scope != Result {
				t.Fatalf("same_as = %+v", a)
			}
		default:
			t.Fatalf("noncanonical attribute %s", a.Name)
		}
	}
}

func TestSourcePublicContractModes(t *testing.T) {
	for _, mode := range []string{"none", "read", "write", "readwrite"} {
		attrs, sig, err := parseTest(t, "//llgo:attr memory("+mode+")\n//llgo:attr param(0) access("+mode+")\nfunc F(p *int) {}")
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(attrs, sig, 64, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"none", "results", "any"} {
		attrs, sig, err := parseTest(t, "//llgo:attr param(0) capture("+mode+")\nfunc F(p *int) {}")
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(attrs, sig, 64, false); err != nil {
			t.Fatal(err)
		}
	}
	attrs, sig, err := parseTest(t, "//llgo:attr cold noreturn\nfunc F() { panic(0) }")
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err != nil {
		t.Fatal(err)
	}
}

func TestSourceGenericValueAndIntegerIdentity(t *testing.T) {
	attrs, sig, err := parseTest(t, `//llgo:attr param(v) range(0,128)
//llgo:attr result(0) same_as(param(v))
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
//llgo:attr result(0) same_as(param(0))
func F(v Counter) int { return int(v) }`)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(attrs, sig, 64, false); err == nil {
		t.Fatal("distinct named integer types accepted")
	}
}

func TestSourceMergeOwnsCanonicalRange(t *testing.T) {
	attrs, _, err := parseTest(t, `//llgo:attr param(v) range(0x0,0x10) range(0,16)
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
	attrs, sig, err := parseTest(t, `//llgo:attr param(v) range(0,18446744073709551616)
//llgo:attr result(0) nonnegative
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
	attrs, sig, err := parseTest(t, `//llgo:attr memory(read,argmem:readwrite)
//llgo:attr param(p) nonnull access(read) capture(results)
//llgo:attr param(n) range(0,18446744073709551616)
//llgo:attr result(1) same_as(param(p))
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
		a, s, err := parseTest(t, "//llgo:attr result(0) nonnegative\nfunc F() int { return 0 }")
		if err != nil {
			t.Fatal(err)
		}
		n, v, full, err := IntegerRange(a[0], s.Results().At(0).Type(), bits)
		if err != nil || full || n != bits || v != [2]uint64{0, uint64(1) << uint(bits-1)} {
			t.Fatalf("%d: %d %v %v %v", bits, n, v, full, err)
		}
	}
	a, s, err := parseTest(t, "//llgo:attr result(0) range(-128, 128)\nfunc F() int8 { return 0 }")
	if err != nil {
		t.Fatal(err)
	}
	_, _, full, err := IntegerRange(a[0], s.Results().At(0).Type(), 64)
	if err != nil || !full {
		t.Fatalf("full signed domain = %v, %v", full, err)
	}
}

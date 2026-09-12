package funcattrs

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Source contracts must fail explicitly if a lowering supplies a mismatched
// logical signature, rather than annotating unrelated bits or transfer storage.
func TestValueContractRejectsMismatchedLogicalSignature(t *testing.T) {
	for _, tc := range []struct {
		name, ir, source, want string
	}{
		{"missing input", "declare void @F()", "//llgo:attr param(p) nonnull\nfunc F(p *int) {}", "source parameter has no logical LLVM value"},
		{"pointer as integer", "declare void @F(i64)", "//llgo:attr param(p) nonnull\nfunc F(p *int) {}", "logical LLVM pointer"},
		{"integer width", "declare i64 @F()", "//llgo:attr result(0) range(0,8)\nfunc F() int32 { return 0 }", "source-width LLVM integer"},
		{"forwarding representation", "declare ptr @F(i64)", "//llgo:attr result(0) same_as(param(p))\nfunc F(p *int) *int { return p }", "same_as values do not share"},
		{"missing tuple slot", "declare {ptr} @F()", "//llgo:attr result(1) nonnull\nfunc F() (int, *int) { return 0,nil }", "source field has no LLVM representation"},
		{"scalar as tuple", "declare ptr @F()", "//llgo:attr result(1) nonnull\nfunc F() (int, *int) { return 0,nil }", "nonaggregate LLVM value"},
		{"missing array slot", "declare [1 x ptr] @F()", "//llgo:attr result(1) nonnull\nfunc F() (*int, *int) { return nil,nil }", "source element has no LLVM representation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := valueTestModule(t, tc.ir)
			attrs, sig, err := parseTest(t, tc.source)
			if err != nil {
				t.Fatal(err)
			}
			err = Apply(mod.Context(), mod.NamedFunction("F"), sig, attrs, 0, 64)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "attributes.go:4:") {
				t.Fatalf("diagnostic = %v; want source-located %q", err, tc.want)
			}
		})
	}
}

func TestWholeResultsInArrayRepresentation(t *testing.T) {
	mod := valueTestModule(t, `
declare [2 x ptr] @F(ptr)
define i1 @caller(ptr %p) {
  %r = call [2 x ptr] @F(ptr %p)
  %first = extractvalue [2 x ptr] %r, 0
  %second = extractvalue [2 x ptr] %r, 1
  %same = icmp eq ptr %first, %p
  %valid = icmp ne ptr %second, null
  %ok = and i1 %same, %valid
  ret i1 %ok
}
`)
	attachValueTestContracts(t, mod, "F", `//llgo:attr result(0) same_as(param(p))
//llgo:attr result(1) nonnull
func F(p *int) (*int, *int) { return p,p }`)
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	optimizeValueTest(t, mod)
	if ir := mod.NamedFunction("caller").String(); !strings.Contains(ir, "ret i1 true") || !strings.Contains(ir, "@F(") {
		t.Fatal(ir)
	}
}

func TestUnsignedNonnegativeAddsNoRestriction(t *testing.T) {
	mod := valueTestModule(t, `
define i32 @F(i32 %n) { ret i32 %n }
define i32 @caller(i32 %n) { %r = call i32 @F(i32 %n) ret i32 %r }
`)
	attachValueTestContracts(t, mod, "F", "//llgo:attr param(n) nonnegative\nfunc F(n uint32) uint32 { return n }")
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mod.String(), "range(") || strings.Contains(mod.String(), "llvm.assume") {
		t.Fatal(mod.String())
	}
	optimizeValueTest(t, mod)
}

func TestValueContractRejectsIncompatibleCallPrototype(t *testing.T) {
	mod := valueTestModule(t, `
declare ptr @F(ptr)
define i64 @caller(ptr %p) {
  %r = call i64 @F(ptr %p)
  ret i64 %r
}


`)
	attachValueTestContracts(t, mod, "F", "//llgo:attr result(0) nonnull\nfunc F(p *int) *int { return p }")
	err := MaterializeValueContracts(mod)
	if err == nil || !strings.Contains(err.Error(), "incompatible logical prototype") {
		t.Fatalf("incompatible imported call accepted: %v", err)
	}
}

func TestValueContractRejectsInvalidBitcodePlan(t *testing.T) {
	for _, tc := range []struct{ plan, want string }{
		{"{", "invalid value contract plan"},
		{`{"Version":2}`, "unsupported value contract plan version"},
	} {
		mod := valueTestModule(t, "declare ptr @F(ptr)")
		mod.NamedFunction("F").AddFunctionAttr(mod.Context().CreateStringAttribute(valuePlanMetadata, tc.plan))
		if err := MaterializeValueContracts(mod); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("invalid bitcode plan diagnostic = %v", err)
		}
	}
}

func TestMethodContractsKeepReceiverAndParameterWithEnvironment(t *testing.T) {
	mod := valueTestModule(t, `
define ptr @F(ptr %env, ptr %recv, ptr %p) { ret ptr %recv }
`)
	attrs, sig, err := parseTest(t, `type T int
//llgo:attr receiver nonnull noalias
//llgo:attr param(p) nonnull access(none) capture(none)
//llgo:attr result(0) same_as(receiver)
func (r *T) F(p *int) *T { return r }
`)
	if err != nil {
		t.Fatal(err)
	}
	fn := mod.NamedFunction("F")
	if err = Apply(mod.Context(), fn, sig, attrs, 1, 64); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		index int
		name  string
	}{{2, "nonnull"}, {2, "noalias"}, {2, "returned"}, {3, "nonnull"}, {3, "readnone"}, {3, "captures"}} {
		if fn.GetEnumAttributeAtIndex(tc.index, llvm.AttributeKindID(tc.name)).IsNil() {
			t.Fatalf("lost %s on %d: %s", tc.name, tc.index, fn.String())
		}
		if !fn.GetEnumAttributeAtIndex(1, llvm.AttributeKindID(tc.name)).IsNil() {
			t.Fatalf("%s leaked onto environment", tc.name)
		}
	}
	if err = MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	if strings.Count(fn.String(), "call void @llvm.assume") != 2 {
		t.Fatal(fn.String())
	}
	optimizeValueTest(t, mod)
}

// Imported contracts must be revalidated against the declaration being emitted;
// stale selectors cannot silently migrate to another parameter or result.
func TestContractsRejectChangedImportedSignature(t *testing.T) {
	for _, tc := range []struct{ source, replacement, ir, want string }{
		{"//llgo:attr result(1) nonnull\nfunc F() (*int,*int) { return nil,nil }", "func F() *int { return nil }", "declare ptr @F()", "selector result(1) does not exist"},
		{"//llgo:attr result(0) same_as(param(q))\nfunc F(p,q *int) *int { return q }", "func F(p *int) *int { return p }", "declare ptr @F(ptr)", "same_as input: selector param(1) does not exist"},
		{"type T int\n//llgo:attr receiver nonnull\nfunc (p *T) F() {}", "func F() {}", "declare void @F()", "selector receiver does not exist"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			attrs, _, err := parseTest(t, tc.source)
			if err != nil {
				t.Fatal(err)
			}
			_, sig, err := parseTest(t, tc.replacement)
			if err != nil {
				t.Fatal(err)
			}
			mod := valueTestModule(t, tc.ir)
			err = Apply(mod.Context(), mod.NamedFunction("F"), sig, attrs, 0, 64)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("changed declaration diagnostic = %v", err)
			}
		})
	}
}

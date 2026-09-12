package funcattrs

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestNoAliasOptimizesModifiedMemoryOnly(t *testing.T) {
	for _, annotated := range []bool{false, true} {
		mod := valueTestModule(t, `
define i32 @rewrite(ptr %p, ptr %q) {
 store i32 1, ptr %p
 store i32 2, ptr %q
 %r = load i32, ptr %p
 ret i32 %r
}
define i32 @read_alias(ptr %p, ptr %q) {
 %a = load i32, ptr %p
 %b = load i32, ptr %q
 %r = add i32 %a, %b
 ret i32 %r
}
define i32 @read_same() {
 %p = alloca i32
 store i32 9, ptr %p
 %r = call i32 @read_alias(ptr %p, ptr %p)
 ret i32 %r
}
`)
		if annotated {
			attrs, sig, err := parseTest(t, "//llgo:attr param(p) noalias\nfunc F(p, q *int32) int32 { return 0 }")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"rewrite", "read_alias"} {
				if err := Apply(mod.Context(), mod.NamedFunction(name), sig, attrs, 0, 64); err != nil {
					t.Fatal(err)
				}
			}
		}
		optimizeValueTest(t, mod)
		folded := strings.Contains(mod.NamedFunction("rewrite").String(), "ret i32 1")
		if folded != annotated {
			t.Fatalf("noalias=%v: %s", annotated, mod.String())
		}
		if !strings.Contains(mod.NamedFunction("read_same").String(), "ret i32 18") {
			t.Fatal(mod.String())
		}
	}
}

func TestNoAliasRemapsParameterNotTransport(t *testing.T) {
	mod := valueTestModule(t, "declare void @old(ptr)\ndeclare void @direct(ptr, ptr)\ndeclare void @indirect(ptr)\n")
	attrs, sig, err := parseTest(t, "//llgo:attr param(p) noalias\nfunc F(p *int32) {}")
	if err != nil {
		t.Fatal(err)
	}
	old := mod.NamedFunction("old")
	if err = Apply(mod.Context(), old, sig, attrs, 0, 64); err != nil {
		t.Fatal(err)
	}
	direct := mod.NamedFunction("direct")
	RemapFunctionEffects(old, direct, ABIMapping{Result: ABIValue{Kind: Indirect, Indices: []int{1}}, Params: []ABIValue{DirectValue(2)}})
	kind := llvm.AttributeKindID("noalias")
	if direct.GetEnumAttributeAtIndex(2, kind).IsNil() || !direct.GetEnumAttributeAtIndex(1, kind).IsNil() {
		t.Fatal(direct.String())
	}
	indirect := mod.NamedFunction("indirect")
	RemapFunctionEffects(old, indirect, ABIMapping{Params: []ABIValue{{Kind: Indirect, Indices: []int{1}}}})
	if !indirect.GetEnumAttributeAtIndex(1, kind).IsNil() {
		t.Fatal(indirect.String())
	}
	WidenForUnknownInstrumentation(direct)
	if !direct.GetEnumAttributeAtIndex(2, kind).IsNil() {
		t.Fatal("noalias survived unknown instrumentation")
	}
}

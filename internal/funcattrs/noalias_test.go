package funcattrs

import (
	"strings"
	"testing"
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
			attrs, sig, err := parseTest(t, "//llgo:param(p) noalias\nfunc F(p, q *int32) int32 { return 0 }")
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"rewrite", "read_alias"} {
				if err := ApplyPointerEffects(mod.Context(), mod.NamedFunction(name), sig, attrs, 0); err != nil {
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

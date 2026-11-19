package llvmfmv

import (
	"github.com/xgo-dev/llvm"
	"strings"
	"testing"
)

func TestGuardedFeatureCombinations(t *testing.T) {
	m := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx512() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
define void @helper(ptr %p) "llgo.fmv.avx2-entry" {
 %v=load <16 x float>,ptr %p,align 4
 %r=fmul <16 x float> %v,%v
 store <16 x float> %r,ptr %p,align 4
 ret void
}
define i1 @root(ptr %p) {
 %x=call i1 @avx512()
 %off=xor i1 %x,true
 br i1 %off,label %out,label %wide
wide:
 call void @helper(ptr %p)
 %y=call i1 @fma()
 br i1 %y,label %both,label %out
both:
 call void @helper(ptr %p)
 br label %out
out:
 %observable=call i1 @fma()
 ret i1 %observable
}
`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	wide := m.NamedFunction("root.__llgo_fmv_guard_avx512")
	both := m.NamedFunction("root.__llgo_fmv_guard_avx512_fma")
	base := m.NamedFunction("root.__llgo_fmv_guard_base")
	if wide.IsNil() || both.IsNil() || base.IsNil() {
		t.Fatalf("missing guarded implementations:\n%s", m.String())
	}
	if strings.Contains(wide.String(), "call i1 @avx512()") || !strings.Contains(wide.String(), "call i1 @fma()") {
		t.Fatal(wide.String())
	}
	if strings.Contains(both.String(), "call i1 @fma()") || !strings.Contains(both.String(), "@helper.__llgo_fmv_guard_avx512_fma") {
		t.Fatal(both.String())
	}
	if !strings.Contains(base.String(), "call i1 @avx512()") || strings.Contains(stringAttribute(base, "target-features"), "+avx512f") {
		t.Fatal(base.String())
	}
	before := m.String()
	if err := Run(m, "v1"); err != nil || m.String() != before {
		t.Fatalf("second run changed module: %v", err)
	}
}

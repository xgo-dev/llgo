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

const guardedVectorFixture = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
define void @root(ptr %p) {
 %a = call i1 @wide()
 %b = call i1 @fma()
 %both = and i1 %a, %b
 br i1 %both, label %vector, label %exit
vector:
 %v = load <16 x float>, ptr %p, align 4
 %neg = fneg <16 x float> %v
 store <16 x float> %neg, ptr %p, align 4
 br label %exit
exit:
 ret void
}
define void @unrelated(ptr %p, i1 %flag) {
 %a = call i1 @wide()
 %either = or i1 %a, %flag
 br i1 %either, label %vector, label %exit
vector:
 %v = load <16 x float>, ptr %p, align 4
 %neg = fneg <16 x float> %v
 store <16 x float> %neg, ptr %p, align 4
 br label %exit
exit:
 ret void
}
`

func TestGuardedVectorConjunction(t *testing.T) {
	m := parseModule(t, guardedVectorFixture)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	both := m.NamedFunction("root.__llgo_fmv_guard_avx512_fma")
	if both.IsNil() || !strings.Contains(stringAttribute(both, "target-features"), "+fma") || !strings.Contains(stringAttribute(both, "target-features"), "+avx512f") {
		t.Fatal("missing specialization for the conjunction")
	}
	if !m.NamedFunction("unrelated.__llgo_fmv_guard_base").IsNil() {
		t.Fatal("a disjunction with an independent flag cannot establish AVX512")
	}
}

func TestGuardedSymbolCollision(t *testing.T) {
	for _, symbol := range []string{
		"root.__llgo_fmv_guard_base",
		"root.__llgo_fmv_guard_avx512_fma",
		featureSymbol,
	} {
		t.Run(symbol, func(t *testing.T) {
			m := parseModule(t, guardedVectorFixture+"\n@\""+symbol+"\" = global i8 0\n")
			err := Run(m, "v1")
			if err == nil || !strings.Contains(err.Error(), "symbol collision: "+symbol) {
				t.Fatalf("expected collision on %s, got %v", symbol, err)
			}
		})
	}
}

// A Go helper taking array pointers may do SIMD work without a vector in its
// signature. The guard must reach the actual computation through local calls.
func TestGuardedScalarSignatureHelpers(t *testing.T) {
	m := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare void @external(ptr)
define void @scalar(ptr %p) noinline {
 store i32 9, ptr %p
 ret void
}
define void @kernel(ptr %p) noinline {
 %v = load <16 x i32>, ptr %p, align 4
 %r = add <16 x i32> %v, %v
 store <16 x i32> %r, ptr %p, align 4
 ret void
}
define void @helper(ptr %p, i1 %again) noinline {
 br i1 %again, label %recurse, label %work
recurse:
 call void @helper(ptr %p, i1 false)
 br label %work
work:
 call void @kernel(ptr %p)
 ret void
}
define void @root(ptr %p, ptr %callback) {
 %a = call i1 @wide()
 br i1 %a, label %work, label %exit
work:
 call void @scalar(ptr %p)
 call void @external(ptr %p)
 call void %callback(ptr %p)
 call void @helper(ptr %p, i1 true)
 br label %exit
exit:
 ret void
}
`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	const suffix = ".__llgo_fmv_guard_avx512"
	root, helper, kernel := m.NamedFunction("root"+suffix), m.NamedFunction("helper"+suffix), m.NamedFunction("kernel"+suffix)
	if root.IsNil() || helper.IsNil() || kernel.IsNil() {
		t.Fatalf("guard did not reach scalar-signature helper and kernel:\n%s", m.String())
	}
	for _, call := range []string{"@scalar(", "@external(", "%callback(", "@helper" + suffix + "("} {
		if !strings.Contains(root.String(), "call void "+call) {
			t.Fatalf("missing call to %s:\n%s", call, root.String())
		}
	}
	if !strings.Contains(helper.String(), "call void @kernel"+suffix) || !strings.Contains(helper.String(), "call void @helper"+suffix) {
		t.Fatal(helper.String())
	}
	if !m.NamedFunction("scalar"+suffix).IsNil() || !strings.Contains(stringAttribute(kernel, "target-features"), "+avx512f") {
		t.Fatalf("incorrect specialization set:\n%s", m.String())
	}
	if strings.Contains(m.NamedFunction("helper").String(), suffix) {
		t.Fatal("unguarded helper calls a specialized entry")
	}
}

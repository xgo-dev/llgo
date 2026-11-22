package llvmfmv

import (
	"fmt"
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
	for _, feature := range []string{"avx512", "avx2"} {
		t.Run(feature, func(t *testing.T) { testGuardedScalarSignatureHelpers(t, feature) })
	}
}

func testGuardedScalarSignatureHelpers(t *testing.T, feature string) {
	t.Helper()
	const ir = `
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
`
	m := parseModule(t, strings.Replace(ir, "x86.avx512", "x86."+feature, 1))
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	suffix := ".__llgo_fmv_guard_" + feature
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
	if !m.NamedFunction("scalar"+suffix).IsNil() || !strings.Contains(stringAttribute(kernel, "target-features"), "+"+feature) {
		t.Fatalf("incorrect specialization set:\n%s", m.String())
	}
	if strings.Contains(m.NamedFunction("helper").String(), suffix) {
		t.Fatal("unguarded helper calls a specialized entry")
	}
	before := m.String()
	if err := Run(m, "v1"); err != nil || m.String() != before {
		t.Fatalf("repeated pass changed local guard: %v\n%s", err, m.String())
	}
}

func TestGuaranteedGuardPredicates(t *testing.T) {
	// A non-baseline query must stay observable even in the widest variant.
	const ir = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare i1 @avx() "llgo.cpu.query"="x86.avx"
declare i1 @sha() "llgo.cpu.query"="x86.sha"
define i1 @root(ptr %p) {
 %a = call i1 @avx()
 %b = call i1 @fma()
 %c = call i1 @wide()
 %ab = and i1 %a, %b
 %ok = and i1 %ab, %c
 br i1 %ok, label %fast, label %exit
fast:
 %x = load <16 x float>, ptr %p
 %y = fneg <16 x float> %x
 store <16 x float> %y, ptr %p
 br label %exit
exit:
 %s = call i1 @sha()
 ret i1 %s
}`
	for _, level := range []string{"v3", "v4"} {
		t.Run(level, func(t *testing.T) {
			m := parseModule(t, ir)
			if err := Run(m, level); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			all := m.String()
			if strings.Contains(all, "call i1 @fma(") || strings.Contains(all, "call i1 @avx(") || !strings.Contains(all, "call i1 @sha(") {
				t.Fatal(all)
			}
			hasSlot := !m.NamedGlobal("root" + slotSuffix).IsNil()
			if hasSlot != (level == "v3") || strings.Contains(all, "call i1 @wide(") != (level == "v3") {
				t.Fatal(all)
			}
			if err := Run(m, level); err != nil || all != m.String() {
				t.Fatalf("repeat: %v\n%s", err, m.String())
			}
		})
	}
}

const mixedAVX2GuardFixture = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
declare i1 @sha() "llgo.cpu.query"="x86.sha"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
define void @kernel(ptr %p) "llgo.fmv.avx2-entry" {
 %x = load <16 x i32>, ptr %p
 %y = add <16 x i32> %x, %x
 store <16 x i32> %y, ptr %p
 ret void
}
define void @root(ptr %p) {
 %a = call i1 @avx2()
 br i1 %a, label %check, label %slowcheck
check:
 %b = call i1 @wide()
 br i1 %b, label %fast, label %exit
fast:
 call void @kernel(ptr %p)
 br label %exit
slowcheck:
 %s = call i1 @sha()
 br i1 %s, label %slow, label %exit
slow:
 call void @kernel(ptr %p)
 br label %exit
exit:
 ret void
}`

func TestMixedAVX2GuardHelper(t *testing.T) {
	m := parseModule(t, mixedAVX2GuardFixture)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	suffix := variantSuffix + guardSuffix(predicateBits["x86.avx512"])
	body, kernel := m.NamedFunction("root"+suffix), m.NamedFunction("kernel"+suffix)
	if body.IsNil() || kernel.IsNil() || !strings.Contains(body.String(), "call void @kernel"+suffix+"(") || !strings.Contains(stringAttribute(kernel, "target-features"), "+avx512f") {
		t.Fatal(m.String())
	}
	resolver := m.NamedFunction("root" + resolverSuffix).String()
	if !strings.Contains(resolver, "ptr @root"+suffix+",") || strings.Contains(resolver, "ptr @root"+variantSuffix+",") {
		t.Fatalf("public resolver points at a second dispatcher:\n%s", resolver)
	}
	base := "root" + baselineSuffix
	if !strings.Contains(resolver, "ptr @"+base+guardSuffix(predicateBits["x86.sha"])+",") ||
		!strings.Contains(resolver, "musttail call void @"+base+symbolPrefix+"guard_base(") {
		t.Fatalf("baseline arm or pre-initialization call still dispatches:\n%s", resolver)
	}
	before := m.String()
	if err := Run(m, "v1"); err != nil || m.String() != before {
		t.Fatalf("repeat: %v\n%s", err, m.String())
	}

}

func TestGuardHelperCollisionLeavesModuleIntact(t *testing.T) {
	m := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
define void @helper(ptr %p) {
 %x = load <16 x float>, ptr %p
 %y = fneg <16 x float> %x
 store <16 x float> %y, ptr %p
 ret void
}
define void @root(ptr %p) {
 %c = call i1 @wide()
 br i1 %c, label %yes, label %no
yes:
 call void @helper(ptr %p)
 br label %no
no:
 ret void
}
@helper.__llgo_fmv_guard_avx512 = global i8 0
`)
	before := m.String()
	if err := Run(m, "v1"); err == nil || !strings.Contains(err.Error(), "helper.__llgo_fmv_guard_avx512") {
		t.Fatalf("collision: %v", err)
	}
	if m.String() != before {
		t.Fatalf("partial transformation:\n%s", m.String())
	}
}

func TestOptionalGuardAtV4(t *testing.T) {
	ir := strings.ReplaceAll(guardedVectorFixture, `"x86.fma"`, `"x86.avx512vnni"`)
	m := parseModule(t, ir)
	if err := Run(m, "v4"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	v := m.NamedFunction("root.__llgo_fmv_guard_avx512vnni")
	if v.IsNil() || m.NamedGlobal("root"+slotSuffix).IsNil() || !strings.Contains(stringAttribute(v, "target-features"), "+avx512vnni") {
		t.Fatal(m.String())
	}
}

func TestGuardBodyEntryAttributes(t *testing.T) {
	ir := strings.Replace(guardedVectorFixture, "define void @root(ptr %p) {", "define void @root(ptr %p) \""+EntryAttribute+"\" \""+InlineEntryAttribute+"\" {", 1)
	m := parseModule(t, ir)
	if err := runGuardedFeatures(m, nil, "v1"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"guard_avx512_fma", "guard_base"} {
		v := m.NamedFunction("root" + symbolPrefix + suffix)
		if v.IsNil() || hasAttribute(v, EntryAttribute) || hasAttribute(v, InlineEntryAttribute) {
			t.Fatal(m.String())
		}
	}
}

func TestMixedGuardCollisionLeavesModuleIntact(t *testing.T) {
	const collision = "kernel.__llgo_fmv_avx2.__llgo_fmv_guard_avx512"
	m := parseModule(t, mixedAVX2GuardFixture+"\n@"+collision+" = global i8 0\n")
	before := m.String()
	if err := Run(m, "v1"); err == nil || !strings.Contains(err.Error(), collision) {
		t.Fatalf("expected collision on %s, got %v", collision, err)
	}
	if got := m.String(); got != before {
		t.Fatalf("mixed-pass collision changed module:\n%s", got)
	}
	m.NamedGlobal(collision).EraseFromParentAsGlobal()
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if m.NamedFunction(collision).IsNil() || m.NamedMetadataNumOperands(moduleDone) != 1 {
		t.Fatal("retry did not finish guarded specialization")
	}
}

func TestBaselineFoldCollisionLeavesModuleIntact(t *testing.T) {
	const ir = `
 target triple = "x86_64-unknown-linux-gnu"
 declare i1 @sha() "llgo.cpu.query"="x86.sha"
 declare i1 @fma() "llgo.cpu.query"="x86.fma"
 define i1 @unrelated() {
  %b = call i1 @fma()
  ret i1 %b
 }
 define void @root(ptr %p) {
  %s = call i1 @sha()
  %b = call i1 @fma()
  %both = and i1 %s, %b
  br i1 %both, label %vector, label %out
 vector:
  %x = load <4 x i32>, ptr %p
  %y = add <4 x i32> %x, %x
  store <4 x i32> %y, ptr %p
  br label %out
 out:
  ret void
 }`
	const collision = "root.__llgo_fmv_guard_sha"
	for _, level := range []string{"v3", "v4"} {
		t.Run(level, func(t *testing.T) {
			m := parseModule(t, ir+"\n@"+collision+" = global i8 0\n")
			before := m.String()
			if err := Run(m, level); err == nil || !strings.Contains(err.Error(), collision) {
				t.Fatalf("expected collision on %s, got %v", collision, err)
			}
			if got := m.String(); got != before {
				t.Errorf("collision changed module:\n%s", got)
			}
			m.NamedGlobal(collision).EraseFromParentAsGlobal()
			if err := Run(m, level); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if m.NamedFunction(collision).IsNil() || !strings.Contains(m.NamedFunction("unrelated").String(), "ret i1 true") {
				t.Fatal(m.String())
			}
		})
	}
}

func TestIndependentGuardResolver(t *testing.T) {
	const ir = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
define void @root(ptr %p, ptr %q) {
 %a = call i1 @fma()
 br i1 %a, label %first, label %next
first:
 %x = load <4 x float>, ptr %p
 %nx = fneg <4 x float> %x
 store <4 x float> %nx, ptr %p
 br label %next
next:
 %b = call i1 @wide()
 br i1 %b, label %second, label %exit
second:
 %y = load <16 x float>, ptr %q
 %ny = fneg <16 x float> %y
 store <16 x float> %ny, ptr %q
 br label %exit
exit:
 ret void
}`
	for _, tc := range []struct {
		bits   uint64
		suffix string
	}{
		{0, "guard_base"}, {256, "guard_avx512"}, {260, "guard_avx512"}, {512, "guard_fma"}, {516, "guard_fma"}, {768, "guard_avx512_fma"}, {772, "guard_avx512_fma"},
	} {
		m := parseModule(t, ir)
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		want := m.NamedFunction("root" + symbolPrefix + tc.suffix)
		if want.IsNil() {
			t.Fatalf("missing %s", tc.suffix)
		}
		if tc.bits == 772 {
			body := want.String()
			features := stringAttribute(want, "target-features")
			if strings.Contains(body, "call i1") || !strings.Contains(body, "store <4 x float>") || !strings.Contains(body, "store <16 x float>") || !strings.Contains(features, "+fma") || !strings.Contains(features, "+avx512f") {
				t.Fatal(body)
			}
		}
		checkCachedGuard(t, m, tc.bits, want)
	}
}

func checkCachedGuard(t *testing.T, m llvm.Module, bits uint64, want llvm.Value) {
	t.Helper()
	resolver := m.NamedFunction("root" + resolverSuffix)
	// Substitute the published runtime mask and fold the actual select chain.
	for inst := resolver.EntryBasicBlock().FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if !inst.IsALoadInst().IsNil() && inst.Operand(0) == m.NamedGlobal(featureSymbol) {
			inst.ReplaceAllUsesWith(llvm.ConstInt(inst.Type(), bits|featureInitialized, false))
			break
		}
	}
	simplifyGuards(resolver)
	published := false
	for bb := resolver.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsAStoreInst().IsNil() && inst.Operand(1) == m.NamedGlobal("root"+slotSuffix) {
				published = inst.Operand(0) == want
			}
		}
	}
	if !published {
		t.Fatalf("mask %d did not cache %s:\n%s", bits, want.Name(), resolver.String())
	}
}

func TestGuardRuntimePredicates(t *testing.T) {
	for _, p := range x86Predicates {
		t.Run(p.method, func(t *testing.T) {
			for _, published := range []uint64{0, featureAVX, p.runtimeBit} {
				m := parseModule(t, fmt.Sprintf(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @query() "llgo.cpu.query"="%s"
define void @root(ptr %%p) {
 %%q = call i1 @query()
 br i1 %%q, label %%yes, label %%exit
yes:
 %%v = load <4 x i32>, ptr %%p
 store <4 x i32> %%v, ptr %%p
 br label %%exit
exit:
 ret void
}`, X86Query(p.method)))
				if err := Run(m, "v1"); err != nil {
					t.Fatal(err)
				}
				base, fast := symbolPrefix+"guard_base", guardSuffix(p.runtimeBit)
				if p.runtimeBit == featureAVX2 {
					base, fast = baselineSuffix, variantSuffix
				} else if !strings.Contains(m.NamedFunction("root"+base).String(), "call i1 @query()") {
					t.Fatal("baseline lost the observable Go predicate")
				}
				want := m.NamedFunction("root" + base)
				if published&p.runtimeBit != 0 {
					want = m.NamedFunction("root" + fast)
				}
				checkCachedGuard(t, m, published, want)
			}
		})
	}
}

func TestCanonicalAVX2EntrySharesGuardedKernel(t *testing.T) {
	m := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define void @kernel(ptr %p) noinline {
 %v = load <8 x i32>, ptr %p
 %r = add <8 x i32> %v, %v
 store <8 x i32> %r, ptr %p
 ret void
}
define void @root(ptr %p) "llgo.fmv.avx2-entry" {
 %q = call i1 @avx2()
 br i1 %q, label %yes, label %exit
yes:
 call void @kernel(ptr %p)
 br label %exit
exit:
 ret void
}
define void @caller(ptr %p) {
 %q = call i1 @avx2()
 br i1 %q, label %yes, label %no
yes:
 call void @root(ptr %p)
 ret void
no:
 call void @root(ptr null)
 ret void
}`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	guard := guardSuffix(featureAVX2)
	for _, call := range []struct{ from, to string }{
		{"root" + variantSuffix, "kernel" + guard},
		{"root" + guard, "kernel" + guard},
		{"caller" + variantSuffix, "root" + variantSuffix},
		{"caller" + baselineSuffix, "root" + symbolPrefix + "guard_base"},
	} {
		fn := m.NamedFunction(call.from)
		if fn.IsNil() || !strings.Contains(fn.String(), "call void @"+call.to+"(") {
			t.Fatalf("%s did not call %s directly:\n%s", call.from, call.to, m.String())
		}
	}
	// An AVX2-only runtime snapshot selects the same kernel on either entry.
	checkCachedGuard(t, m, featureAVX2, m.NamedFunction("root"+guard))
}

func TestGuardCombinationBudget(t *testing.T) {
	masks := dispatchMasks(map[uint64]bool{2: true, 4: true, 8: true, 16: true, 32: true})
	if len(masks) != 6 || masks[len(masks)-1] != 62 {
		t.Fatalf("expected five observed masks and their full union, got %v", masks)
	}
}

func TestOpaqueGuardedCallsKeepDirectEntry(t *testing.T) {
	for _, callee := range []string{
		`declare void @kernel(ptr) "llgo.fmv.avx2-entry"`,
		`define void @kernel(ptr %p) "llgo.fmv.avx2-entry" { store i32 7,ptr %p
ret void }`,
	} {
		m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
`+callee+`
define void @root(ptr %p) {
 %q=call i1 @wide()
 br i1 %q,label %yes,label %no
yes: call void @kernel(ptr %p)
 br label %no
no: ret void
}`)
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		if !m.NamedGlobal("root"+slotSuffix).IsNil() || !strings.Contains(m.NamedFunction("root").String(), "call void @kernel(") {
			t.Fatal(m.String())
		}
	}
}

func TestGuardVectorReturningCalls(t *testing.T) {
	for _, tc := range []struct {
		callee      string
		specializes bool
	}{
		{"opaque", false},
		{"llvm.sqrt.v4f32", true},
	} {
		m := parseModule(t, strings.ReplaceAll(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare <4 x float> @callee(<4 x float>)
define <4 x float> @root(<4 x float> %x) {
 %q = call i1 @wide()
 br i1 %q, label %yes, label %no
yes:
 %v = call <4 x float> @callee(<4 x float> %x)
 ret <4 x float> %v
no:
 ret <4 x float> %x
}`, "@callee", "@"+tc.callee))
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		if got := !m.NamedGlobal("root" + slotSuffix).IsNil(); got != tc.specializes {
			t.Errorf("%s: specialized = %t, want %t", tc.callee, got, tc.specializes)
		}
	}
}

func TestGuardBodyLimit(t *testing.T) {
	observed := make(map[uint64]bool)
	for i := 1; i <= 48; i++ {
		observed[uint64(i)<<1] = true
	}
	masks := dispatchMasks(observed)
	if len(masks) != maxGuardChoices || masks[len(masks)-1] != 126 {
		t.Fatal(masks)
	}
	var ir strings.Builder
	ir.WriteString(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
define void @root(ptr %p) {
 %q=call i1 @wide()
 br i1 %q,label %yes,label %next
yes: call void @kernel0(ptr %p)
 br label %next
next:
 %f=call i1 @fma()
 br i1 %f,label %fmayes,label %no
fmayes: call void @kernel0(ptr %p)
 br label %no
no: ret void
}
`)
	for i := 0; i < maxGuardBodies+4; i++ {
		fmt.Fprintf(&ir, "define void @kernel%d(ptr %%p) {\n", i)
		if i < maxGuardBodies+3 {
			fmt.Fprintf(&ir, "call void @kernel%d(ptr %%p)\n", i+1)
		} else {
			ir.WriteString("%x = load <16 x i32>,ptr %p\nstore <16 x i32> %x,ptr %p\n")
		}
		ir.WriteString("ret void\n}\n")
	}
	m := parseModule(t, ir.String())
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	count := 0
	for f := m.FirstFunction(); !f.IsNil(); f = llvm.NextFunction(f) {
		if strings.Contains(f.Name(), symbolPrefix) {
			count++
		}
	}
	if count != maxGuardBodies {
		t.Fatalf("created %d bodies, want %d", count, maxGuardBodies)
	}
	union := guardSuffix(256 | 512)
	if !strings.Contains(m.NamedFunction("root"+union).String(), "call void @kernel0"+union) {
		t.Fatal("widest root did not get helper budget")
	}
	if !strings.Contains(m.NamedFunction("kernel58"+union).String(), "call void @kernel59(") {
		t.Fatal("budget boundary did not preserve the ordinary call")
	}
}

func TestAVX2GuardExposedInSpecializedBody(t *testing.T) {
	m := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define void @root(ptr %p) {
 %f = call i1 @fma()
 br i1 %f, label %first, label %check
first:
 %x = load volatile <4 x i32>, ptr %p
 br label %check
check:
 %a = call i1 @avx2()
 %enabled = select i1 %f, i1 %a, i1 false
 br i1 %enabled, label %second, label %exit
second:
 %y = load volatile <4 x i32>, ptr %p
 br label %exit
exit:
 ret void
}`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	fma := m.NamedFunction("root.__llgo_fmv_guard_fma")
	if fma.IsNil() || !strings.Contains(fma.String(), "br i1 %a,") {
		t.Fatalf("expected a direct AVX2 guard after folding FMA:\n%s", m.String())
	}
	before := m.String()
	if err := Run(m, "v1"); err != nil || m.String() != before {
		t.Fatalf("second pass changed the specialized body: %v\n%s", err, m.String())
	}
}

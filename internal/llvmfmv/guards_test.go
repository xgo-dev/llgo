package llvmfmv

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

const guardedVectorFixture = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.avx512"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <16 x float> @llvm.fma.v16f32(<16 x float>, <16 x float>, <16 x float>)
define void @root(ptr %p) {
 %a = call i1 @wide()
 %b = call i1 @fma()
 %both = and i1 %a, %b
 br i1 %both, label %vector, label %exit
vector:
 %v = load <16 x float>, ptr %p, align 4
 %neg = call <16 x float> @llvm.fma.v16f32(<16 x float> %v, <16 x float> %v, <16 x float> %v)
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
 %neg = call <16 x float> @llvm.fma.v16f32(<16 x float> %v, <16 x float> %v, <16 x float> %v)
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
	both := m.NamedFunction("root.__llgo_fmv_guard_zmm_fma")
	if both.IsNil() || !strings.Contains(stringAttribute(both, "target-features"), "+fma") || !strings.Contains(stringAttribute(both, "target-features"), "+avx512f") {
		t.Fatal("missing specialization for the conjunction")
	}
	for _, suffix := range []string{"guard_base", "guard_ymm", "guard_zmm"} {
		body := m.NamedFunction("unrelated" + symbolPrefix + suffix)
		if body.IsNil() || !strings.Contains(body.String(), "call i1 @wide()") || !strings.Contains(body.String(), "or i1") {
			t.Fatalf("width alone folded the independent branch in %s", suffix)
		}
	}
}

func TestGuardedVectorDisjunction(t *testing.T) {
	ir := strings.Replace(guardedVectorFixture, "and i1", "or i1", 1)
	ir = strings.Replace(ir, `"x86.fma"`, `"x86.avx512vbmi2"`, 1)
	m := parseModule(t, ir)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	ext := predicateBits[X86Query("AVX512VBMI2")]
	body := m.NamedFunction("root" + symbolPrefix + "guard_zmm")
	if body.IsNil() || !strings.Contains(body.String(), "call i1 @wide()") || !strings.Contains(body.String(), "call i1 @fma()") {
		t.Fatal(m.String())
	}
	checkCachedGuard(t, m, ext, body)
}

func TestGuardedSymbolCollision(t *testing.T) {
	for _, symbol := range []string{
		"root.__llgo_fmv_guard_base",
		"root.__llgo_fmv_guard_zmm_fma",
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

func TestOptionalGuardAtV4(t *testing.T) {
	ir := strings.ReplaceAll(guardedVectorFixture, `"x86.fma"`, `"x86.avx512vnni"`)
	ir = strings.Replace(ir, "define void @root", `declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64)
define void @root`, 1)
	ir = strings.Replace(ir, "vector:", `vector:
 call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64 8192)`, 1)
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
	ir := strings.Replace(guardedVectorFixture, "define void @root(ptr %p) {", "define void @root(ptr %p) \""+InlineEntryAttribute+"\" {", 1)
	m := parseModule(t, ir)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"guard_zmm_fma", "guard_base"} {
		v := m.NamedFunction("root" + symbolPrefix + suffix)
		if v.IsNil() || hasAttribute(v, InlineEntryAttribute) {
			t.Fatal(m.String())
		}
	}
}

func TestBaselineFoldCollisionLeavesModuleIntact(t *testing.T) {
	const ir = `
 target triple = "x86_64-unknown-linux-gnu"
 declare i1 @sha() "llgo.cpu.query"="x86.sha"
 declare i1 @fma() "llgo.cpu.query"="x86.fma"
 declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64)
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
  call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64 128)
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

func checkCachedGuard(t *testing.T, m llvm.Module, bits uint64, want llvm.Value) {
	t.Helper()
	checkEntryCachedGuard(t, m, "root", bits, want)
}

func checkEntryCachedGuard(t *testing.T, m llvm.Module, entry string, bits uint64, want llvm.Value) {
	t.Helper()
	target := cachedGuardTarget(t, m, entry, bits)
	if want.IsNil() || target != want {
		t.Fatalf("mask %d cached unexpected body %s for %s", bits, target.Name(), entry)
	}
}

func cachedGuardTarget(t *testing.T, m llvm.Module, entry string, bits uint64) llvm.Value {
	t.Helper()
	resolver := m.NamedFunction(entry + resolverSuffix)
	if resolver.IsNil() {
		t.Fatalf("missing resolver for %s", entry)
	}
	// Substitute the published runtime mask and fold the actual select chain.
	for inst := resolver.EntryBasicBlock().FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
		if !inst.IsALoadInst().IsNil() && inst.Operand(0) == m.NamedGlobal(featureSymbol) {
			inst.ReplaceAllUsesWith(llvm.ConstInt(inst.Type(), bits|featureInitialized, false))
			break
		}
	}
	simplifyGuards(resolver)
	for bb := resolver.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsAStoreInst().IsNil() && inst.Operand(1) == m.NamedGlobal(entry+slotSuffix) {
				return inst.Operand(0)
			}
		}
	}
	t.Fatalf("mask %d did not publish a target:\n%s", bits, resolver.String())
	return llvm.Value{}
}

func TestGuardRuntimePredicates(t *testing.T) {
	for _, p := range x86Predicates {
		t.Run(p.method, func(t *testing.T) {
			bundle := predicateBits[X86Query("AVX512")]
			masks := []uint64{0, featureAVX, p.runtimeBit, featureAVX | p.runtimeBit, bundle, bundle | p.runtimeBit}
			for _, published := range masks {
				m := parseModule(t, fmt.Sprintf(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @query() "llgo.cpu.query"="%s"
declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64)
define void @root(ptr %%p) {
 %%q = call i1 @query()
 br i1 %%q, label %%yes, label %%exit
yes:
 call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64 %d)
 %%v = load <4 x i32>, ptr %%p
 store <4 x i32> %%v, ptr %%p
 br label %%exit
exit:
 ret void
}`, X86Query(p.method), p.runtimeBit))
				if err := Run(m, "v1"); err != nil {
					t.Fatal(err)
				}
				base, fast := symbolPrefix+"guard_base", symbolPrefix+"guard_"+strings.ToLower(p.method)
				if !strings.Contains(m.NamedFunction("root"+base).String(), "call i1 @query()") {
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

func TestOpaqueGuardedCallsKeepDirectEntry(t *testing.T) {
	for _, callee := range []string{
		`declare void @kernel(ptr)`,
		`define void @kernel(ptr %p) { store i32 7,ptr %p
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
		{"llvm.fma.v4f32", true},
	} {
		m := parseModule(t, strings.ReplaceAll(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @wide() "llgo.cpu.query"="x86.fma"
declare <4 x float> @callee(<4 x float>, <4 x float>, <4 x float>)
define <4 x float> @root(<4 x float> %x) {
 %q = call i1 @wide()
 br i1 %q, label %yes, label %no
yes:
 %v = call <4 x float> @callee(<4 x float> %x, <4 x float> %x, <4 x float> %x)
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

func TestGuardedWideCallThroughLongHelperChain(t *testing.T) {
	const depth = 66
	var ir strings.Builder
	ir.WriteString(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @vbmi2() "llgo.cpu.query"="x86.avx512vbmi2"
declare void @wide(<16 x i32>)
define void @root() {
 %q=call i1 @vbmi2()
 br i1 %q,label %yes,label %out
yes: call void @kernel0()
 br label %out
out: ret void
}
`)
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&ir, "define void @kernel%d() noinline {\n", i)
		if i+1 < depth {
			fmt.Fprintf(&ir, "call void @kernel%d()\n", i+1)
		} else {
			ir.WriteString("call void @wide(<16 x i32> zeroinitializer)\n")
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
	for i := 0; i < depth; i++ {
		caller := "root"
		if i > 0 {
			caller = fmt.Sprintf("kernel%d", i-1)
		}
		target := fmt.Sprintf("kernel%d", i)
		fn := m.NamedFunction(caller)
		if !m.NamedGlobal(caller+slotSuffix).IsNil() || !strings.Contains(fn.String(), "call void @"+target+"(") {
			t.Fatalf("%s must call the original %s entry:\n%s", caller, target, fn.String())
		}
	}
	leaf := fmt.Sprintf("kernel%d", depth-1)
	body := m.NamedFunction(leaf + symbolPrefix + "guard_zmm")
	if body.IsNil() || targetWidth(body, "v1") != 512 || !strings.Contains(body.String(), "call void @wide(") {
		t.Fatalf("the final wide call was not lowered at its own site:\n%s", m.NamedFunction(leaf).String())
	}
}

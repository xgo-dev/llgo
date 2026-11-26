package llvmfmv

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSharedCalleeKeepsItsOwnEntry(t *testing.T) {
	const levels = 9
	var ir strings.Builder
	ir.WriteString(`target triple = "x86_64-unknown-linux-gnu"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <4 x float> @llvm.fma.v4f32(<4 x float>, <4 x float>, <4 x float>)
define void @kernel(ptr %p) noinline {
 %q = call i1 @fma()
 br i1 %q, label %work, label %done
work:
 %x = load <4 x float>, ptr %p
 %y = call <4 x float> @llvm.fma.v4f32(<4 x float> %x, <4 x float> %x, <4 x float> %x)
 store <4 x float> %y, ptr %p
 br label %done
done: ret void
}
`)
	for i := range levels {
		pred := x86Predicates[i]
		fmt.Fprintf(&ir, "declare i1 @query%d() %q=%q\n", i, QueryAttribute, X86Query(pred.method))
		next := fmt.Sprintf("layer%d", i+1)
		if i == levels-1 {
			next = "kernel"
		}
		fmt.Fprintf(&ir, `define void @layer%d(ptr %%p) noinline {
 %%q = call i1 @query%d()
 br i1 %%q, label %%direct, label %%via
direct: call void @%s(ptr %%p)
 ret void
via: call void @relay%d(ptr %%p)
 ret void
}
define void @relay%d(ptr %%p) noinline {
 call void @%s(ptr %%p)
 ret void
}
`, i, i, next, i, i, next)
	}
	m := parseModule(t, ir.String())
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		name := fn.Name()
		if strings.HasPrefix(name, "layer") || strings.HasPrefix(name, "relay") {
			if strings.Contains(name, symbolPrefix) || strings.Contains(fn.String(), symbolPrefix) {
				t.Fatalf("a scalar forwarding path inherited the caller's profile:\n%s", fn.String())
			}
		}
	}
	fast := m.NamedFunction("kernel" + symbolPrefix + "guard_fma")
	if fast.IsNil() || !strings.Contains(fast.String(), "call <4 x float> @llvm.fma.") ||
		strings.Contains(fast.String(), "call i1 @fma()") || !strings.Contains(stringAttribute(fast, "target-features"), "+fma") {
		t.Fatal("missing independent FMA implementation at the leaf")
	}
	checkEntryCachedGuard(t, m, "kernel", predicateBits[X86Query("FMA")], fast)
}

func TestIndependentPredicatesDoNotEnumerateSubsets(t *testing.T) {
	predicates := []string{"SHA", "VAES", "AVX512GFNI", "FMA", "AVX512BITALG", "AVX512VBMI", "AVX512VBMI2", "AVX512VNNI", "AVXVNNI"}
	// Each enabled operation sets a separate visible bit. A partially enabled
	// machine must enter a body that still evaluates every unproven source query.
	var ir strings.Builder
	ir.WriteString(`target triple = "x86_64-unknown-linux-gnu"
declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64)
`)
	for _, pred := range predicates {
		fmt.Fprintf(&ir, "declare i1 @%s() %q=%q\n", pred, QueryAttribute, X86Query(pred))
	}
	ir.WriteString("define i32 @root() {\n")
	previous := "0"
	for i, pred := range predicates {
		bit := predicateBits[X86Query(pred)]
		fmt.Fprintf(&ir, ` %%q%d = call i1 @%s()
 br i1 %%q%d, label %%work%d, label %%skip%d
work%d:
 call void @"github.com/xgo-dev/llgo/runtime/internal/runtime.SIMDRequireX86Features"(i64 %d)
 %%on%d = or i32 %s, %d
 br label %%next%d
skip%d: br label %%next%d
next%d:
 %%v%d = phi i32 [ %%on%d, %%work%d ], [ %s, %%skip%d ]
`, i, pred, i, i, i, i, bit, i, previous, 1<<i, i, i, i, i, i, i, i, previous, i)
		previous = fmt.Sprintf("%%v%d", i)
	}
	fmt.Fprintf(&ir, " ret i32 %s\n}\n", previous)
	// Each mask starts from the original IR because evaluating the resolver
	// and folding the remaining source queries below mutates the module.
	for _, mask := range []uint64{0, predicateBits[X86Query("FMA")], predicateBits[X86Query("SHA")] | predicateBits[X86Query("AVX512VNNI")]} {
		m := parseModule(t, ir.String())
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		bodies := 0
		for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
			if strings.HasPrefix(fn.Name(), "root"+symbolPrefix+"guard_") {
				bodies++
			}
		}
		// FMA is the only member of the common ISA levels in this fixture.
		if bodies != 2 || m.NamedFunction("root"+symbolPrefix+"guard_base").IsNil() ||
			m.NamedFunction("root"+symbolPrefix+"guard_fma").IsNil() {
			t.Fatalf("%d independent guards generated %d bodies, want just the baseline and FMA", len(predicates), bodies)
		}
		body := m.NamedFunction("root")
		if !m.NamedGlobal("root" + slotSuffix).IsNil() {
			body = cachedGuardTarget(t, m, "root", mask)
		}
		// Simulate the effective Go predicates; only the reachable blocks may
		// contribute. Query-only operations cannot disappear on a partial mask.
		for bb := body.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				callee := directGuardCallee(inst)
				if callee.IsNil() {
					continue
				}
				if bit := predicateBits[checkedQuery(callee)]; bit != 0 {
					var v uint64
					if bit&mask != 0 {
						v = 1
					}
					inst.ReplaceAllUsesWith(llvm.ConstInt(inst.Type(), v, false))
				}
			}
		}
		simplifyGuards(body)
		optimizePolicyModule(t, m, "function(instcombine,simplifycfg)")
		want := 0
		for i, pred := range predicates {
			if mask&predicateBits[X86Query(pred)] != 0 {
				want |= 1 << i
			}
		}
		if !strings.Contains(body.String(), fmt.Sprintf("ret i32 %d\n", want)) {
			t.Fatalf("mask %#x: expected %d in %s:\n%s", mask, want, body.Name(), body.String())
		}
	}
}

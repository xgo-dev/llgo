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
	predicates := []string{"FMA", "AVX512BITALG", "AVX512VPOPCNTDQ"}
	// Each enabled operation sets a separate visible bit. A partially enabled
	// machine must enter a body that still evaluates every unproven source query.
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare i1 @bytepop() "llgo.cpu.query"="x86.avx512bitalg"
declare i1 @wordpop() "llgo.cpu.query"="x86.avx512vpopcntdq"
declare <4 x float> @llvm.fma.v4f32(<4 x float>, <4 x float>, <4 x float>)
declare <16 x i8> @llvm.ctpop.v16i8(<16 x i8>)
declare <4 x i32> @llvm.ctpop.v4i32(<4 x i32>)
define i32 @root(ptr %p) {
entry:
 %f = call i1 @fma()
 br i1 %f, label %fma, label %next
fma:
 %x = load <4 x float>, ptr %p
 %y = call <4 x float> @llvm.fma.v4f32(<4 x float> %x, <4 x float> %x, <4 x float> %x)
 store <4 x float> %y, ptr %p
 br label %next
next:
 %first = phi i32 [ 0, %entry ], [ 1, %fma ]
 %b = call i1 @bytepop()
 br i1 %b, label %bytes, label %last
bytes:
 %bi = load <16 x i8>, ptr %p
 %bo = call <16 x i8> @llvm.ctpop.v16i8(<16 x i8> %bi)
 store <16 x i8> %bo, ptr %p
 %setbyte = or i32 %first, 2
 br label %last
last:
 %second = phi i32 [ %first, %next ], [ %setbyte, %bytes ]
 %w = call i1 @wordpop()
 br i1 %w, label %words, label %done
words:
 %wi = load <4 x i32>, ptr %p
 %wo = call <4 x i32> @llvm.ctpop.v4i32(<4 x i32> %wi)
 store <4 x i32> %wo, ptr %p
 %setword = or i32 %second, 4
 ret i32 %setword
done:
 ret i32 %second
}`
	// Each mask starts from the original IR because evaluating the resolver
	// and folding the remaining source queries below mutates the module.
	for want := 0; want < 1<<len(predicates); want++ {
		var mask uint64
		for i, pred := range predicates {
			if want&(1<<i) != 0 {
				mask |= predicateBits[X86Query(pred)]
			}
		}
		m := parseModule(t, ir)
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
		body := cachedGuardTarget(t, m, "root", mask)
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
		if !strings.Contains(body.String(), fmt.Sprintf("ret i32 %d\n", want)) {
			t.Fatalf("mask %#x: expected %d in %s:\n%s", mask, want, body.Name(), body.String())
		}
	}
}

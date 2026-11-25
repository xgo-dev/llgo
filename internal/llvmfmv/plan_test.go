package llvmfmv

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Eight independent predicates control eight calls, but they all establish the
// same register width for a helper that only doubles a 512-bit integer vector.
func TestGuardedCallsShareZMMImplementation(t *testing.T) {
	guards := []string{"avx512", "avx512vbmi2", "avx512gfni", "avx512bitalg", "avx512vbmi", "avx512vnni", "avx512vpclmulqdq", "avx512vpopcntdq"}
	var ir strings.Builder
	ir.WriteString(`target triple = "x86_64-unknown-linux-gnu"
define void @kernel(ptr %p) noinline {
 %x = load <16 x i32>, ptr %p, align 4
 %y = add <16 x i32> %x, %x
 store <16 x i32> %y, ptr %p, align 4
 ret void
}
`)
	for _, guard := range guards {
		fmt.Fprintf(&ir, "declare i1 @%s() \"llgo.cpu.query\"=\"x86.%s\"\n", guard, guard)
	}
	ir.WriteString("define void @root(ptr %p) noinline {\n")
	for i, guard := range guards {
		fmt.Fprintf(&ir, ` %%q%d = call i1 @%s()
 br i1 %%q%d, label %%yes%d, label %%next%d
yes%d:
 call void @kernel(ptr %%p)
 br label %%next%d
next%d:
`, i, guard, i, i, i, i, i, i)
	}
	ir.WriteString(" ret void\n}")
	m := parseModule(t, ir.String())
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	root := m.NamedFunction("root")
	var kernel llvm.Value
	for _, guard := range guards {
		callee := guardedCallTarget(t, root, m.NamedFunction(guard))
		if kernel.IsNil() {
			kernel = callee
		}
		if callee != kernel {
			t.Fatalf("guard %s calls %s instead of shared %s", guard, callee.Name(), kernel.Name())
		}
	}
	if targetWidth(kernel, "v1") != 512 || !strings.Contains(kernel.String(), "add <16 x i32>") {
		t.Fatalf("expected direct calls to the ZMM implementation:\n%s", kernel.String())
	}
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if strings.HasPrefix(fn.Name(), "root"+symbolPrefix) ||
			(strings.HasPrefix(fn.Name(), "kernel"+symbolPrefix) && fn != kernel) {
			t.Fatalf("unexpected extra body %s", fn.Name())
		}
	}
}

// A guard inside a pointer helper can control how many wide calls execute.
// The helper inherits ZMM from its caller while the inner query stays dynamic.
func TestKnownZMMEntryPreservesIndependentQuery(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @vbmi2() "llgo.cpu.query"="x86.avx512vbmi2"
declare i1 @vnni() "llgo.cpu.query"="x86.avx512vnni"
declare <16 x i32> @double(<16 x i32>)
define void @kernel(ptr %p) noinline {
 %x = load <16 x i32>, ptr %p, align 4
 %q = call i1 @vnni()
 br i1 %q, label %extra, label %last
extra:
 %extraValue = call <16 x i32> @double(<16 x i32> %x)
 br label %last
last:
 %arg = phi <16 x i32> [ %x, %0 ], [ %extraValue, %extra ]
 %r = call <16 x i32> @double(<16 x i32> %arg)
 store <16 x i32> %r, ptr %p, align 4
 ret void
}
define void @root(ptr %p) noinline {
 %q = call i1 @vbmi2()
 br i1 %q, label %yes, label %done
yes:
 call void @kernel(ptr %p)
 br label %done
done:
 ret void
}`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	kernel := guardedCallTarget(t, m.NamedFunction("root"), m.NamedFunction("vbmi2"))
	if targetWidth(kernel, "v1") != 512 || strings.Contains(stringAttribute(kernel, "target-features"), "+avx512vnni") {
		t.Fatalf("incorrect target for the VBMI2 call:\n%s", kernel.String())
	}
	double := m.NamedFunction("double")
	if guardedCallTarget(t, kernel, m.NamedFunction("vnni")) != double ||
		strings.Count(kernel.String(), "call <16 x i32> @double(") != 2 {
		t.Fatalf("expected one conditional and one unconditional call to double:\n%s", kernel.String())
	}
}

func guardedCallTarget(t *testing.T, fn, query llvm.Value) llvm.Value {
	t.Helper()
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		br := bb.LastInstruction()
		if br.InstructionOpcode() != llvm.Br || br.SuccessorsCount() != 2 || directGuardCallee(br.Operand(0)) != query {
			continue
		}
		for i := br.Successor(0).FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
			if callee := directGuardCallee(i); !callee.IsNil() {
				return callee
			}
		}
	}
	t.Fatalf("%s has no direct call guarded by %s:\n%s", fn.Name(), query.Name(), fn.String())
	return llvm.Value{}
}

// The helper has both an inherited ZMM call ABI and an independent FMA
// operation. Selecting the FMA implementation must preserve the inherited ABI.
func TestKnownZMMEntryDispatchesOptionalFMA(t *testing.T) {
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare i1 @vbmi2() "llgo.cpu.query"="x86.avx512vbmi2"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <4 x float> @llvm.fma.v4f32(<4 x float>, <4 x float>, <4 x float>)
declare void @consume(<16 x i32>)
define void @kernel(ptr %p) noinline {
 %q = call i1 @fma()
 br i1 %q, label %extra, label %wide
extra:
 %x = load <4 x float>, ptr %p
 %y = call <4 x float> @llvm.fma.v4f32(<4 x float> %x, <4 x float> %x, <4 x float> %x)
 store <4 x float> %y, ptr %p
 br label %wide
wide:
 call void @consume(<16 x i32> zeroinitializer)
 ret void
}
define void @root(ptr %p) {
 %q = call i1 @vbmi2()
 br i1 %q, label %yes, label %done
yes: call void @kernel(ptr %p)
 br label %done
done: ret void
}`
	for _, tc := range []struct {
		fma  bool
		body string
	}{{false, "guard_zmm"}, {true, "guard_zmm_fma"}} {
		m := parseModule(t, ir)
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		entry := guardedCallTarget(t, m.NamedFunction("root"), m.NamedFunction("vbmi2"))
		bits := predicateBits[X86Query("AVX512VBMI2")]
		if tc.fma {
			bits |= predicateBits[X86Query("FMA")]
		}
		body := m.NamedFunction("kernel" + symbolPrefix + tc.body)
		checkEntryCachedGuard(t, m, entry.Name(), bits, body)
		if targetWidth(body, "v1") != 512 || !strings.Contains(body.String(), "call void @consume(") {
			t.Fatal(body.String())
		}
	}
}

// A direct AVX2 guard over local work gives kernel a canonical AVX2 entry.
// Callers with an independent ZMM proof need that width on both AVX2 paths.
func TestKnownZMMEntryOfCanonicalAVX2Root(t *testing.T) {
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare i1 @vbmi2() "llgo.cpu.query"="x86.avx512vbmi2"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
declare void @consume(<16 x i32>)
define void @kernel(ptr %p) {
 %q = call i1 @avx2()
 br i1 %q, label %extra, label %wide
extra:
 %x = load <4 x i32>, ptr %p
 %y = add <4 x i32> %x, %x
 store volatile <4 x i32> %y, ptr %p
 br label %wide
wide: call void @consume(<16 x i32> zeroinitializer)
 ret void
}
define void @root(ptr %p) {
 %q = call i1 @vbmi2()
 br i1 %q, label %yes, label %done
yes: call void @kernel(ptr %p)
 br label %done
done: ret void
}`
	for _, avx2 := range []uint64{0, featureAVX2} {
		m := parseModule(t, ir)
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		if m.NamedFunction("kernel" + variantSuffix).IsNil() {
			t.Fatal("fixture must have a canonical AVX2 variant")
		}
		mask := predicateBits[X86Query("AVX512VBMI2")] | avx2
		body := guardedCallTarget(t, m.NamedFunction("root"), m.NamedFunction("vbmi2"))
		if !m.NamedFunction(body.Name() + resolverSuffix).IsNil() {
			body = cachedGuardTarget(t, m, body.Name(), mask)
		}
		for bb := body.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for i := bb.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if directGuardCallee(i) == m.NamedFunction("avx2") {
					i.ReplaceAllUsesWith(llvm.ConstInt(i.Type(), avx2/featureAVX2, false))
				}
			}
		}
		simplifyGuards(body)
		code := body.String()
		if targetWidth(body, "v1") != 512 || !strings.Contains(code, "call void @consume(") ||
			strings.Contains(code, "store volatile <4 x i32>") != (avx2 != 0) {
			t.Fatalf("AVX2=%t lost ZMM or selected wrong Go path:\n%s", avx2 != 0, code)
		}
	}
}

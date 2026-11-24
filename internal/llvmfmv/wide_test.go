package llvmfmv

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestWideCallWidths(t *testing.T) {
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare <8 x float> @ymm(<8 x float>)
declare { [2 x <8 x float>], i64 } @tuple()
declare void @address(ptr)
define i32 @root(i32 %mode, ptr %p, ptr %indirect) {
 switch i32 %mode, label %scalar [i32 1, label %y i32 2, label %z]
y:
 %a = call <8 x float> @ymm(<8 x float> zeroinitializer)
 store <8 x float> %a, ptr %p
 ret i32 1
z:
 %b = call <16 x float> %indirect(<16 x float> zeroinitializer)
 store <16 x float> %b, ptr %p
 ret i32 2
scalar:
 ret i32 3
}`
	for _, tc := range []struct {
		mask   uint64
		suffix string
	}{
		{0, "width_base"},
		{featureAVX, "ymm"},
		{featureAVX2, "ymm"},
		{predicateBits[X86Query("FMA")], "ymm"},
		{featureAVX2 | predicateBits[X86Query("AVX512VBMI2")], "ymm"},
		{predicateBits[X86Query("AVX512")] | predicateBits[X86Query("AVX512VBMI2")], "zmm"},
		{predicateBits[X86Query("AVX512")], "zmm"},
	} {
		mod := parseModule(t, ir)
		if err := Run(mod, "v1"); err != nil {
			t.Fatal(err)
		}
		if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		if stringAttribute(mod.NamedFunction("tuple"), "target-features") != "+avx" || stringAttribute(mod.NamedFunction("address"), "target-features") != "" {
			t.Fatal("unexpected register floor on tuple or pointer signature")
		}
		body := mod.NamedFunction("root" + symbolPrefix + tc.suffix)
		if body.IsNil() {
			t.Fatal(mod.String())
		}
		checkCachedGuard(t, mod, tc.mask, body)
		code := body.String()
		if !strings.Contains(code, "ret i32 3") ||
			strings.Contains(code, "call <8 x float> @ymm") != (tc.suffix != "width_base") ||
			strings.Contains(code, "call <16 x float> %indirect") != (tc.suffix == "zmm") {
			t.Fatal(code)
		}
	}
}

func TestZMMCallDispatch(t *testing.T) {
	const ir = `target triple = "x86_64-unknown-linux-gnu"
declare i1 @vbmi2() "llgo.cpu.query"="x86.avx512vbmi2"
declare i1 @opaque()
declare void @wide(<16 x float>)
define void @root(ptr %p) {
 %ok = call i1 @QUERY()
 br i1 %ok, label %yes, label %exit
yes:
 %v = load <16 x float>, ptr %p
 call void @wide(<16 x float> %v)
 br label %exit
exit:
 ret void
}`
	ext := predicateBits[X86Query("AVX512VBMI2")]
	bundle := predicateBits[X86Query("AVX512")]
	for _, tc := range []struct {
		query  string
		mask   uint64
		suffix string
	}{
		{"vbmi2", ext, "guard_avx512vbmi2"},
		{"vbmi2", ext | bundle, "guard_avx512vbmi2"},
		{"vbmi2", bundle, "guard_base"},
		{"opaque", ext, "width_base"},
		{"opaque", bundle, "zmm"},
	} {
		m := parseModule(t, strings.Replace(ir, "QUERY", tc.query, 1))
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
			t.Fatal(err)
		}
		body := m.NamedFunction("root" + symbolPrefix + tc.suffix)
		if body.IsNil() {
			t.Fatalf("missing %s", tc.suffix)
		}
		checkCachedGuard(t, m, tc.mask, body)
		switch tc.suffix {
		case "width_base":
			if !strings.Contains(body.String(), "@llvm.trap(") {
				t.Fatal(body.String())
			}
		case "guard_avx512vbmi2", "zmm":
			if !strings.Contains(body.String(), "call void @wide(") {
				t.Fatal(body.String())
			}
		}
	}
}

// The target attributes emitted for a predicate already guarantee its register
// width. Its guarded body must not dispatch again just to obtain that width.
func TestGuardRegisterFloor(t *testing.T) {
	mod := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
define void @guard() { ret void }
`)
	fn := mod.NamedFunction("guard")
	for _, p := range x86Predicates {
		fn.AddTargetDependentFunctionAttr("target-features", p.features)
		if got := targetWidth(fn, "v1"); got != p.registerBits {
			t.Errorf("%s: target width %d, want %d", p.method, got, p.registerBits)
		}
	}
}

func TestWideStackForwarding(t *testing.T) {
	const ir = `
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define <8 x float> @many(i32 %a, i32 %b, i32 %c, i32 %d, i32 %e, i32 %f, <8 x float> %x) {
 %ok = call i1 @avx2()
 br i1 %ok, label %yes, label %no
yes:
 %y = fadd <8 x float> %x, %x
 ret <8 x float> %y
no:
 ret <8 x float> %x
}`
	for _, tc := range []struct{ triple, forwarding string }{
		{"x86_64-unknown-linux-gnu", "musttail call"}, {"x86_64-w64-windows-gnu", "notail call x86_vectorcallcc"},
		{"x86_64-w64-mingw32", "notail call x86_vectorcallcc"},
	} {
		m := parseModule(t, "target triple = \""+tc.triple+"\"\n"+ir)
		if err := Run(m, "v1"); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"many", "many" + resolverSuffix} {
			if body := m.NamedFunction(name).String(); !strings.Contains(body, tc.forwarding) {
				t.Fatalf("%s: %s", tc.triple, body)
			}
		}
	}
}

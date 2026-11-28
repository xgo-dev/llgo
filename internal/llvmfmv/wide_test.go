package llvmfmv

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Return the call in the original body and the call that actually reaches the
// target. A narrow body must pass the address to a width-enabled adapter;
// a wide body must retain the original call directly.
func checkFixedCall(t *testing.T, body, target llvm.Value, width int) (site, inner llvm.Value) {
	t.Helper()
	adapted := targetWidth(body, "v1") < width
	for bb := body.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for call := bb.FirstInstruction(); !call.IsNil(); call = llvm.NextInstruction(call) {
			if call.IsACallInst().IsNil() && call.IsAInvokeInst().IsNil() {
				continue
			}
			callee := call.CalledValue()
			if !adapted && callee == target {
				return call, call
			}
			if !adapted || call.ArgOperandsCount() == 0 || call.Operand(0) != target || callee.IsAFunction().IsNil() {
				continue
			}
			if targetWidth(callee, "v1") != width {
				t.Fatalf("wrong adapter width:\n%s", callee.String())
			}
			for op := callee.FirstBasicBlock().FirstInstruction(); !op.IsNil(); op = llvm.NextInstruction(op) {
				if !op.IsACallInst().IsNil() && op.CalledValue() == callee.Param(0) {
					return call, op
				}
			}
		}
	}
	t.Fatalf("%s did not reach %s with a %d-bit call:\n%s", body.Name(), target.Name(), width, body.String())
	return
}

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
		{0, "guard_base"},
		{machineYMM, "guard_ymm"},
		{machineZMM, "guard_zmm"},
		{machineYMM | machineZMM, "guard_zmm"},
		{machineZMM | predicateBits[X86Query("FMA")], "guard_zmm"},
		{featureAVX, "guard_ymm"},
		{featureAVX2, "guard_ymm"},
		{predicateBits[X86Query("FMA")], "guard_ymm"},
		{featureAVX2 | predicateBits[X86Query("AVX512VBMI2")], "guard_zmm"},
		{predicateBits[X86Query("AVX512")] | predicateBits[X86Query("AVX512VBMI2")], "guard_zmm"},
		{predicateBits[X86Query("AVX512")], "guard_zmm"},
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
		checkFixedCall(t, body, mod.NamedFunction("ymm"), 256)
		checkFixedCall(t, body, body.Param(2), 512)
		if !strings.Contains(code, "ret i32 3") {
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
 call void @wide(<16 x float> zeroinitializer)
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
		{"vbmi2", ext, "guard_zmm"},
		{"vbmi2", ext | bundle, "guard_zmm"},
		{"vbmi2", bundle, "guard_zmm"},
		{"vbmi2", 0, "guard_base"},
		{"opaque", ext, "guard_zmm"},
		{"opaque", machineZMM, "guard_zmm"},
		{"opaque", 0, "guard_base"},
		{"opaque", bundle, "guard_zmm"},
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
		if !strings.Contains(body.String(), "call i1 @"+tc.query+"()") || !strings.Contains(body.String(), "br i1") {
			t.Fatal("width changed the Go control flow", body.String())
		}
		checkFixedCall(t, body, m.NamedFunction("wide"), 512)
		if strings.Contains(body.String(), "alloca ") {
			t.Fatal(body.String())
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
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <8 x float> @llvm.fma.v8f32(<8 x float>, <8 x float>, <8 x float>)
define <8 x float> @many(i32 %a, i32 %b, i32 %c, i32 %d, i32 %e, i32 %f, <8 x float> %x) {
 %ok = call i1 @fma()
 br i1 %ok, label %yes, label %no
yes:
 %y = call <8 x float> @llvm.fma.v8f32(<8 x float> %x, <8 x float> %x, <8 x float> %x)
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

// An external caller enters with the fixed ZMM ABI. The pointer-only helper
// selects its own implementation, including its path through the narrow body.
func TestWideSignatureCallsOrdinaryHelper(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare void @consume(<16 x i32>)
define void @entry(<16 x i32> %x, ptr %p) {
 store <16 x i32> %x, ptr %p
 call void @helper(ptr %p)
 ret void
}
define void @helper(ptr %p) {
 %x = load <16 x i32>, ptr %p
 call void @consume(<16 x i32> %x)
 ret void
}`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	entry := m.NamedFunction("entry")
	if targetWidth(entry, "v1") != 512 || !strings.Contains(entry.String(), "call void @helper(ptr %p)") {
		t.Fatalf("wide public entry did not use the ordinary helper:\n%s", entry.String())
	}
	wide := m.NamedFunction("helper" + symbolPrefix + "guard_zmm")
	if wide.IsNil() || !strings.Contains(wide.String(), "call void @consume(") || targetWidth(wide, "v1") != 512 {
		t.Fatal("missing native wide call from helper")
	}
	narrow := m.NamedFunction("helper" + symbolPrefix + "guard_base")
	if narrow.IsNil() || targetWidth(narrow, "v1") != baselineWidth("v1") {
		t.Fatal("missing baseline body without ZMM features")
	}
	checkFixedCall(t, narrow, m.NamedFunction("consume"), 512)
	checkEntryCachedGuard(t, m, "helper", 0, narrow)
}

func TestRegisterAdapterSharing(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
%pair = type { [2 x <8 x i32>], i16 }
declare %pair @direct(ptr nest, <8 x i32>)
define void @root(ptr %ctx, ptr %p, ptr %closure, ptr %plain) {
 %x = load <8 x i32>, ptr %p
 %a = call %pair @direct(ptr nest %ctx, <8 x i32> %x)
 store %pair %a, ptr %p
 %b = call %pair %closure(ptr nest %ctx, <8 x i32> %x)
 store %pair %b, ptr %p
 %c = call %pair %plain(ptr %ctx, <8 x i32> %x)
 store %pair %c, ptr %p
 ret void
}`)
	if err := Run(m, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	body := m.NamedFunction("root" + symbolPrefix + "guard_base")
	direct, _ := checkFixedCall(t, body, m.NamedFunction("direct"), 256)
	closure, nested := checkFixedCall(t, body, body.Param(2), 256)
	plain, ordinary := checkFixedCall(t, body, body.Param(3), 256)
	if direct.CalledValue() != closure.CalledValue() || closure.CalledValue() == plain.CalledValue() {
		t.Fatal("adapter sharing did not follow the target ABI", body.String())
	}
	nest := llvm.AttributeKindID("nest")
	if nested.GetCallSiteEnumAttribute(1, nest).IsNil() || !ordinary.GetCallSiteEnumAttribute(1, nest).IsNil() {
		t.Fatal("closure context attribute changed", nested.String(), ordinary.String())
	}
}

func TestRegisterAdapterAcrossModules(t *testing.T) {
	const ir = `target triple = "%[1]s"
declare void @%[2]s.target(ptr, <8 x i32>)
define void @%[2]s.caller(ptr %%p, ptr %%fn) {
 call void @%[2]s.target(ptr %[3]s %%p, <8 x i32> zeroinitializer)
 call void %%fn(ptr %%p, <8 x i32> zeroinitializer)
 ret void
}`
	for _, tc := range []struct {
		triple string
		comdat bool
	}{
		{"x86_64-unknown-linux-gnu", true},
		{"x86_64-pc-windows-msvc", true},
		{"x86_64-apple-darwin", false},
	} {
		t.Run(tc.triple, func(t *testing.T) {
			var names []string
			for i, pkg := range []string{"left", "right"} {
				promise := []string{"dereferenceable(8)", "dereferenceable_or_null(64)"}[i]
				mod := parseModule(t, fmt.Sprintf(ir, tc.triple, pkg, promise))
				if err := Run(mod, "v1"); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				body := mod.NamedFunction(pkg + ".caller" + symbolPrefix + "guard_base")
				direct, inner := checkFixedCall(t, body, mod.NamedFunction(pkg+".target"), 256)
				indirect, _ := checkFixedCall(t, body, body.Param(1), 256)
				fn := direct.CalledValue()
				if fn != indirect.CalledValue() || strings.Contains(inner.String(), "dereferenceable") {
					t.Fatal("expected a shared adapter without the caller's pointer guarantee", fn.String())
				}
				if fn.Linkage() != llvm.LinkOnceODRLinkage || fn.Visibility() != llvm.HiddenVisibility {
					t.Fatalf("expected a hidden linkonce_odr definition:\n%s", fn.String())
				}
				group := fn.Comdat()
				if (group.C != nil) != tc.comdat || tc.comdat && (group != mod.Comdat(fn.Name()) || group.SelectionKind() != llvm.AnyComdatSelectionKind) {
					t.Fatalf("unexpected adapter COMDAT:\n%s", mod.String())
				}
				names = append(names, fn.Name())
			}
			if names[0] != names[1] {
				t.Fatalf("different names for the same ABI in independent modules: %s, %s", names[0], names[1])
			}
		})
	}
}

func TestRegisterAdapterDistinctContracts(t *testing.T) {
	for _, tc := range []struct {
		name, ir, innerCall string
		types               [2]string
	}{
		// Each module owns its named types. Their names alone cannot identify
		// the storage that the caller copies for a byval argument.
		{"byval", `
%inner = type { [TYPE x i64] }
%outer = type { %inner }
declare void @target(ptr byval(%outer), <8 x i32>)
define void @caller(ptr %p) {
 call void @target(ptr byval(%outer) %p, <8 x i32> zeroinitializer)
 ret void
}`, "byval(%outer)", [2]string{"1", "5"}},
		// Both vectors use the same transport pieces, but the actual variadic
		// argument type has to survive the indirect call through the adapter.
		{"varargs", `
declare void @target(<8 x i32>, ...)
define void @caller() {
 call void (<8 x i32>, ...) @target(<8 x i32> zeroinitializer, <8 x TYPE> zeroinitializer)
 ret void
}`, "<8 x TYPE>", [2]string{"i32", "float"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var name string
			for _, typ := range tc.types {
				mod := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"`+"\n"+strings.ReplaceAll(tc.ir, "TYPE", typ))
				if err := Run(mod, "v1"); err != nil {
					t.Fatal(err)
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				site, inner := checkFixedCall(t, mod.NamedFunction("caller"+symbolPrefix+"guard_base"), mod.NamedFunction("target"), 256)
				if !strings.Contains(inner.String(), strings.ReplaceAll(tc.innerCall, "TYPE", typ)) {
					t.Fatal("target ABI changed", inner.String())
				}
				fn := site.CalledValue()
				if fn.Linkage() != llvm.LinkOnceODRLinkage || fn.Name() == name {
					t.Fatal("expected distinct linkable names for incompatible calls", fn.Name())
				}
				name = fn.Name()
			}
		})
	}
}

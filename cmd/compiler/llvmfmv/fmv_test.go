package llvmfmv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func parseModule(t *testing.T, source string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "fmv.ll")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mod.Dispose)
	return mod
}

const fixture = `
target triple = "x86_64-unknown-linux-gnu"
@address = global ptr @root
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
declare i1 @fma() "llgo.cpu.query"="x86.fma"
declare <4 x float> @external(<4 x float>)
declare <4 x float> @assembly(<4 x float>)
declare void @consume(ptr)
define void @addressOnly() {
  call void @consume(ptr @avx2)
  ret void
}
define <4 x float> @helper(<4 x float> %x) {
  %y = call <4 x float> @external(<4 x float> %x)
  ret <4 x float> %y
}
define <4 x float> @root(<4 x float> %x, ptr %indirect) "disable-tail-calls"="true" {
  %ok = call i1 @avx2()
  %no = xor i1 %ok, true
  br i1 %no, label %fallback, label %fast
fast:
  %sum = fadd <4 x float> %x, %x
  call void asm sideeffect "__llgo_pcsite_000000000000002a_$\7B:uid\7D:\0A.quad 0x000000000000002a", ""()
  %y = call <4 x float> @helper(<4 x float> %sum)
  %z = call <4 x float> %indirect(<4 x float> %y)
  %a = call <4 x float> @assembly(<4 x float> %z)
  %f = call i1 @fma()
  br i1 %f, label %hasfma, label %nofma
hasfma:
  ret <4 x float> %a
nofma:
  ret <4 x float> %z
fallback:
  ret <4 x float> zeroinitializer
}
define <4 x float> @recursive(<4 x float> %x, i32 %n) {
  %end = icmp eq i32 %n, 0
  br i1 %end, label %done, label %again
again:
  %next = sub i32 %n, 1
  %r = call <4 x float> @recursive(<4 x float> %x, i32 %next)
  ret <4 x float> %r
done:
  ret <4 x float> %x
}
define <8 x float> @wide(<8 x float> %x) {
  %ok = call i1 @avx2()
  ret <8 x float> %x
}
!llgo.funcinfo = !{!0}
!llgo.pcline = !{!1}
!0 = !{i32 1, !"root", !"example.root", !"example.go", i32 10, i32 1}
!1 = !{i32 1, i64 42, !"root", !"example.go", i32 12, i32 1}
`

func TestEarlySpecialization(t *testing.T) {
	mod := parseModule(t, fixture)
	if err := Run(mod, "v1"); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
	root := mod.NamedFunction("root").String()
	baseline := mod.NamedFunction("root.__llgo_fmv_guard_base").String()
	resolver := mod.NamedFunction("root.__llgo_fmv_resolve").String()
	variant := mod.NamedFunction("root.__llgo_fmv_guard_avx2").String()
	if !strings.Contains(mod.NamedGlobal("address").String(), "ptr @root") || strings.Contains(mod.NamedGlobal("address").String(), "__llgo_fmv") {
		t.Fatal("function address no longer names baseline entry")
	}
	for _, want := range []string{"load atomic ptr, ptr @root.__llgo_fmv_slot monotonic", "musttail call <4 x float> %target"} {
		if !strings.Contains(root, want) {
			t.Errorf("dispatcher missing %q:\n%s", want, root)
		}
	}
	if strings.Contains(root, "@avx2(") || !strings.Contains(baseline, "@avx2(") || !strings.Contains(baseline, "ret <4 x float> zeroinitializer") {
		t.Fatalf("dispatcher or baseline changed the source guard:\n%s\n%s", root, baseline)
	}
	for _, want := range []string{"fmv.uninitialized:", "musttail call <4 x float> @root.__llgo_fmv_guard_base", "store atomic ptr %selected, ptr @root.__llgo_fmv_slot monotonic"} {
		if !strings.Contains(resolver, want) {
			t.Errorf("resolver missing %q:\n%s", want, resolver)
		}
	}
	for _, want := range []string{"@helper(", "@fma()", "%indirect(", "@assembly("} {
		if !strings.Contains(variant, want) {
			t.Errorf("variant missing %q:\n%s", want, variant)
		}
	}
	for _, unwanted := range []string{"@avx2()", "fallback:", "000000000000002a"} {
		if strings.Contains(variant, unwanted) {
			t.Errorf("variant retains %q:\n%s", unwanted, variant)
		}
	}
	for _, name := range []string{"helper", "external", "recursive", "wide", "assembly", "addressOnly"} {
		if !mod.NamedFunction(name+symbolPrefix+"avx2").IsNil() || !mod.NamedFunction(name+symbolPrefix+"guard_avx2").IsNil() {
			t.Fatalf("ordinary %s gained a variant without local guarded work", name)
		}
	}
	if !strings.Contains(mod.NamedFunction("helper").String(), "@external(") ||
		!strings.Contains(mod.NamedFunction("recursive").String(), "@recursive(") {
		t.Fatal("ordinary direct or recursive call was retargeted")
	}
	for _, want := range []string{`!"root.__llgo_fmv_guard_avx2", !"example.root"`, `!"root.__llgo_fmv_guard_avx2", !"example.go"`, `"target-features"="+avx,+avx2"`} {
		if !strings.Contains(mod.String(), want) {
			t.Errorf("missing %s", want)
		}
	}
	before := mod.String()
	if err := Run(mod, "v1"); err != nil {
		t.Fatal(err)
	}
	if mod.String() != before {
		t.Fatal("FMV is not idempotent")
	}
}

func checkUnversionedFunction(t *testing.T, mod llvm.Module, baseline, name string) {
	t.Helper()
	fn := mod.NamedFunction(name)
	before := fn.String()
	if err := Run(mod, baseline); err != nil {
		t.Fatal(err)
	}
	if fn.String() != before || strings.Contains(mod.String(), symbolPrefix) || !mod.NamedGlobal(featureSymbol).IsNil() {
		t.Fatalf("unexpected FMV changes to %s:\n%s", name, mod.String())
	}
	if mod.NamedMetadataNumOperands(moduleDone) != 1 {
		t.Fatal("missing FMV completion marker")
	}
}

func TestAVX2QueryResultKeepsOrdinaryEntry(t *testing.T) {
	for _, baseline := range []string{"v1", "v2"} {
		t.Run(baseline, func(t *testing.T) {
			mod := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define i1 @ready() {
  %ok = call i1 @avx2()
  ret i1 %ok
}
`)
			checkUnversionedFunction(t, mod, baseline, "ready")
		})
	}
}

func TestStoredQueryKeepsOrdinaryEntry(t *testing.T) {
	mod := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define void @root(ptr %p, ptr %flag) {
 %cpu = call i1 @avx2()
 store volatile i1 %cpu, ptr %flag
 %enabled = load volatile i1, ptr %flag
 br i1 %enabled, label %work, label %done
work:
 %x = load <4 x i32>, ptr %p
 %y = add <4 x i32> %x, %x
 store <4 x i32> %y, ptr %p
 br label %done
done: ret void
}`)
	checkUnversionedFunction(t, mod, "v1", "root")
}

func TestOtherTargetUnchanged(t *testing.T) {
	mod := parseModule(t, strings.Replace(fixture, "x86_64-unknown-linux-gnu", "aarch64-unknown-linux-gnu", 1))
	before := mod.String()
	if err := Run(mod, "v1"); err != nil || mod.String() != before {
		t.Fatalf("non-amd64 module changed: %v", err)
	}
}

func TestSymbolCollision(t *testing.T) {
	for _, symbol := range []string{
		"declare void @root.__llgo_fmv_guard_avx2()",
		"@root.__llgo_fmv_guard_avx2 = global i8 0",
		"@root.__llgo_fmv_guard_avx2 = alias <4 x float> (<4 x float>, ptr), ptr @root",
		"@root.__llgo_fmv_guard_base = global i8 0",
		"declare void @root.__llgo_fmv_resolve()",
		"@root.__llgo_fmv_slot = global ptr null",
		"@\"github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures\" = global i8 0",
	} {
		mod := parseModule(t, fixture+"\n"+symbol+"\n")
		before := mod.String()
		if err := Run(mod, "v1"); err == nil || !strings.Contains(err.Error(), "symbol collision") {
			t.Fatalf("expected collision diagnostic for %s, got %v", symbol, err)
		}
		if mod.String() != before {
			t.Fatal("collision partially mutated module")
		}
	}
}

func TestUnsupportedSignaturesUnchanged(t *testing.T) {
	for name, signature := range map[string]string{
		"variadic":      "void @unsupported(ptr %p, ...)",
		"naked":         "void @unsupported(ptr %p) naked",
		"returns_twice": "void @unsupported(ptr %p) returns_twice",
	} {
		t.Run(name, func(t *testing.T) {
			mod := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define `+signature+` {
 %ok = call i1 @avx2()
 br i1 %ok, label %yes, label %out
yes:
 %x = load <4 x i32>, ptr %p
 %y = add <4 x i32> %x, %x
 store <4 x i32> %y, ptr %p
 br label %out
out:
 ret void
}`)
			checkUnversionedFunction(t, mod, "v1", "unsupported")
		})
	}
}

func TestZeroSizedReceiverAndVoidDispatch(t *testing.T) {
	for _, tc := range []struct{ name, triple, receiver string }{
		{"sysv", "x86_64-unknown-linux-gnu", "{}"},
		{"win64", "x86_64-w64-windows-gnu", "i32"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `
target triple = "x86_64-unknown-linux-gnu"
declare fastcc i1 @avx2({}) "llgo.cpu.query"="x86.avx2"
define fastcc void @root(ptr nonnull %p) {
  %ok = call fastcc i1 @avx2({} zeroinitializer)
  br i1 %ok, label %fast, label %fallback
fast:
  %x = load <8 x i32>, ptr %p
  %y = add <8 x i32> %x, %x
  store <8 x i32> %y, ptr %p
  ret void
fallback:
  store i32 0, ptr %p
  ret void
}
`
			mod := parseModule(t, strings.ReplaceAll(strings.ReplaceAll(source, "x86_64-unknown-linux-gnu", tc.triple), "{}", tc.receiver))
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			root := mod.NamedFunction("root").String()
			variant := mod.NamedFunction("root.__llgo_fmv_guard_avx2").String()
			for _, want := range []string{
				"load atomic ptr, ptr @root.__llgo_fmv_slot monotonic",
				"musttail call fastcc void %target(ptr nonnull %p)",
			} {
				if !strings.Contains(root, want) {
					t.Fatalf("dispatch lost receiver or call ABI %q:\n%s", want, root)
				}
			}
			if strings.Contains(variant, "@avx2(") || strings.Contains(variant, "store i32 0") || !strings.Contains(variant, "add <8 x i32>") {
				t.Fatalf("zero-sized receiver query did not specialize:\n%s", variant)
			}
		})
	}
}

func TestInvalidQuerySignaturesUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name, triple, declaration, call string
	}{
		{"extra receiver", "x86_64-unknown-linux-gnu", "i1 @avx2({}, {})", "i1 @avx2({} zeroinitializer, {} zeroinitializer)"},
		{"nonboolean result", "x86_64-unknown-linux-gnu", "i32 @avx2()", "i32 @avx2()"},
		{"variadic query", "x86_64-unknown-linux-gnu", "i1 @avx2(...)", "i1 (...) @avx2()"},
		{"integer receiver on SysV", "x86_64-unknown-linux-gnu", "i1 @avx2(i32)", "i1 @avx2(i32 0)"},
		{"wrong Win64 receiver width", "x86_64-w64-windows-gnu", "i1 @avx2(i64)", "i1 @avx2(i64 0)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mod := parseModule(t, `target triple = "`+tc.triple+`"
declare `+tc.declaration+` "llgo.cpu.query"="x86.avx2"
define void @root() {
  %q = call `+tc.call+`
  ret void
}
`)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			checkUnversionedFunction(t, mod, "v1", "root")
		})
	}
}

func TestIncompleteAggregateABI(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	// An opaque named type has no defined layout. It must not be mistaken
	// for a defined empty struct just because both have zero fields.
	typ := ctx.StructCreateNamed("Block")
	if supportedType(typ) {
		t.Fatal("incomplete aggregate accepted at the FMV boundary")
	}
	typ.StructSetBody([]llvm.Type{ctx.StructType([]llvm.Type{
		llvm.ArrayType(llvm.VectorType(ctx.FloatType(), 4), 2), ctx.Int64Type(),
	}, false)}, false)
	if !supportedType(typ) {
		t.Fatal("resolved nested SIMD128 aggregate rejected at the FMV boundary")
	}
}

func TestDispatchRequiresFeatureAndWidth(t *testing.T) {
	fma := predicateBits[X86Query("FMA")]
	vbmi2 := predicateBits[X86Query("AVX512VBMI2")]
	wide := predicateBits[X86Query("AVX512")]
	for _, tc := range []struct {
		mask uint64
		impl string
	}{
		{0, "base"}, {fma, "base"}, {vbmi2 | wide, "base"},
		{fma | vbmi2, "impl"}, {fma | wide, "impl"}, {fma | vbmi2 | wide, "impl"},
	} {
		m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
define void @root() { ret void }
define void @base() { ret void }
define void @impl() { ret void }
`)
		emitDispatch(m, newSourceInfo(m), m.NamedFunction("root"), dispatchPlan{
			baseline: m.NamedFunction("base"),
			choices:  []dispatchChoice{{bits: fma, anyOf: wide | vbmi2, fn: m.NamedFunction("impl")}},
		})
		checkCachedGuard(t, m, tc.mask, m.NamedFunction(tc.impl))
	}
}

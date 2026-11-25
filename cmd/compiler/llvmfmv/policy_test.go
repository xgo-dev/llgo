package llvmfmv

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLazyDispatcherPolicy(t *testing.T) {
	mod := parseModule(t, fixture)
	if err := Run(mod, "v1"); err != nil {
		t.Fatal(err)
	}
	if mod.NamedGlobal("root.__llgo_fmv_slot").IsNil() {
		t.Fatal("dispatcher must load a cached implementation, initially the resolver")
	}
	if mod.NamedFunction("root.__llgo_fmv_baseline").IsNil() {
		t.Fatal("baseline body must have its own implementation identity")
	}
	resolver := mod.NamedFunction("root.__llgo_fmv_resolve")
	if resolver.IsNil() || resolver.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
		t.Fatal("only the lazy resolver needs a synthetic noinline boundary")
	}
	if mod.NamedGlobal("root"+slotSuffix).Initializer() != resolver {
		t.Fatal("first call does not enter the resolver")
	}
	for bb := resolver.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		if bb.AsValue().Name() != "fmv.uninitialized" {
			continue
		}
		calls := 0
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsAStoreInst().IsNil() {
				t.Fatal("pre-initialization path cached a fallback")
			}
			if inst.IsACallInst().IsNil() {
				continue
			}
			calls++
			if inst.CalledValue() != mod.NamedFunction("root"+baselineSuffix) || inst.TailCallKind() != llvm.TailCallKindMustTail || inst.GetCallSiteEnumAttribute(-1, llvm.AttributeKindID("noinline")).IsNil() {
				t.Fatal("pre-initialization fallback must remain an opaque tail transfer")
			}
		}
		if calls != 1 {
			t.Fatalf("uninitialized resolver has %d calls, want one tail transfer", calls)
		}
	}
	for _, name := range []string{"root", "root.__llgo_fmv_baseline", "root.__llgo_fmv_avx2"} {
		if !mod.NamedFunction(name).GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
			t.Fatalf("%s must remain eligible for feature-compatible inlining", name)
		}
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
}

const inlineFixture = `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define <4 x float> @helper(<4 x float> %x) "llgo.fmv.avx2-entry" "target-cpu"="x86-64" {
  %y = fadd <4 x float> %x, %x
  ret <4 x float> %y
}
define <4 x float> @root(<4 x float> %x) "target-cpu"="x86-64" {
  %ok = call i1 @avx2()
  br i1 %ok, label %fast, label %slow
fast:
  %v = call <4 x float> @helper(<4 x float> %x)
  ret <4 x float> %v
slow:
  ret <4 x float> %x
}
define <4 x float> @caller(<4 x float> %x) noinline "target-cpu"="x86-64" {
  %v = call <4 x float> @root(<4 x float> %x)
  %r = fmul <4 x float> %v, %x
  ret <4 x float> %r
}
`

func TestCompileBaselineElidesVersions(t *testing.T) {
	for _, baseline := range []string{"v3", "v4"} {
		t.Run(baseline, func(t *testing.T) {
			mod := parseModule(t, inlineFixture)
			if err := Run(mod, baseline); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(mod.String(), "__llgo_fmv") || strings.Contains(mod.NamedFunction("root").String(), "@avx2(") || !mod.NamedGlobal(featureSymbol).IsNil() {
				t.Fatalf("compile baseline retained redundant versioning:\n%s", mod.String())
			}
			root := mod.NamedFunction("root").String()
			if strings.Contains(root, "br i1") || strings.Contains(root, "slow:") || !strings.Contains(root, "call <4 x float> @helper(") {
				t.Fatalf("compile baseline did not select the enabled branch:\n%s", root)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func optimizePolicyModule(t *testing.T, mod llvm.Module, pipeline string) {
	t.Helper()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	target, err := llvm.GetTargetFromTriple(mod.Target())
	if err != nil {
		t.Fatal(err)
	}
	tm := target.CreateTargetMachine(mod.Target(), "x86-64", "", llvm.CodeGenLevelDefault, llvm.RelocDefault, llvm.CodeModelDefault)
	defer tm.Dispose()
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	options.SetVerifyEach(true)
	if err := mod.RunPasses(pipeline, tm, options); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("%v\n%s", err, mod.String())
	}
}

func TestInliningPolicy(t *testing.T) {
	for _, pipeline := range []string{"default<O2>", "thinlto-pre-link<O2>,thinlto<O2>", "lto-pre-link<O2>,lto<O2>"} {
		t.Run(pipeline, func(t *testing.T) {
			mod := parseModule(t, inlineFixture)
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			ctx := mod.Context()
			b := ctx.NewBuilder()
			defer b.Dispose()
			helper := mod.NamedFunction("helper.__llgo_fmv_avx2")
			for _, name := range []string{"compatible", "incompatible"} {
				fn := llvm.AddFunction(mod, name, helper.GlobalValueType())
				fn.AddFunctionAttr(ctx.CreateStringAttribute("target-cpu", "x86-64"))
				if name == "compatible" {
					fn.AddFunctionAttr(ctx.CreateStringAttribute("target-features", "+avx,+avx2"))
				}
				b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
				call := b.CreateCall(helper.GlobalValueType(), helper, fn.Params(), "")
				b.CreateRet(b.CreateFMul(call, fn.Param(0), ""))
			}
			optimizePolicyModule(t, mod, pipeline)
			caller := mod.NamedFunction("caller").String()
			if strings.Contains(caller, "@root(") || strings.Contains(caller, "musttail") || !strings.Contains(caller, "fmul") {
				t.Fatalf("dispatcher did not inline with its caller continuation preserved:\n%s", caller)
			}
			if strings.Contains(mod.NamedFunction("compatible").String(), "@helper.__llgo_fmv_avx2(") {
				t.Fatal("feature-compatible caller failed to inline the implementation")
			}
			if !strings.Contains(mod.NamedFunction("incompatible").String(), "@helper.__llgo_fmv_avx2(") {
				t.Fatal("AVX2 implementation was inlined into a baseline caller")
			}
		})
	}
}

func TestSourceInliningAttributes(t *testing.T) {
	for _, attr := range []string{"noinline", "alwaysinline"} {
		t.Run(attr, func(t *testing.T) {
			mod := parseModule(t, strings.Replace(inlineFixture, `@root(<4 x float> %x) "target-cpu"`, `@root(<4 x float> %x) `+attr+` "target-cpu"`, 1))
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			for _, suffix := range []string{"", baselineSuffix, variantSuffix} {
				if mod.NamedFunction("root" + suffix).GetEnumFunctionAttribute(llvm.AttributeKindID(attr)).IsNil() {
					t.Fatalf("%s lost source %s policy", "root"+suffix, attr)
				}
			}
			resolver := mod.NamedFunction("root" + resolverSuffix)
			if !resolver.GetEnumFunctionAttribute(llvm.AttributeKindID("alwaysinline")).IsNil() || resolver.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
				t.Fatal("resolver did not establish its own synthetic boundary")
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPhysicalFrameDoesNotPinDispatcher(t *testing.T) {
	for _, pipeline := range []string{"default<O2>", "thinlto-pre-link<O2>,thinlto<O2>", "lto-pre-link<O2>,lto<O2>"} {
		t.Run(pipeline, func(t *testing.T) {
			mod := parseModule(t, strings.Replace(inlineFixture, `@root(<4 x float> %x) "target-cpu"`, `@root(<4 x float> %x) noinline "disable-tail-calls"="true" "llgo.fmv.inline-entry" "target-cpu"`, 1))
			if err := Run(mod, "v1"); err != nil {
				t.Fatal(err)
			}
			if !mod.NamedFunction("root").GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
				t.Fatal("physical-frame policy leaked onto the synthetic dispatcher")
			}
			for _, suffix := range []string{baselineSuffix, variantSuffix} {
				impl := mod.NamedFunction("root" + suffix)
				if impl.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() || stringAttribute(impl, "disable-tail-calls") != "true" {
					t.Fatal("dispatcher inlining removed the required physical source frame")
				}
			}
			optimizePolicyModule(t, mod, pipeline)
			caller := mod.NamedFunction("caller").String()
			if strings.Contains(caller, "@root(") || strings.Contains(caller, "musttail") || !strings.Contains(caller, "fmul") {
				t.Fatalf("dispatcher did not inline while preserving its caller continuation:\n%s", caller)
			}
		})
	}
}

func TestDispatcherInlineDebugIdentity(t *testing.T) {
	mod := parseModule(t, `
target triple = "x86_64-unknown-linux-gnu"
declare i1 @avx2() "llgo.cpu.query"="x86.avx2"
define float @root(float %x) "target-cpu"="x86-64" !dbg !4 {
  %ok = call i1 @avx2(), !dbg !5
  br i1 %ok, label %fast, label %slow, !dbg !5
fast:
  %y = fadd float %x, %x, !dbg !5
  ret float %y, !dbg !5
slow:
  ret float %x, !dbg !5
}
define float @middle(float %x) alwaysinline "target-cpu"="x86-64" !dbg !6 {
  %v = call float @root(float %x), !dbg !7
  %r = fmul float %v, %x, !dbg !7
  ret float %r, !dbg !7
}
define float @outer(float %x) noinline "target-cpu"="x86-64" !dbg !8 {
  %v = call float @middle(float %x), !dbg !9
  %r = fadd float %v, %x, !dbg !9
  ret float %r, !dbg !9
}
!llvm.dbg.cu = !{!0}
!llvm.module.flags = !{!10, !11}
!llgo.funcinfo = !{!12}
!0 = distinct !DICompileUnit(language: DW_LANG_Go, file: !1, producer: "llgo", isOptimized: true, runtimeVersion: 0, emissionKind: FullDebug)
!1 = !DIFile(filename: "inline.go", directory: "/src")
!2 = !DISubroutineType(types: !3)
!3 = !{}
!4 = distinct !DISubprogram(name: "root", linkageName: "root", scope: !1, file: !1, line: 10, type: !2, scopeLine: 10, spFlags: DISPFlagDefinition | DISPFlagOptimized, unit: !0)
!5 = !DILocation(line: 11, column: 1, scope: !4)
!6 = distinct !DISubprogram(name: "middle", linkageName: "middle", scope: !1, file: !1, line: 20, type: !2, scopeLine: 20, spFlags: DISPFlagDefinition | DISPFlagOptimized, unit: !0)
!7 = !DILocation(line: 21, column: 1, scope: !6)
!8 = distinct !DISubprogram(name: "outer", linkageName: "outer", scope: !1, file: !1, line: 30, type: !2, scopeLine: 30, spFlags: DISPFlagDefinition | DISPFlagOptimized, unit: !0)
!9 = !DILocation(line: 31, column: 1, scope: !8)
!10 = !{i32 2, !"Dwarf Version", i32 4}
!11 = !{i32 2, !"Debug Info Version", i32 3}
!12 = !{i32 1, !"root", !"root", !"inline.go", i32 10, i32 1}
`)
	if err := Run(mod, "v1"); err != nil {
		t.Fatal(err)
	}
	optimizePolicyModule(t, mod, "default<O2>")
	outer := mod.NamedFunction("outer")
	middle := mod.NamedFunction("middle")
	seen := false
	for bb := outer.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.IsALoadInst().IsNil() || inst.Operand(0) != mod.NamedGlobal("root"+slotSuffix) {
				continue
			}
			seen = true
			loc := inst.InstructionDebugLoc()
			if loc.IsNil() || loc.LocationScope() != middle.Subprogram() || loc.LocationLine() != 21 {
				t.Fatalf("synthetic dispatcher did not inherit its real caller location:\n%s", mod.String())
			}
			parent := loc.LocationInlinedAt()
			if parent.IsNil() || parent.LocationScope() != outer.Subprogram() || parent.LocationLine() != 31 || !parent.LocationInlinedAt().IsNil() {
				t.Fatalf("outer inline chain was lost or gained a synthetic frame:\n%s", mod.String())
			}
		}
	}
	if !seen {
		t.Fatalf("dispatcher was not inlined into outer:\n%s", mod.String())
	}
}

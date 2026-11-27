/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ssaopt_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func buildTestSSAForSpec(t *testing.T, src string, spec target.Spec) (*ssa.Package, *types.Context) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(spec),
	})
	pkg, _, err := ssautil.BuildPackage(&types.Config{Context: ctx, Sizes: ctx}, fset, types.NewPackage("example.com/test", "test"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatalf("build SSA: %v", err)
	}
	return pkg, ctx
}

func buildTestSSA(t *testing.T, src string) (*ssa.Package, *types.Context) {
	return buildTestSSAForSpec(t, src, target.Spec{GOOS: "linux", GOARCH: "amd64"})
}

// TestSSAOptZeroLLVMDependency enforces the architectural guardrail that
// cmd/compiler/ssaopt and cmd/compiler/goescape never depend on LLVM or llvmgen.
func TestSSAOptZeroLLVMDependency(t *testing.T) {
	for _, pkg := range []string{
		"github.com/xgo-dev/llgo/cmd/compiler/ssaopt",
		"github.com/xgo-dev/llgo/cmd/compiler/goescape",
		"github.com/xgo-dev/llgo/cmd/compiler/abipolicy",
	} {
		out, err := exec.Command("go", "list", "-f", `{{join .Deps "\n"}}`, pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("go list %s: %v\n%s", pkg, err, out)
		}
		for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if dep == "github.com/xgo-dev/llvm" || strings.HasPrefix(dep, "github.com/xgo-dev/llvm/") ||
				dep == "github.com/xgo-dev/llgo/cmd/compiler/llvmgen" {
				t.Fatalf("%s unexpectedly depends on LLVM backend package %s", pkg, dep)
			}
		}
	}
}

func TestNilCheckPlanO0VsO2AndVerify(t *testing.T) {
	const src = `package test

type T struct { X int }
func (*T) M(int) {}

func guarded(p *T) {
	if p != nil {
		(*p).M(1)
	}
}

func repeated(p *T) {
	(*p).M(1)
	(*p).M(2)
}

func unguarded(p *T) {
	(*p).M(1)
}
`
	pkg, ctx := buildTestSSA(t, src)

	// At -O0, guarded and repeated must retain active receiver nil checks.
	for _, name := range []string{"guarded", "repeated", "unguarded"} {
		fn := pkg.Func(name)
		plan, err := ssaopt.PlanFunction(fn, ssaopt.Config{
			TypeCtx:  ctx,
			OptLevel: optlevel.O0,
			Verify:   true,
		})
		if err != nil {
			t.Fatalf("O0 PlanFunction(%s): %v", name, err)
		}
		active := 0
		for _, nc := range plan.NilChecks {
			if nc.NeedAddrCheck || nc.NeedBaseCheck {
				active++
			}
		}
		if active == 0 {
			t.Fatalf("O0 %s unexpectedly elided all nil checks: %+v", name, plan.NilChecks)
		}
	}

	// At -O2, guarded(*p).M is elided by dominating p != nil branch, and the
	// second (*p).M in repeated is elided by dominating first (*p).M check.
	planGuarded, err := ssaopt.PlanFunction(pkg.Func("guarded"), ssaopt.Config{
		TypeCtx:  ctx,
		OptLevel: optlevel.O2,
		Verify:   true,
	})
	if err != nil {
		t.Fatalf("O2 PlanFunction(guarded): %v", err)
	}
	var elidedCond int
	for _, nc := range planGuarded.NilChecks {
		if nc.Elided && nc.ProofKind == ssaopt.NilProofDomCond && nc.ProofAnchor != nil {
			elidedCond++
		}
	}
	if elidedCond != 1 {
		t.Fatalf("O2 guarded: expected 1 dom-cond elided nil check, got %d (%+v)", elidedCond, planGuarded.NilChecks)
	}

	planRepeated, err := ssaopt.PlanFunction(pkg.Func("repeated"), ssaopt.Config{
		TypeCtx:  ctx,
		OptLevel: optlevel.O2,
		Verify:   true,
	})
	if err != nil {
		t.Fatalf("O2 PlanFunction(repeated): %v", err)
	}
	var activeRep, elidedRep int
	for _, nc := range planRepeated.NilChecks {
		if nc.Elided && nc.ProofKind == ssaopt.NilProofDomCheck {
			elidedRep++
		} else if nc.NeedAddrCheck {
			activeRep++
		}
	}
	if activeRep != 1 || elidedRep != 1 {
		t.Fatalf("O2 repeated: got active=%d elided=%d, want 1 and 1", activeRep, elidedRep)
	}
}

func TestVerifyPlanRejectsInvalidAnchorsAndBadNoScan(t *testing.T) {
	const src = `package test

type T struct { P *int }
func (*T) M() {}

func f(cond bool, p *T) *T {
	x := new(T)
	if cond {
		(*p).M()
	} else {
		(*p).M()
	}
	return x
}
`
	pkg, ctx := buildTestSSA(t, src)
	fn := pkg.Func("f")
	plan, err := ssaopt.PlanFunction(fn, ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O0, Verify: true})
	if err != nil {
		t.Fatalf("valid plan failed Verify: %v", err)
	}

	// 1. NoScan on pointer-bearing Alloc must be rejected.
	for alloc, ap := range plan.Allocs {
		bad := *plan
		bad.Allocs = map[*ssa.Alloc]ssaopt.AllocPlan{alloc: {Heap: ap.Heap, NoScan: true}}
		if err := ssaopt.VerifyPlan(fn, &bad); err == nil || !strings.Contains(err.Error(), "NoScan") {
			t.Fatalf("expected VerifyPlan to reject NoScan on pointer-bearing alloc, got %v", err)
		}
		break
	}

	// 2. Non-dominating proof anchor (sibling else-branch call proving then-branch call) must be rejected.
	var calls []*ssa.Call
	for ins := range plan.NilChecks {
		if c, ok := ins.(*ssa.Call); ok {
			calls = append(calls, c)
		}
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls in f, got %d", len(calls))
	}
	badDom := &ssaopt.FunctionPlan{
		Fn: fn,
		NilChecks: map[ssa.Instruction]ssaopt.NilCheckPlan{
			calls[0]: {
				Elided:      true,
				ProofKind:   ssaopt.NilProofDomCheck,
				ProofAnchor: calls[1], // sibling branch does NOT dominate calls[0]
			},
		},
	}
	if err := ssaopt.VerifyPlan(fn, badDom); err == nil || !strings.Contains(err.Error(), "dominate") {
		t.Fatalf("expected VerifyPlan to reject non-dominating proof anchor, got %v", err)
	}

	// 3. Elided NilCheckPlan with NilProofNone must be rejected.
	badNoProof := &ssaopt.FunctionPlan{
		Fn: fn,
		NilChecks: map[ssa.Instruction]ssaopt.NilCheckPlan{
			calls[0]: {Elided: true, ProofKind: ssaopt.NilProofNone},
		},
	}
	if err := ssaopt.VerifyPlan(fn, badNoProof); err == nil || !strings.Contains(err.Error(), "ProofKind") {
		t.Fatalf("expected VerifyPlan to reject elided check with NilProofNone, got %v", err)
	}
}

func TestPureGoEscapeAndAllocPlanNoScan(t *testing.T) {
	const src = `package test

type Scalar struct { A int; B float64 }
type HasPtr struct { P *int }

var sink any

func escapeAndStack() *Scalar {
	local := new(Scalar)
	local.A = 1
	escaped := new(Scalar)
	escaped.A = local.A + 2
	ptrObj := new(HasPtr)
	sink = ptrObj
	return escaped
}
`
	pkg, ctx := buildTestSSA(t, src)
	esc := ssaopt.PrepareEscapeAnalysis(pkg.Prog, ssaopt.EscapeConfig{
		TypeCtx:         ctx,
		PreloadedSyntax: true,
	})
	plan, err := ssaopt.PlanFunction(pkg.Func("escapeAndStack"), ssaopt.Config{
		TypeCtx:  ctx,
		OptLevel: optlevel.O2,
		Escape:   esc,
		Verify:   true,
	})
	if err != nil {
		t.Fatalf("PlanFunction: %v", err)
	}
	var stackScalar, heapScalarNoScan, heapPtrScanned int
	for alloc, ap := range plan.Allocs {
		elem := alloc.Type().(*types.Pointer).Elem()
		if !elem.HasPtrData() {
			if !ap.NoScan {
				t.Fatalf("scalar alloc %v should have NoScan=true", alloc)
			}
			if ap.Heap {
				heapScalarNoScan++
			} else {
				stackScalar++
			}
		} else {
			if ap.NoScan {
				t.Fatalf("pointer-bearing alloc %v must have NoScan=false", alloc)
			}
			if ap.Heap {
				heapPtrScanned++
			}
		}
	}
	if stackScalar == 0 || heapScalarNoScan == 0 || heapPtrScanned == 0 {
		t.Fatalf("unexpected alloc plan counts: stackScalar=%d heapScalarNoScan=%d heapPtrScanned=%d",
			stackScalar, heapScalarNoScan, heapPtrScanned)
	}
}

func TestBoundsPlanO0VsO2AndWasm32WideIndexSafety(t *testing.T) {
	const src = `package test

func sumLoop(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}

func bceIdiom(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func maskArray(a *[8]int, i int) int {
	return a[i&7]
}

func sliceAndConvert(s []byte) (*[4]byte, []byte) {
	if len(s) >= 4 {
		return (*[4]byte)(s), s[:4]
	}
	return nil, nil
}

func wideIndex(s []int, i uint64) int {
	return s[i]
}

func wideIndexTruncatedGuard(s []int, i uint64) int {
	if int(i) >= 0 && int(i) < len(s) {
		return s[i]
	}
	return 0
}
`
	pkg, ctx := buildTestSSA(t, src)

	// sumLoop: checked at -O0, elided at -O2 with proof anchor.
	planO0, err := ssaopt.PlanFunction(pkg.Func("sumLoop"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O0, Verify: true})
	if err != nil {
		t.Fatalf("O0 sumLoop: %v", err)
	}
	planO2, err := ssaopt.PlanFunction(pkg.Func("sumLoop"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 sumLoop: %v", err)
	}
	countBounds := func(p *ssaopt.FunctionPlan) (checked, elided int) {
		for _, bp := range p.Bounds {
			if bp.CheckBounds {
				checked++
			} else {
				elided++
			}
		}
		return
	}
	if c, e := countBounds(planO0); c != 1 || e != 0 {
		t.Fatalf("O0 sumLoop bounds: got checked=%d elided=%d, want 1 and 0", c, e)
	}
	if c, e := countBounds(planO2); c != 0 || e != 1 {
		t.Fatalf("O2 sumLoop bounds: got checked=%d elided=%d, want 0 and 1", c, e)
	}

	// bceIdiom: 5 IndexAddr ops -> 1 checked (b[3]), 4 elided at -O2.
	planIdiom, err := ssaopt.PlanFunction(pkg.Func("bceIdiom"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 bceIdiom: %v", err)
	}
	if c, e := countBounds(planIdiom); c != 1 || e != 4 {
		t.Fatalf("O2 bceIdiom bounds: got checked=%d elided=%d, want 1 and 4", c, e)
	}

	// maskArray: a[i&7] on *[8]int elided at -O2.
	planMask, err := ssaopt.PlanFunction(pkg.Func("maskArray"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 maskArray: %v", err)
	}
	if c, e := countBounds(planMask); c != 0 || e != 1 {
		t.Fatalf("O2 maskArray bounds: got checked=%d elided=%d, want 0 and 1", c, e)
	}

	// sliceAndConvert: both SliceToArrayPointer and s[:4] elided at -O2.
	planConv, err := ssaopt.PlanFunction(pkg.Func("sliceAndConvert"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 sliceAndConvert: %v", err)
	}
	if c, e := countBounds(planConv); c != 0 || e != 2 {
		t.Fatalf("O2 sliceAndConvert bounds: got checked=%d elided=%d, want 0 and 2", c, e)
	}

	// Wasm32 wide index safety: uint64 dynamic & truncated-guard indices MUST remain checked!
	wasmPkg, wasmCtx := buildTestSSAForSpec(t, src, target.Spec{GOOS: "wasip1", GOARCH: "wasm"})
	for _, name := range []string{"wideIndex", "wideIndexTruncatedGuard"} {
		wp, err := ssaopt.PlanFunction(wasmPkg.Func(name), ssaopt.Config{TypeCtx: wasmCtx, OptLevel: optlevel.O2, Verify: true})
		if err != nil {
			t.Fatalf("wasm32 %s: %v", name, err)
		}
		if c, e := countBounds(wp); c != 1 || e != 0 {
			t.Fatalf("wasm32 %s must keep bounds check: checked=%d elided=%d", name, c, e)
		}
	}
}

func TestIfaceBoxAndConvPlan(t *testing.T) {
	const src = `package test

type Reader interface { Read() int }
type ReadCloser interface { Read() int; Close() }

type Buf struct { N int }
func (b Buf) Read() int { return b.N }
func (b Buf) Close() {}

type PtrBuf struct { P *int }

var sink any

func convAndAssert(n int) (Reader, int, bool, string, bool) {
	var rc ReadCloser = Buf{N: n}
	var rd Reader = rc
	var a any = n
	v := a.(int)
	s, ok := a.(string)
	_, okInt := a.(int)
	sink = PtrBuf{}
	return rd, v, okInt, s, ok
}
`
	pkg, ctx := buildTestSSA(t, src)
	fn := pkg.Func("convAndAssert")

	planO0, err := ssaopt.PlanFunction(fn, ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O0, Verify: true})
	if err != nil {
		t.Fatalf("O0 convAndAssert: %v", err)
	}
	if len(planO0.IfaceConvs) != 0 {
		t.Fatalf("O0 should not fold interface conversions/assertions, got %d", len(planO0.IfaceConvs))
	}

	planO2, err := ssaopt.PlanFunction(fn, ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 convAndAssert: %v", err)
	}
	var knownConc, staticSucc, staticFail int
	for _, ic := range planO2.IfaceConvs {
		switch ic.Kind {
		case ssaopt.IfaceConvKnownConcrete:
			knownConc++
		case ssaopt.IfaceConvStaticSuccess:
			staticSucc++
		case ssaopt.IfaceConvStaticFail:
			staticFail++
		}
	}
	if knownConc < 1 || staticSucc < 2 || staticFail < 1 {
		t.Fatalf("O2 iface convs: knownConc=%d staticSucc=%d staticFail=%d, want >=1, >=2, >=1",
			knownConc, staticSucc, staticFail)
	}

	var noScanBoxes, scannedBoxes int
	for _, ib := range planO2.IfaceBoxes {
		if ib.NoScan {
			noScanBoxes++
		} else {
			scannedBoxes++
		}
	}
	if noScanBoxes == 0 || scannedBoxes == 0 {
		t.Fatalf("expected both NoScan and scanned IfaceBoxes, got noScan=%d scanned=%d", noScanBoxes, scannedBoxes)
	}
}

func TestDevirtPlanTier1AndTier2RTA(t *testing.T) {
	const src = `package test

type Valuer interface { Val() int }
type Unique interface { Only() int }

type A struct { X int }
func (a *A) Val() int { return a.X }
func (a *A) Only() int { return a.X + 1 }

type B struct { Y int }
func (b *B) Val() int { return b.Y * 2 }

func tier1Direct(x int) int {
	var v Valuer = &A{X: x}
	return v.Val()
}

func tier1PhiMaybeNil(cond bool, x int) int {
	var v Valuer
	if cond {
		v = &A{X: x}
	}
	return v.Val()
}

func tier1PhiAmbiguous(cond bool, x int) int {
	var v Valuer
	if cond {
		v = &A{X: x}
	} else {
		v = &B{Y: x}
	}
	return v.Val()
}

func tier2Opaque(u Unique, v Valuer) int {
	return u.Only() + v.Val()
}

func entry() int {
	a := &A{X: 3}
	b := &B{Y: 4}
	return tier2Opaque(a, a) + tier2Opaque(a, b) + tier1Direct(1) + tier1PhiMaybeNil(true, 2) + tier1PhiAmbiguous(false, 5)
}
`
	pkg, ctx := buildTestSSA(t, src)

	// Tier 1 direct: not devirtualized at -O0, devirtualized at -O2 without nil check.
	pO0, err := ssaopt.PlanFunction(pkg.Func("tier1Direct"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O0, Verify: true})
	if err != nil {
		t.Fatalf("O0 tier1Direct: %v", err)
	}
	if len(pO0.Devirt) != 0 {
		t.Fatalf("O0 tier1Direct should not devirtualize")
	}
	pO2, err := ssaopt.PlanFunction(pkg.Func("tier1Direct"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 tier1Direct: %v", err)
	}
	if len(pO2.Devirt) != 1 {
		t.Fatalf("O2 tier1Direct expected 1 devirt, got %d", len(pO2.Devirt))
	}
	for _, dt := range pO2.Devirt {
		if dt.NeedNilCheck || dt.WholeProgram || dt.Receiver == nil || dt.Target.Name() != "Val" {
			t.Fatalf("unexpected tier1Direct devirt target: %+v", dt)
		}
	}

	// Tier 1 Phi with nil edge: devirtualized with NeedNilCheck=true and Receiver=nil.
	pPhiNil, err := ssaopt.PlanFunction(pkg.Func("tier1PhiMaybeNil"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 tier1PhiMaybeNil: %v", err)
	}
	if len(pPhiNil.Devirt) != 1 {
		t.Fatalf("O2 tier1PhiMaybeNil expected 1 devirt, got %d", len(pPhiNil.Devirt))
	}
	for _, dt := range pPhiNil.Devirt {
		if !dt.NeedNilCheck || dt.Receiver != nil || dt.Target.Name() != "Val" {
			t.Fatalf("tier1PhiMaybeNil must keep NeedNilCheck=true and Receiver=nil: %+v", dt)
		}
	}

	// Tier 1 Phi with A and B: must NOT devirtualize.
	pPhiAmb, err := ssaopt.PlanFunction(pkg.Func("tier1PhiAmbiguous"), ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("O2 tier1PhiAmbiguous: %v", err)
	}
	if len(pPhiAmb.Devirt) != 0 {
		t.Fatalf("O2 tier1PhiAmbiguous must not devirtualize, got %+v", pPhiAmb.Devirt)
	}

	// Tier 2 RTA: Unique has only *A -> devirtualized; Valuer has *A and *B -> not devirtualized.
	rtaRes := rta.Analyze([]*ssa.Function{pkg.Func("entry")}, false)
	pTier2Off, err := ssaopt.PlanFunction(pkg.Func("tier2Opaque"), ssaopt.Config{
		TypeCtx:      ctx,
		OptLevel:     optlevel.O2,
		WholeProgram: false,
		RTA:          rtaRes,
		Verify:       true,
	})
	if err != nil {
		t.Fatalf("tier2 off: %v", err)
	}
	if len(pTier2Off.Devirt) != 0 {
		t.Fatalf("tier2Opaque with WholeProgram=false must not devirtualize")
	}
	pTier2On, err := ssaopt.PlanFunction(pkg.Func("tier2Opaque"), ssaopt.Config{
		TypeCtx:      ctx,
		OptLevel:     optlevel.O2,
		WholeProgram: true,
		RTA:          rtaRes,
		Verify:       true,
	})
	if err != nil {
		t.Fatalf("tier2 on: %v", err)
	}
	if len(pTier2On.Devirt) != 1 {
		t.Fatalf("tier2Opaque with WholeProgram=true expected 1 devirt (Unique.Only), got %d", len(pTier2On.Devirt))
	}
	for _, dt := range pTier2On.Devirt {
		if !dt.WholeProgram || !dt.NeedNilCheck || dt.Target.Name() != "Only" {
			t.Fatalf("unexpected tier2 devirt target: %+v", dt)
		}
	}
}

func TestSafepointPassInFunctionPlan(t *testing.T) {
	const src = `package test

func spin(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	return sum
}
`
	pkg, ctx := buildTestSSA(t, src)
	plan, err := ssaopt.PlanFunction(pkg.Func("spin"), ssaopt.Config{
		TypeCtx:               ctx,
		OptLevel:              optlevel.O2,
		CooperativeSafepoints: true,
		Verify:                true,
	})
	if err != nil {
		t.Fatalf("PlanFunction: %v", err)
	}
	if len(plan.SafepointBackedges) == 0 {
		t.Fatalf("expected cooperative safepoint backedges for loop function")
	}
}

func TestNoInterfaceMethodNotFoldedToStaticSuccess(t *testing.T) {
	const src = `package test

type Hidden interface { Bad() int }
type Visible interface { Good() int }

type E struct{}

//go:nointerface
func (E) Bad() int { return 1 }

func (E) Good() int { return 2 }

func check() (bool, bool) {
	var a any = E{}
	_, okBad := a.(Hidden)
	_, okGood := a.(Visible)
	return okBad, okGood
}
`
	pkg, ctx := buildTestSSA(t, src)
	plan, err := ssaopt.PlanFunction(pkg.Func("check"), ssaopt.Config{
		TypeCtx:  ctx,
		OptLevel: optlevel.O2,
		Verify:   true,
	})
	if err != nil {
		t.Fatalf("PlanFunction: %v", err)
	}
	var goodStatic, badStatic int
	for ins, ic := range plan.IfaceConvs {
		ta, ok := ins.(*ssa.TypeAssert)
		if !ok || ic.Kind != ssaopt.IfaceConvStaticSuccess {
			continue
		}
		switch ta.AssertedType.String() {
		case "example.com/test.Hidden":
			badStatic++
		case "example.com/test.Visible":
			goodStatic++
		}
	}
	if badStatic != 0 {
		t.Fatalf("//go:nointerface method assertion must not fold to StaticSuccess, got %d", badStatic)
	}
	if goodStatic != 1 {
		t.Fatalf("normal interface assertion should fold to StaticSuccess, got %d", goodStatic)
	}
}

func TestVerifyPlanRejectsInvalidM3Plans(t *testing.T) {
	const src = `package test

type Valuer interface { Val() int }
type A struct { P *int }
func (a *A) Val() int { return 1 }

func sample(cond bool, s []int, a *A) int {
	var v Valuer = a
	if cond {
		_ = s[0]
		return v.Val()
	}
	return s[0]
}
`
	pkg, ctx := buildTestSSA(t, src)
	fn := pkg.Func("sample")
	plan, err := ssaopt.PlanFunction(fn, ssaopt.Config{TypeCtx: ctx, OptLevel: optlevel.O2, Verify: true})
	if err != nil {
		t.Fatalf("valid plan failed Verify: %v", err)
	}

	// 1. Non-dominating BoundsPlan ProofAnchor must fail VerifyPlan.
	var idxInstrs []ssa.Instruction
	for ins := range plan.Bounds {
		idxInstrs = append(idxInstrs, ins)
	}
	if len(idxInstrs) < 2 {
		t.Fatalf("expected >=2 bounds instructions, got %d", len(idxInstrs))
	}
	badBounds := &ssaopt.FunctionPlan{
		Fn: fn,
		Bounds: map[ssa.Instruction]ssaopt.BoundsPlan{
			idxInstrs[0]: {CheckBounds: false, ProofAnchor: idxInstrs[1]},
		},
	}
	if err := ssaopt.VerifyPlan(fn, badBounds); err == nil || !strings.Contains(err.Error(), "dominate") {
		t.Fatalf("expected VerifyPlan to reject non-dominating BoundsPlan anchor, got %v", err)
	}

	// 2. IfaceBoxPlan.NoScan on pointer-bearing type (*A) must fail VerifyPlan.
	for mi := range plan.IfaceBoxes {
		badBox := &ssaopt.FunctionPlan{
			Fn:         fn,
			IfaceBoxes: map[*ssa.MakeInterface]ssaopt.IfaceBoxPlan{mi: {Kind: ssaopt.IfaceBoxDirect, NoScan: true}},
		}
		if err := ssaopt.VerifyPlan(fn, badBox); err == nil || !strings.Contains(err.Error(), "NoScan") {
			t.Fatalf("expected VerifyPlan to reject IfaceBoxPlan.NoScan on pointer type, got %v", err)
		}
		break
	}

	// 3. Devirt with non-dominating ProofAnchor must fail VerifyPlan.
	for call, dt := range plan.Devirt {
		badDevirt := &ssaopt.FunctionPlan{
			Fn: fn,
			Devirt: map[*ssa.Call]ssaopt.DevirtTarget{
				call: {Target: dt.Target, ProofAnchor: idxInstrs[1]},
			},
		}
		if err := ssaopt.VerifyPlan(fn, badDevirt); err == nil {
			// Try the other index instruction in case idxInstrs[1] happened to dominate call.
			badDevirt.Devirt[call] = ssaopt.DevirtTarget{Target: dt.Target, ProofAnchor: idxInstrs[0]}
			if err2 := ssaopt.VerifyPlan(fn, badDevirt); err2 == nil {
				t.Fatalf("expected VerifyPlan to reject non-dominating Devirt proof anchor")
			}
		}
		break
	}
}

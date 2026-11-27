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

	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func buildTestSSA(t *testing.T, src string) (*ssa.Package, *types.Context) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
	})
	pkg, _, err := ssautil.BuildPackage(&types.Config{Context: ctx, Sizes: ctx}, fset, types.NewPackage("example.com/test", "test"), []*ast.File{file}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatalf("build SSA: %v", err)
	}
	return pkg, ctx
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

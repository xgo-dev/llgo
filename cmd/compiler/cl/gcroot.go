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

package cl

import (
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func (p *context) prepareGCRoots(fn *ssa.Function, hasClosureContext bool) {
	p.gcRoots = nil
	p.gcClosureRoot = llvmgen.Nil
	if !p.prog.GCRootsEnabled() || fn == nil || len(fn.Blocks) == 0 {
		return
	}

	plan := p.optPlan
	if plan == nil || !plan.GCRootPlanned {
		// Defensive fallback if an external caller bypassed optPlanFor or toggled
		// GCRoots after ProgramPlan creation.
		var patchTypes map[string]*types.Package
		if len(p.patches) > 0 {
			patchTypes = make(map[string]*types.Package, len(p.patches))
			for path, patch := range p.patches {
				if patch.Types != nil {
					patchTypes[path] = patch.Types
				}
			}
		}
		plan, _ = ssaopt.PlanFunction(fn, ssaopt.Config{
			TypeCtx:               p.prog.TypeContext(),
			OptLevel:              p.options.OptLevel,
			GCRoots:               true,
			DeadcodeDrop:          p.prog.DeadcodeDropEnabled(),
			CooperativeSafepoints: p.prog.CooperativeSafepointsEnabled(),
			Escape:                p.options.EscapeAnalysis,
			PatchTypes:            patchTypes,
			RuntimePkg:            p.prog.RuntimePkg(),
		})
	}
	if plan == nil {
		return
	}

	counts := plan.GCRoots
	total := 0
	for _, param := range fn.Params {
		total += counts[param]
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				total += counts[value]
			}
		}
	}
	hasClosureRoot := hasClosureContext && plan.GCClosureRoot
	if hasClosureRoot {
		total++
	}
	allSlots := p.fn.NewGCRoots(total)
	next := 0
	roots := make(map[ssa.Value][]llvmgen.Expr, len(counts))
	assign := func(value ssa.Value) {
		if n := counts[value]; n != 0 {
			roots[value] = allSlots[next : next+n]
			next += n
		}
	}
	for _, param := range fn.Params {
		assign(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				assign(value)
			}
		}
	}
	p.gcRoots = roots
	if hasClosureRoot {
		p.gcClosureRoot = allSlots[next]
	}
}

func mayContainGCRoot(typ types.Type) bool {
	return ssaopt.MayContainGCRoot(typ)
}

func (p *context) initGCRoots(b llvmgen.Builder, fn *ssa.Function) {
	if len(p.gcRoots) == 0 && p.gcClosureRoot.IsNil() {
		return
	}
	b.SetBlockEx(p.fn.Block(0), llvmgen.AtEnd, true)
	for i, param := range fn.Params {
		if _, ok := p.gcRoots[param]; ok {
			p.publishGCRoot(b, param, b.Param(i))
		}
	}
	if !p.gcClosureRoot.IsNil() {
		b.SetGCRoot(p.gcClosureRoot, p.fn.ClosureContextParam())
	}
}

func functionHasGCSafepoint(fn *ssa.Function) bool {
	return ssaopt.FunctionHasGCSafepoint(fn)
}

func gcSafepoint(instr ssa.Instruction) bool {
	return ssaopt.GCSafepoint(instr)
}

func gcConversionSafepoint(src, dst types.Type) bool {
	return ssaopt.GCConversionSafepoint(src, dst)
}

func basicKind(typ types.Type) types.BasicKind {
	if basic, ok := types.Unalias(typ).Underlying().(*types.Basic); ok {
		return basic.Kind()
	}
	return types.Invalid
}

func (p *context) publishGCRoot(b llvmgen.Builder, value ssa.Value, expr llvmgen.Expr) {
	slots, ok := p.gcRoots[value]
	if !ok || expr.IsNil() {
		return
	}
	if basicKind(value.Type()) == types.Uintptr {
		// prepareGCRoots reserves exactly one slot for pragma-designated uintptr
		// values even though their ordinary GC root count is zero.
		b.SetGCRoot(slots[0], expr)
		return
	}
	roots := b.GCRootPointers(expr)
	if len(roots) != len(slots) {
		panic("cl: inconsistent GC root layout")
	}
	for i, root := range roots {
		b.SetGCRoot(slots[i], root)
	}
}

func gcRootOwner(value ssa.Value) ssa.Value {
	return ssaopt.GCRootOwner(value)
}

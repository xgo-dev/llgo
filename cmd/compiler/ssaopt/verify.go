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

package ssaopt

import (
	"fmt"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// VerifyFunctionAndPlan verifies both the underlying SSA IR (reusing
// ssa.VerifyFunction for CFG/Phi/SSA dominance invariants) and the
// ssaopt-specific FunctionPlan invariants.
func VerifyFunctionAndPlan(fn *ssa.Function, plan *FunctionPlan) error {
	if err := ssa.VerifyFunction(fn); err != nil {
		return err
	}
	return VerifyPlan(fn, plan)
}

// VerifyPlan checks plan-specific invariants for fn:
//   - every plan key and proof anchor belongs to fn;
//   - every elided check has an explicit justification and, when an instruction
//     anchor is provided, that anchor strictly dominates the checked instruction;
//   - every AllocPlan with NoScan==true targets a pointer-free element type and
//     is not a //go:uintptrescapes backing array;
//   - every devirtualized call target is non-nil and signature-compatible.
func VerifyPlan(fn *ssa.Function, plan *FunctionPlan) error {
	if fn == nil {
		return fmt.Errorf("ssaopt: nil function")
	}
	if plan == nil {
		return fmt.Errorf("ssaopt: nil FunctionPlan for %s", fn.String())
	}
	if plan.Fn != nil && plan.Fn != fn {
		return fmt.Errorf("ssaopt: FunctionPlan belongs to %s, not %s", plan.Fn.String(), fn.String())
	}
	if len(fn.Blocks) == 0 {
		if len(plan.NilChecks) > 0 || len(plan.Allocs) > 0 || len(plan.Bounds) > 0 ||
			len(plan.IfaceBoxes) > 0 || len(plan.IfaceConvs) > 0 || len(plan.Devirt) > 0 {
			return fmt.Errorf("ssaopt: %s has no body blocks but non-empty FunctionPlan", fn.String())
		}
		return nil
	}

	type instrPos struct {
		blk *ssa.BasicBlock
		idx int
	}
	positions := make(map[ssa.Instruction]instrPos, 64)
	for _, blk := range fn.Blocks {
		for idx, ins := range blk.Instrs {
			positions[ins] = instrPos{blk: blk, idx: idx}
		}
	}

	hasDomTree := len(fn.Blocks) > 0 && fn.Blocks[0].Dominates(fn.Blocks[0])
	strictlyDominates := func(proof, target ssa.Instruction) bool {
		pp, okP := positions[proof]
		tp, okT := positions[target]
		if !okP || !okT {
			return false
		}
		if pp.blk == tp.blk {
			return pp.idx < tp.idx
		}
		if !hasDomTree {
			return false
		}
		return pp.blk.Dominates(tp.blk)
	}

	for ins, nc := range plan.NilChecks {
		if _, ok := positions[ins]; !ok {
			return fmt.Errorf("ssaopt: %s: NilCheckPlan key %T does not belong to function", fn.String(), ins)
		}
		switch ins.(type) {
		case *ssa.Call, *ssa.Defer, *ssa.Go, *ssa.MakeClosure:
		default:
			return fmt.Errorf("ssaopt: %s: NilCheckPlan attached to unsupported instruction %T", fn.String(), ins)
		}
		if nc.Elided {
			if nc.ProofKind == NilProofNone {
				return fmt.Errorf("ssaopt: %s: elided NilCheckPlan on %T lacks ProofKind", fn.String(), ins)
			}
			if nc.NeedBaseCheck || nc.NeedAddrCheck {
				return fmt.Errorf("ssaopt: %s: NilCheckPlan on %T marked both Elided and active check", fn.String(), ins)
			}
			if nc.ProofAnchor != nil && !strictlyDominates(nc.ProofAnchor, ins) {
				return fmt.Errorf("ssaopt: %s: NilCheckPlan proof anchor %T does not dominate %T", fn.String(), nc.ProofAnchor, ins)
			}
		}
	}

	for load := range plan.MethodNilDerefChecks {
		if _, ok := positions[load]; !ok {
			return fmt.Errorf("ssaopt: %s: MethodNilDerefChecks key %v does not belong to function", fn.String(), load)
		}
	}
	for load := range plan.RecvNilDerefChecks {
		if _, ok := positions[load]; !ok {
			return fmt.Errorf("ssaopt: %s: RecvNilDerefChecks key %v does not belong to function", fn.String(), load)
		}
	}

	for alloc, ap := range plan.Allocs {
		if _, ok := positions[alloc]; !ok {
			return fmt.Errorf("ssaopt: %s: AllocPlan key %v does not belong to function", fn.String(), alloc)
		}
		ptr, ok := alloc.Type().(*types.Pointer)
		if !ok || ptr.Elem() == nil {
			return fmt.Errorf("ssaopt: %s: AllocPlan on non-pointer alloc %v", fn.String(), alloc)
		}
		if ap.NoScan {
			if ptr.Elem().HasPtrData() {
				return fmt.Errorf("ssaopt: %s: AllocPlan.NoScan set on pointer-bearing element type %s", fn.String(), ptr.Elem())
			}
			if ap.UintptrEscapes {
				return fmt.Errorf("ssaopt: %s: AllocPlan.NoScan set on //go:uintptrescapes backing array %s", fn.String(), alloc.Name())
			}
		}
	}

	for ins, bp := range plan.Bounds {
		if _, ok := positions[ins]; !ok {
			return fmt.Errorf("ssaopt: %s: BoundsPlan key %T does not belong to function", fn.String(), ins)
		}
		switch ins.(type) {
		case *ssa.IndexAddr, *ssa.Index, *ssa.Slice, *ssa.SliceToArrayPointer:
		default:
			return fmt.Errorf("ssaopt: %s: BoundsPlan attached to unsupported instruction %T", fn.String(), ins)
		}
		if !bp.CheckBounds && bp.ProofAnchor != nil && !strictlyDominates(bp.ProofAnchor, ins) {
			return fmt.Errorf("ssaopt: %s: BoundsPlan proof anchor %T does not dominate %T", fn.String(), bp.ProofAnchor, ins)
		}
	}

	for mi, ib := range plan.IfaceBoxes {
		if _, ok := positions[mi]; !ok {
			return fmt.Errorf("ssaopt: %s: IfaceBoxPlan key %v does not belong to function", fn.String(), mi)
		}
		if ib.NoScan && mi.X != nil && mi.X.Type().HasPtrData() {
			return fmt.Errorf("ssaopt: %s: IfaceBoxPlan.NoScan set on pointer-bearing boxed type %s", fn.String(), mi.X.Type())
		}
	}

	for ins, ic := range plan.IfaceConvs {
		if _, ok := positions[ins]; !ok {
			return fmt.Errorf("ssaopt: %s: IfaceConvPlan key %T does not belong to function", fn.String(), ins)
		}
		switch ins.(type) {
		case *ssa.ChangeInterface, *ssa.TypeAssert:
		default:
			return fmt.Errorf("ssaopt: %s: IfaceConvPlan attached to unsupported instruction %T", fn.String(), ins)
		}
		if ic.ProofAnchor != nil && !strictlyDominates(ic.ProofAnchor, ins) {
			return fmt.Errorf("ssaopt: %s: IfaceConvPlan proof anchor %T does not dominate %T", fn.String(), ic.ProofAnchor, ins)
		}
	}

	for call, dt := range plan.Devirt {
		if _, ok := positions[call]; !ok {
			return fmt.Errorf("ssaopt: %s: Devirt key %v does not belong to function", fn.String(), call)
		}
		if !call.Call.IsInvoke() {
			return fmt.Errorf("ssaopt: %s: Devirt attached to non-invoke call %v", fn.String(), call)
		}
		if dt.Target == nil {
			return fmt.Errorf("ssaopt: %s: Devirt target is nil for %v", fn.String(), call)
		}
		if dt.Target.Signature == nil || dt.Target.Signature.Recv() == nil {
			return fmt.Errorf("ssaopt: %s: Devirt target %s lacks method receiver for %v", fn.String(), dt.Target.String(), call)
		}
		if dt.Target.Signature.Params().Len() != len(call.Call.Args) {
			return fmt.Errorf("ssaopt: %s: Devirt target %s param count mismatch for %v", fn.String(), dt.Target.String(), call)
		}
		if dt.Receiver != nil {
			if recvIns, ok := dt.Receiver.(ssa.Instruction); ok && !strictlyDominates(recvIns, call) {
				return fmt.Errorf("ssaopt: %s: Devirt receiver %T does not dominate %v", fn.String(), recvIns, call)
			}
		}
		if dt.ProofAnchor != nil && !strictlyDominates(dt.ProofAnchor, call) {
			return fmt.Errorf("ssaopt: %s: Devirt proof anchor %T does not dominate %v", fn.String(), dt.ProofAnchor, call)
		}
	}

	for ins := range plan.SafepointBackedges {
		if _, ok := positions[ins]; !ok {
			return fmt.Errorf("ssaopt: %s: SafepointBackedges key %T does not belong to function", fn.String(), ins)
		}
	}

	if len(plan.GCRoots) > 0 {
		params := make(map[ssa.Value]struct{}, len(fn.Params))
		for _, p := range fn.Params {
			params[p] = struct{}{}
		}
		for val, slots := range plan.GCRoots {
			if slots <= 0 {
				return fmt.Errorf("ssaopt: %s: GCRoots[%v] has non-positive slot count %d", fn.String(), val, slots)
			}
			if _, isParam := params[val]; isParam {
				continue
			}
			ins, ok := val.(ssa.Instruction)
			if !ok {
				return fmt.Errorf("ssaopt: %s: GCRoots key %T is neither a parameter nor an instruction", fn.String(), val)
			}
			if _, ok := positions[ins]; !ok {
				return fmt.Errorf("ssaopt: %s: GCRoots key %T does not belong to function", fn.String(), val)
			}
		}
	}

	return nil
}

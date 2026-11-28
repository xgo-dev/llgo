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
	"go/constant"
	"math"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// IfacePass plans interface boxing placement/GC-scanning policies and folds
// interface conversions / type assertions whose concrete type is statically
// known along the dominator tree.
//
// At -O0, only IfaceBoxes metadata is populated (preserving all runtime
// ChangeInterface and TypeAssert behavior). At >= -O1, dominating MakeInterface
// origins allow ChangeInterface to reuse a compile-time static itab and allow
// TypeAssert to resolve statically without dynamic type checks or runtime
// Implements calls.
type IfacePass struct{}

func (IfacePass) Name() string { return "iface" }

func (IfacePass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}

	is32Bits := false
	if cfg.TypeCtx != nil && cfg.TypeCtx.Spec().WordSize == 4 {
		is32Bits = true
	}

	instrPos := make(map[ssa.Instruction]struct {
		blk *ssa.BasicBlock
		idx int
	}, 64)
	for _, blk := range fn.Blocks {
		for idx, ins := range blk.Instrs {
			instrPos[ins] = struct {
				blk *ssa.BasicBlock
				idx int
			}{blk: blk, idx: idx}
		}
	}
	hasDom := fn.Blocks[0].Dominates(fn.Blocks[0])
	strictlyDominates := func(proof, target ssa.Instruction) bool {
		pp, okP := instrPos[proof]
		tp, okT := instrPos[target]
		if !okP || !okT {
			return false
		}
		if pp.blk == tp.blk {
			return pp.idx < tp.idx
		}
		return hasDom && pp.blk.Dominates(tp.blk)
	}

	var boxes map[*ssa.MakeInterface]IfaceBoxPlan
	var convs map[ssa.Instruction]IfaceConvPlan

	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.MakeInterface:
				bp := planMakeInterface(ins, cfg, is32Bits)
				if boxes == nil {
					boxes = make(map[*ssa.MakeInterface]IfaceBoxPlan)
				}
				boxes[ins] = bp

			case *ssa.ChangeInterface:
				if !cfg.Optimizing() {
					continue
				}
				mi, ok := traceConcreteMakeInterface(ins.X)
				if !ok || !strictlyDominates(mi, ins) {
					continue
				}
				concrete := mi.X.Type()
				dstIntf, ok := types.Unalias(ins.Type()).Underlying().(*types.Interface)
				if !ok || dstIntf.Empty() || !implementsRuntimeInterface(concrete, dstIntf) {
					continue
				}
				if convs == nil {
					convs = make(map[ssa.Instruction]IfaceConvPlan)
				}
				convs[ins] = IfaceConvPlan{
					Kind:        IfaceConvKnownConcrete,
					KnownType:   concrete,
					KnownValue:  mi.X,
					ProofAnchor: mi,
				}

			case *ssa.TypeAssert:
				if !cfg.Optimizing() {
					continue
				}
				mi, ok := traceConcreteMakeInterface(ins.X)
				if !ok || !strictlyDominates(mi, ins) {
					continue
				}
				concrete := mi.X.Type()
				asserted := ins.AssertedType
				if asserted == nil {
					continue
				}
				if dstIntf, isIntf := types.Unalias(asserted).Underlying().(*types.Interface); isIntf {
					if implementsRuntimeInterface(concrete, dstIntf) {
						if convs == nil {
							convs = make(map[ssa.Instruction]IfaceConvPlan)
						}
						convs[ins] = IfaceConvPlan{
							Kind:        IfaceConvStaticSuccess,
							KnownType:   concrete,
							KnownValue:  mi.X,
							ProofAnchor: mi,
						}
					} else if ins.CommaOk && !types.AssignableTo(concrete, dstIntf) {
						if convs == nil {
							convs = make(map[ssa.Instruction]IfaceConvPlan)
						}
						convs[ins] = IfaceConvPlan{
							Kind:        IfaceConvStaticFail,
							KnownType:   concrete,
							ProofAnchor: mi,
						}
					}
					continue
				}
				// Concrete asserted type: skip function/signature types so
				// runtime MatchesClosure semantics for named vs unnamed
				// closures remain authoritative.
				if _, isSig := types.Unalias(asserted).Underlying().(*types.Signature); isSig {
					continue
				}
				if _, isSig := types.Unalias(concrete).Underlying().(*types.Signature); isSig {
					continue
				}
				if types.Identical(types.Unalias(concrete), types.Unalias(asserted)) {
					if convs == nil {
						convs = make(map[ssa.Instruction]IfaceConvPlan)
					}
					convs[ins] = IfaceConvPlan{
						Kind:        IfaceConvStaticSuccess,
						KnownType:   concrete,
						KnownValue:  mi.X,
						ProofAnchor: mi,
					}
				} else if ins.CommaOk {
					if convs == nil {
						convs = make(map[ssa.Instruction]IfaceConvPlan)
					}
					convs[ins] = IfaceConvPlan{
						Kind:        IfaceConvStaticFail,
						KnownType:   concrete,
						ProofAnchor: mi,
					}
				}
			}
		}
	}

	plan.IfaceBoxes = boxes
	plan.IfaceConvs = convs
	return nil
}

func planMakeInterface(mi *ssa.MakeInterface, cfg Config, is32Bits bool) IfaceBoxPlan {
	if mi == nil || mi.X == nil || isUntypedNilVal(mi.X) {
		return IfaceBoxPlan{Kind: IfaceBoxDirect, NoScan: true}
	}
	elem := mi.X.Type()
	patched := patchEscapeType(elem, cfg.PatchTypes)
	noScan := !elem.HasPtrData() && !patched.HasPtrData()

	if !InterfaceIndirect(patched, cfg.TypeCtx, is32Bits) {
		return IfaceBoxPlan{Kind: IfaceBoxDirect, NoScan: noScan}
	}
	if isStaticBoxableConst(mi.X) {
		return IfaceBoxPlan{Kind: IfaceBoxStatic, NoScan: noScan}
	}
	if !FreshNilAppendAllocation(mi) && cfg.Escape != nil {
		if decision, ok := cfg.Escape.LookupDecision(mi); ok && !decision.Heap {
			return IfaceBoxPlan{Kind: IfaceBoxStack, NoScan: noScan}
		}
	}
	return IfaceBoxPlan{Kind: IfaceBoxHeap, NoScan: noScan}
}

func isStaticBoxableConst(v ssa.Value) bool {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil {
		return false
	}
	if c.Value.Kind() == constant.Float {
		f, _ := constant.Float64Val(c.Value)
		if math.IsNaN(f) {
			return false
		}
	}
	return true
}

func isUntypedNilVal(v ssa.Value) bool {
	c, ok := v.(*ssa.Const)
	if !ok || !c.IsNil() {
		return false
	}
	b, ok := c.Type().(*types.Basic)
	return ok && b.Kind() == types.UntypedNil
}

// implementsRuntimeInterface reports whether concrete satisfies dstIntf both
// in the Go type system and in LLGo's runtime ABI method table (excluding
// methods marked with //go:nointerface).
func implementsRuntimeInterface(concrete types.Type, dstIntf *types.Interface) bool {
	if concrete == nil || dstIntf == nil || !types.AssignableTo(concrete, dstIntf) {
		return false
	}
	if dstIntf.Empty() {
		return true
	}
	mset := types.NewMethodSet(concrete)
	for i := 0; i < dstIntf.NumMethods(); i++ {
		im := dstIntf.Method(i)
		sel := mset.Lookup(im.Pkg(), im.Name())
		if sel == nil {
			return false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.NoInterface() {
			return false
		}
	}
	return true
}

// traceConcreteMakeInterface traces an interface SSA value through zero or more
// ChangeInterface conversions (or non-comma-ok interface TypeAsserts) back to a
// non-nil *ssa.MakeInterface in the same function.
func traceConcreteMakeInterface(v ssa.Value) (*ssa.MakeInterface, bool) {
	seen := make(map[ssa.Value]bool, 4)
	for v != nil && !seen[v] {
		seen[v] = true
		switch x := v.(type) {
		case *ssa.MakeInterface:
			if x.X == nil || isUntypedNilVal(x.X) {
				return nil, false
			}
			// Never fold an interface wrapping another interface (invalid SSA,
			// defensive check).
			if _, isIntf := types.Unalias(x.X.Type()).Underlying().(*types.Interface); isIntf {
				return nil, false
			}
			return x, true
		case *ssa.ChangeInterface:
			v = x.X
		case *ssa.TypeAssert:
			if x.CommaOk {
				return nil, false
			}
			v = x.X
		default:
			return nil, false
		}
	}
	return nil, false
}

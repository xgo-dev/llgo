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
	"github.com/xgo-dev/llgo/callgraph/rta"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// DevirtPass resolves interface method invocations (*ssa.Call with
// Call.IsInvoke()) to concrete direct calls when safe.
//
// Two tiers are supported (both enabled only at >= -O1):
//   - Tier 1 (intra-procedural, always sound): propagates concrete types from
//     dominating MakeInterface, ChangeInterface, non-comma-ok TypeAssert, and
//     single-concrete-type Phi nodes.
//   - Tier 2 (whole-program, opt-in via Config.WholeProgram && Config.RTA != nil):
//     uses RTA RuntimeTypes to detect interfaces with a unique concrete
//     implementation across the closed program, retaining a nil-interface check
//     before the direct call.
type DevirtPass struct{}

func (DevirtPass) Name() string { return "devirt" }

func (DevirtPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if !cfg.Optimizing() || fn == nil || len(fn.Blocks) == 0 || fn.Prog == nil {
		return nil
	}
	if IsMethodReceiverWrapper(fn) {
		return nil
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

	var devirt map[*ssa.Call]DevirtTarget

	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			call, ok := ins.(*ssa.Call)
			if !ok || !call.Call.IsInvoke() {
				continue
			}
			mthd := call.Call.Method
			if mthd == nil {
				continue
			}

			// Tier 1: intra-procedural concrete type propagation.
			if concrete, recvVal, needNil, anchor, ok := resolveIntraConcreteType(call.Call.Value); ok {
				if anchor != nil && !strictlyDominates(anchor, call) {
					anchor = nil
				}
				// If Receiver is an instruction, require strict dominance too.
				if recvIns, isIns := recvVal.(ssa.Instruction); isIns && !strictlyDominates(recvIns, call) {
					recvVal = nil
				}
				if targetFn := lookupDevirtCallee(fn.Prog, concrete, mthd, call); targetFn != nil {
					if devirt == nil {
						devirt = make(map[*ssa.Call]DevirtTarget)
					}
					devirt[call] = DevirtTarget{
						Target:       targetFn,
						Receiver:     recvVal,
						NeedNilCheck: needNil,
						WholeProgram: false,
						ProofAnchor:  anchor,
					}
					continue
				}
			}

			// Tier 2: optional whole-program RTA unique implementation.
			if cfg.WholeProgram && cfg.RTA != nil {
				if concrete, ok := uniqueRTAImpl(cfg.RTA, call.Call.Value.Type()); ok {
					if targetFn := lookupDevirtCallee(fn.Prog, concrete, mthd, call); targetFn != nil {
						if devirt == nil {
							devirt = make(map[*ssa.Call]DevirtTarget)
						}
						devirt[call] = DevirtTarget{
							Target:       targetFn,
							Receiver:     nil,
							NeedNilCheck: true,
							WholeProgram: true,
						}
					}
				}
			}
		}
	}

	plan.Devirt = devirt
	return nil
}

// resolveIntraConcreteType propagates a concrete Go type (and optionally exact
// receiver value) through MakeInterface, ChangeInterface, TypeAssert, and Phi.
func resolveIntraConcreteType(v ssa.Value) (concrete types.Type, recv ssa.Value, needNil bool, anchor ssa.Instruction, ok bool) {
	if mi, found := traceConcreteMakeInterface(v); found {
		return mi.X.Type(), mi.X, false, mi, true
	}

	// Handle Phi where every non-nil edge resolves to the exact same concrete
	// type. Any nil-interface edge marks needNil=true so runtime nil-interface
	// panic semantics are preserved.
	phi, isPhi := v.(*ssa.Phi)
	if !isPhi || len(phi.Edges) == 0 {
		return nil, nil, false, nil, false
	}
	seen := make(map[ssa.Value]bool, 8)
	var commonType types.Type
	mayBeNil := false
	var walk func(ssa.Value) bool
	walk = func(cur ssa.Value) bool {
		if cur == nil {
			return false
		}
		if seen[cur] {
			return true
		}
		seen[cur] = true
		switch x := cur.(type) {
		case *ssa.Const:
			if x.IsNil() {
				mayBeNil = true
				return true
			}
			return false
		case *ssa.MakeInterface:
			if x.X == nil || isUntypedNilVal(x.X) {
				mayBeNil = true
				return true
			}
			t := x.X.Type()
			if _, isIntf := types.Unalias(t).Underlying().(*types.Interface); isIntf {
				return false
			}
			if commonType == nil {
				commonType = t
			} else if !types.Identical(types.Unalias(commonType), types.Unalias(t)) {
				return false
			}
			return true
		case *ssa.ChangeInterface:
			return walk(x.X)
		case *ssa.TypeAssert:
			if x.CommaOk {
				return false
			}
			return walk(x.X)
		case *ssa.Phi:
			for _, edge := range x.Edges {
				if !walk(edge) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
	if !walk(phi) || commonType == nil {
		return nil, nil, false, nil, false
	}
	return commonType, nil, mayBeNil, nil, true
}

func uniqueRTAImpl(res *rta.Result, ifaceTyp types.Type) (types.Type, bool) {
	if res == nil || ifaceTyp == nil {
		return nil, false
	}
	rawIntf, ok := types.Unalias(ifaceTyp).Underlying().(*types.Interface)
	if !ok || rawIntf.Empty() {
		return nil, false
	}
	var unique types.Type
	ambiguous := false
	res.RuntimeTypes.Iterate(func(key types.Type, _ any) {
		if ambiguous || key == nil {
			return
		}
		if _, isIntf := types.Unalias(key).Underlying().(*types.Interface); isIntf {
			return
		}
		if !implementsRuntimeInterface(key, rawIntf) {
			return
		}
		if unique == nil {
			unique = key
		} else if !types.Identical(types.Unalias(unique), types.Unalias(key)) {
			ambiguous = true
		}
	})
	if ambiguous || unique == nil {
		return nil, false
	}
	return unique, true
}

func lookupDevirtCallee(prog *ssa.Program, concrete types.Type, mthd *types.Func, call *ssa.Call) *ssa.Function {
	if prog == nil || concrete == nil || mthd == nil || call == nil {
		return nil
	}
	mset := prog.MethodSets.MethodSet(concrete)
	if mset == nil {
		return nil
	}
	sel := mset.Lookup(mthd.Pkg(), mthd.Name())
	if sel == nil {
		return nil
	}
	selFunc, ok := sel.Obj().(*types.Func)
	if !ok || selFunc.NoInterface() {
		return nil
	}
	target := prog.MethodValue(sel)
	if target == nil || target.Signature == nil {
		return nil
	}
	recv := target.Signature.Recv()
	if recv == nil {
		return nil
	}
	if !types.Identical(types.Unalias(recv.Type()), types.Unalias(concrete)) {
		return nil
	}
	if obj, ok := target.Object().(*types.Func); ok && obj.NoInterface() {
		return nil
	}
	callSig := call.Call.Signature()
	if callSig == nil {
		return nil
	}
	if target.Signature.Params().Len() != len(call.Call.Args) {
		return nil
	}
	if target.Signature.Results().Len() != callSig.Results().Len() {
		return nil
	}
	return target
}

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

// Package gcrootplan computes the Go SSA values that must remain visible to a
// tracing collector while a function is stopped at a safepoint.
package gcrootplan

import "golang.org/x/tools/go/ssa"

// Plan returns values accepted by needsRoot that are live immediately before
// an instruction accepted by isSafepoint.
func Plan(fn *ssa.Function, needsRoot func(ssa.Value) bool, isSafepoint func(ssa.Instruction) bool) map[ssa.Value]struct{} {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}

	blocks := make([]blockInfo, len(fn.Blocks))
	for _, block := range fn.Blocks {
		info := &blocks[block.Index]
		info.def = make(valueSet)
		info.use = make(valueSet)
		info.phiDef = make(valueSet)
		info.edgeUse = make(map[int]valueSet)

		for _, instr := range block.Instrs {
			if phi, ok := instr.(*ssa.Phi); ok {
				info.def.add(phi)
				info.phiDef.add(phi)
				for i, pred := range block.Preds {
					edge := info.edgeUse[pred.Index]
					if edge == nil {
						edge = make(valueSet)
						info.edgeUse[pred.Index] = edge
					}
					addOperand(edge, phi.Edges[i])
				}
				continue
			}
			for _, operand := range instr.Operands(nil) {
				if operand != nil && *operand != nil {
					value := *operand
					if _, defined := info.def[value]; !defined {
						addOperand(info.use, value)
					}
				}
			}
			if value, ok := instr.(ssa.Value); ok {
				info.def.add(value)
			}
		}
	}

	liveIn := make([]valueSet, len(blocks))
	liveOut := make([]valueSet, len(blocks))
	changed := true
	for changed {
		changed = false
		for i := len(fn.Blocks) - 1; i >= 0; i-- {
			block := fn.Blocks[i]
			out := make(valueSet)
			for _, succ := range block.Succs {
				for value := range liveIn[succ.Index] {
					if _, isPhi := blocks[succ.Index].phiDef[value]; !isPhi {
						out.add(value)
					}
				}
				for value := range blocks[succ.Index].edgeUse[block.Index] {
					out.add(value)
				}
			}
			in := out.clone()
			in.removeAll(blocks[block.Index].def)
			in.addAll(blocks[block.Index].use)
			if !out.equal(liveOut[block.Index]) || !in.equal(liveIn[block.Index]) {
				liveOut[block.Index] = out
				liveIn[block.Index] = in
				changed = true
			}
		}
	}

	roots := make(map[ssa.Value]struct{})
	for _, block := range fn.Blocks {
		live := liveOut[block.Index].clone()
		for i := len(block.Instrs) - 1; i >= 0; i-- {
			instr := block.Instrs[i]
			if _, ok := instr.(*ssa.Phi); ok {
				continue
			}
			if value, ok := instr.(ssa.Value); ok {
				delete(live, value)
			}
			for _, operand := range instr.Operands(nil) {
				if operand != nil && *operand != nil {
					addOperand(live, *operand)
				}
			}
			if isSafepoint(instr) {
				for value := range live {
					if needsRoot(value) {
						roots[value] = struct{}{}
					}
				}
			}
		}
	}
	return roots
}

type blockInfo struct {
	def     valueSet
	use     valueSet
	phiDef  valueSet
	edgeUse map[int]valueSet
}

type valueSet map[ssa.Value]struct{}

func (s valueSet) add(value ssa.Value) {
	if value != nil {
		s[value] = struct{}{}
	}
}

func (s valueSet) addAll(other valueSet) {
	for value := range other {
		s.add(value)
	}
}

func (s valueSet) removeAll(other valueSet) {
	for value := range other {
		delete(s, value)
	}
}

func (s valueSet) clone() valueSet {
	clone := make(valueSet, len(s))
	clone.addAll(s)
	return clone
}

func (s valueSet) equal(other valueSet) bool {
	if len(s) != len(other) {
		return false
	}
	for value := range s {
		if _, ok := other[value]; !ok {
			return false
		}
	}
	return true
}

func addOperand(set valueSet, value ssa.Value) {
	switch value.(type) {
	case nil, *ssa.Builtin, *ssa.Const, *ssa.Function, *ssa.Global:
		return
	default:
		set.add(value)
	}
}

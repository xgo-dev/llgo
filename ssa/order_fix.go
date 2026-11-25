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

package ssa

import "go/token"

// fixReturnLocalLoadOrder aligns return-statement evaluation order with
// cmd/compile for patterns such as:
//
//	var o T
//	return o, o.mutate()
//
// where mutate has a pointer receiver and mutates fields of o. Standard SSA
// construction may materialize the load of o before the call; moving loads of
// local allocs that feed a Return result (and have no intervening executable
// use or store) to immediately after the last call referencing that alloc
// preserves cmd/compile stdlib compatibility (e.g. crypto/x509.ParseOID)
// natively inside the SSA builder without external post-passes.
func fixReturnLocalLoadOrder(fn *Function) {
	if fn == nil || len(fn.Blocks) == 0 {
		return
	}
	for _, b := range fn.Blocks {
		fixReturnLocalLoadOrderBlock(b)
	}
}

func fixReturnLocalLoadOrderBlock(b *BasicBlock) {
	if b == nil || len(b.Instrs) == 0 {
		return
	}
	retIdx := -1
	var ret *Return
	for i := len(b.Instrs) - 1; i >= 0; i-- {
		if r, ok := b.Instrs[i].(*Return); ok {
			retIdx = i
			ret = r
			break
		}
	}
	if retIdx < 0 || ret == nil {
		return
	}

	for _, rv := range ret.Results {
		u, ok := rv.(*UnOp)
		if !ok || u.Op != token.MUL {
			continue
		}
		alloc, ok := u.X.(*Alloc)
		if !ok {
			continue
		}

		loadIdx := orderIndexOfInstr(b.Instrs, u)
		if loadIdx < 0 || loadIdx >= retIdx {
			continue
		}

		lastCallIdx := -1
		for i := loadIdx + 1; i < retIdx; i++ {
			ci, ok := b.Instrs[i].(CallInstruction)
			if !ok {
				continue
			}
			if orderCallUsesValue(ci, alloc) {
				lastCallIdx = i
			}
		}
		if lastCallIdx < 0 {
			continue
		}

		writtenBeforeReturn := false
		for i := loadIdx + 1; i < retIdx; i++ {
			if orderStoreWritesAlloc(b.Instrs[i], alloc) {
				writtenBeforeReturn = true
				break
			}
		}
		if writtenBeforeReturn {
			continue
		}

		movingIndices := map[int]struct{}{loadIdx: {}}
		moved := orderMovedValuesForIndices(b.Instrs, movingIndices)
		orderIncludeDebugRefsForMovedValues(b.Instrs, movingIndices, moved, loadIdx+1, retIdx)
		moving := make(map[Instruction]struct{}, len(movingIndices))
		for i := range movingIndices {
			moving[b.Instrs[i]] = struct{}{}
		}
		usedBeforeReturn := false
		for i := loadIdx + 1; i < retIdx; i++ {
			if _, isMoving := movingIndices[i]; isMoving {
				continue
			}
			if orderInstrUsesValue(b.Instrs[i], u) {
				usedBeforeReturn = true
				break
			}
		}
		if usedBeforeReturn {
			continue
		}

		b.Instrs = orderMoveInstrsAfter(b.Instrs, moving, b.Instrs[lastCallIdx])
		retIdx = orderIndexOfInstr(b.Instrs, ret)
	}
}

func orderIndexOfInstr(instrs []Instruction, target Instruction) int {
	for i, ins := range instrs {
		if ins == target {
			return i
		}
	}
	return -1
}

func orderMovedValuesForIndices(instrs []Instruction, move map[int]struct{}) map[Value]struct{} {
	moved := make(map[Value]struct{}, len(move))
	for i := range move {
		if i < 0 || i >= len(instrs) {
			continue
		}
		if v, ok := instrs[i].(Value); ok && v != nil {
			moved[v] = struct{}{}
		}
	}
	return moved
}

func orderIncludeDebugRefsForMovedValues(instrs []Instruction, move map[int]struct{}, moved map[Value]struct{}, from, through int) {
	if from < 0 {
		from = 0
	}
	if through > len(instrs) {
		through = len(instrs)
	}
	for i := from; i < through; i++ {
		if _, moving := move[i]; moving {
			continue
		}
		ref, ok := instrs[i].(*DebugRef)
		if !ok {
			continue
		}
		for v := range moved {
			if orderInstrUsesValue(ref, v) {
				move[i] = struct{}{}
				break
			}
		}
	}
}

func orderInstrUsesValue(ins Instruction, v Value) bool {
	if ins == nil || v == nil {
		return false
	}
	for _, op := range ins.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
}

func orderCallUsesValue(ci CallInstruction, v Value) bool {
	if ci == nil || v == nil {
		return false
	}
	c := ci.Common()
	if c == nil {
		return false
	}
	for _, op := range c.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
}

func orderStoreWritesAlloc(ins Instruction, alloc *Alloc) bool {
	store, ok := ins.(*Store)
	if !ok || store == nil || alloc == nil {
		return false
	}
	return orderValueDependsOn(store.Addr, alloc, map[Value]struct{}{})
}

func orderValueDependsOn(v, target Value, seen map[Value]struct{}) bool {
	if v == nil || target == nil {
		return false
	}
	if v == target {
		return true
	}
	if _, ok := seen[v]; ok {
		return false
	}
	seen[v] = struct{}{}
	ins, ok := v.(Instruction)
	if !ok || ins == nil {
		return false
	}
	for _, op := range ins.Operands(nil) {
		if op != nil && orderValueDependsOn(*op, target, seen) {
			return true
		}
	}
	return false
}

func orderMoveInstrsAfter(instrs []Instruction, moving map[Instruction]struct{}, anchor Instruction) []Instruction {
	if len(moving) == 0 || anchor == nil {
		return instrs
	}
	if _, ok := moving[anchor]; ok {
		panic("orderMoveInstrsAfter: anchor is in moving set")
	}
	moved := make([]Instruction, 0, len(moving))
	remaining := make([]Instruction, 0, len(instrs))
	for _, instr := range instrs {
		if _, ok := moving[instr]; ok {
			moved = append(moved, instr)
			continue
		}
		remaining = append(remaining, instr)
	}
	for i, instr := range remaining {
		if instr == anchor {
			ret := make([]Instruction, 0, len(instrs))
			ret = append(ret, remaining[:i+1]...)
			ret = append(ret, moved...)
			ret = append(ret, remaining[i+1:]...)
			return ret
		}
	}
	return instrs
}

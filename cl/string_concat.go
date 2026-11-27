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
	"go/token"
	"slices"

	llabi "github.com/xgo-dev/llgo/internal/abi"
	"golang.org/x/tools/go/ssa"
)

const maxStringConcatParts = 16

// A root maps to its ordered operands. A present entry with nil operands is an
// intermediate consumed by that root. All other instructions lower normally.
type stringConcatPlans map[*ssa.BinOp][]ssa.Value

func (plans stringConcatPlans) omits(add *ssa.BinOp) bool {
	parts, found := plans[add]
	return found && parts == nil
}

func isStringAdd(v *ssa.BinOp) bool {
	return v != nil && v.Op == token.ADD && isStringType(v.Type())
}

// Dynamic parts may be empty. Literal bytes give a lower bound on the result.
func concatMayFitStackBuffer(parts []ssa.Value) bool {
	remaining := llabi.SmallStackBufferSize
	for _, part := range parts {
		literal, _ := constStr(part)
		remaining -= len(literal)
		if remaining < 0 {
			return false
		}
	}
	return true
}

func (p *context) prepareStringConcatPlans(fn *ssa.Function) {
	p.concatPlans = nil
	var run []*ssa.BinOp
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if _, debug := instr.(*ssa.DebugRef); debug {
				continue
			}
			if add, _ := instr.(*ssa.BinOp); isStringAdd(add) {
				run = append(run, add)
				continue
			}
			if len(run) > 0 {
				p.planStringConcatRun(run)
				run = run[:0]
			}
		}
		p.planStringConcatRun(run)
		run = run[:0]
	}
}

// Each group is a contiguous subtree in SSA evaluation order. A new addition
// can absorb the preceding groups only while their results feed it exclusively.
// A full group stays materialized and can be an operand of the next group.
func (p *context) planStringConcatRun(run []*ssa.BinOp) {
	if len(run) < 2 {
		return
	}
	ends := []int{0}
	for i, add := range run {
		for len(ends) > 1 {
			start, end := ends[len(ends)-2], ends[len(ends)-1]
			// The subtree run[start:i+1] has one more operand than additions.
			if i-start+2 > maxStringConcatParts || singleNonDebugReferrer(run[end-1]) != add {
				break
			}
			ends = ends[:len(ends)-1]
		}
		ends = append(ends, i+1)
	}
	for i, end := range ends[1:] {
		group := run[ends[i]:end]
		if len(group) < 2 {
			continue
		}
		parts := collectStringConcatParts(group, p.staticInitInstrs)
		if parts == nil || (len(parts) > 2 && !p.fn.ReserveStringConcatTable(len(parts))) {
			continue
		}
		if p.concatPlans == nil {
			p.concatPlans = make(stringConcatPlans)
		}
		p.concatPlans[group[len(group)-1]] = parts
		for _, add := range group[:len(group)-1] {
			p.concatPlans[add] = nil
		}
	}
}

func collectStringConcatParts(group []*ssa.BinOp, static map[ssa.Instruction]none) []ssa.Value {
	parts := make([]ssa.Value, 0, len(group)+1)
	// Keep at least two operands. Even a single non-empty input must be copied
	// because its bytes may belong to a temporary stack slot.
	emptyToDrop := len(group) - 1
	var collect func(ssa.Value) bool
	collect = func(v ssa.Value) bool {
		if instr, ok := v.(ssa.Instruction); ok {
			if _, skipped := static[instr]; skipped {
				return false
			}
		}
		if add, ok := v.(*ssa.BinOp); ok && slices.Contains(group, add) {
			return collect(add.X) && collect(add.Y)
		}
		if literal, ok := constStr(v); ok && literal == "" && emptyToDrop > 0 {
			emptyToDrop--
			return true
		}
		parts = append(parts, v)
		return true
	}
	if !collect(group[len(group)-1]) {
		return nil
	}
	return parts
}

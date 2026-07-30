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

package wasmresume

import (
	"fmt"
	"sort"

	"github.com/xgo-dev/llvm"
)

type slotKind uint8

const (
	slotParameter slotKind = iota
	slotFunctionResult
	slotAlloca
	slotValue
	slotUnwind
)

type frameSlot struct {
	id      uint32
	kind    slotKind
	typ     llvm.Type
	value   llvm.Value
	dynamic bool
}

type callSite struct {
	id         uint32
	call       llvm.Value
	live       []uint32
	resultSlot uint32
}

type framePlan struct {
	function    llvm.Value
	slots       []frameSlot
	resultSlot  uint32
	calls       []callSite
	unwindSlot  uint32
	unwindPC    uint32
	unwindBlock llvm.BasicBlock
}

type blockLiveness struct {
	def     valueSet
	use     valueSet
	liveIn  valueSet
	liveOut valueSet
}

type valueSet map[llvm.Value]struct{}

// planFrames computes the persistent values needed by each generated frame.
// Inventory runs first so every resumable call has a stable in-function ID.
func planFrames(mod llvm.Module) ([]framePlan, error) {
	if _, err := Inventory(mod); err != nil {
		return nil, err
	}

	kind := mod.Context().MDKindID(CallMetadata)
	var plans []framePlan
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !hasFunctionMarker(fn) {
			continue
		}
		if err := llvm.VerifyFunction(fn, llvm.ReturnStatusAction); err != nil {
			return nil, fmt.Errorf("%s: invalid resumable function: %w", fn.Name(), err)
		}
		plan, err := planFunctionFrame(fn, kind)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fn.Name(), err)
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func planFunctionFrame(fn llvm.Value, metadataKind int) (framePlan, error) {
	unwind, err := findUnwindPlan(fn)
	if err != nil {
		return framePlan{}, err
	}
	values, candidates, kinds := frameCandidates(fn)
	blocks, liveness := analyzeLiveness(fn, candidates)

	var rawCalls []struct {
		id     uint32
		call   llvm.Value
		live   valueSet
		result llvm.Value
	}
	needed := make(valueSet)
	if !unwind.block.IsNil() {
		unionInto(needed, liveness[unwind.block].liveIn)
	}
	for _, block := range blocks {
		live := cloneSet(liveness[block].liveOut)
		for instr := block.LastInstruction(); !instr.IsNil(); instr = llvm.PrevInstruction(instr) {
			if hasMetadata(instr, metadataKind) {
				id, err := resumeID(instr.Metadata(metadataKind))
				if err != nil {
					return framePlan{}, err
				}
				across := cloneSet(live)
				var result llvm.Value
				if _, ok := across[instr]; ok {
					result = instr
					delete(across, instr)
					needed[instr] = struct{}{}
				}
				unionInto(across, referencedAllocas(instr, candidates))
				for value := range across {
					needed[value] = struct{}{}
				}
				rawCalls = append(rawCalls, struct {
					id     uint32
					call   llvm.Value
					live   valueSet
					result llvm.Value
				}{id: id, call: instr, live: across, result: result})
			}

			delete(live, instr)
			if instr.InstructionOpcode() != llvm.PHI {
				addLocalOperands(live, instr, candidates)
			}
		}
	}

	sort.Slice(rawCalls, func(i, j int) bool {
		return rawCalls[i].id < rawCalls[j].id
	})

	plan := framePlan{function: fn}
	slots := make(map[llvm.Value]uint32)
	addSlot := func(kind slotKind, typ llvm.Type, value llvm.Value, dynamic bool) uint32 {
		id := uint32(len(plan.slots) + 1)
		plan.slots = append(plan.slots, frameSlot{
			id: id, kind: kind, typ: typ, value: value, dynamic: dynamic,
		})
		if !value.IsNil() {
			slots[value] = id
		}
		return id
	}
	for _, value := range values {
		if kinds[value] == slotParameter {
			addSlot(slotParameter, value.Type(), value, false)
		}
	}
	if typ := fn.GlobalValueType().ReturnType(); typ.TypeKind() != llvm.VoidTypeKind {
		plan.resultSlot = addSlot(slotFunctionResult, typ, llvm.Value{}, false)
	}
	for _, value := range values {
		if kinds[value] == slotParameter {
			continue
		}
		if _, ok := needed[value]; !ok {
			continue
		}
		typ, dynamic := persistentSlotType(value, kinds[value])
		addSlot(kinds[value], typ, value, dynamic)
	}
	if !unwind.block.IsNil() {
		plan.unwindSlot = addSlot(slotUnwind, unwind.typ, llvm.Value{}, false)
		plan.unwindPC = 1
		if len(rawCalls) != 0 {
			plan.unwindPC = rawCalls[len(rawCalls)-1].id + 1
		}
		if plan.unwindPC > maxResumeID {
			return framePlan{}, fmt.Errorf("unwind state exceeds maximum resume ID")
		}
		plan.unwindBlock = unwind.block
	}

	for _, raw := range rawCalls {
		site := callSite{id: raw.id, call: raw.call}
		for _, value := range values {
			if _, ok := raw.live[value]; ok {
				site.live = append(site.live, slots[value])
			}
		}
		if !raw.result.IsNil() {
			site.resultSlot = slots[raw.result]
		}
		plan.calls = append(plan.calls, site)
	}
	return plan, nil
}

type unwindPlan struct {
	block llvm.BasicBlock
	typ   llvm.Type
}

func findUnwindPlan(fn llvm.Value) (unwindPlan, error) {
	var plan unwindPlan
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if instr.InstructionOpcode() != llvm.Call ||
				instr.CalledValue().Name() != RegisterUnwindSymbol {
				continue
			}
			if !plan.block.IsNil() {
				return unwindPlan{}, fmt.Errorf("multiple unwind registrations")
			}
			if instr.OperandsCount() < 3 {
				return unwindPlan{}, fmt.Errorf("invalid unwind registration")
			}
			address := instr.Operand(1)
			if address.OperandsCount() != 2 ||
				address.Operand(0) != fn ||
				!address.Operand(1).IsBasicBlock() {
				return unwindPlan{}, fmt.Errorf("invalid unwind handler")
			}
			plan.block = address.Operand(1).AsBasicBlock()
			plan.typ = instr.Operand(0).Type()
		}
	}
	return plan, nil
}

func persistentSlotType(value llvm.Value, kind slotKind) (llvm.Type, bool) {
	if kind != slotAlloca {
		return value.Type(), false
	}
	elem := value.AllocatedType()
	if value.OperandsCount() == 0 {
		return elem, false
	}
	count := value.Operand(0).IsAConstantInt()
	if count.IsNil() {
		return value.Type(), true
	}
	n := count.ZExtValue()
	if n == 1 {
		return elem, false
	}
	return llvm.ArrayType(elem, int(n)), false
}

func frameCandidates(fn llvm.Value) ([]llvm.Value, valueSet, map[llvm.Value]slotKind) {
	var values []llvm.Value
	candidates := make(valueSet)
	kinds := make(map[llvm.Value]slotKind)
	add := func(value llvm.Value, kind slotKind) {
		values = append(values, value)
		candidates[value] = struct{}{}
		kinds[value] = kind
	}
	for param := fn.FirstParam(); !param.IsNil(); param = llvm.NextParam(param) {
		add(param, slotParameter)
	}
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if instr.Type().TypeKind() == llvm.VoidTypeKind {
				continue
			}
			kind := slotValue
			if !instr.IsAAllocaInst().IsNil() {
				kind = slotAlloca
			}
			add(instr, kind)
		}
	}
	return values, candidates, kinds
}

func analyzeLiveness(fn llvm.Value, candidates valueSet) ([]llvm.BasicBlock, map[llvm.BasicBlock]*blockLiveness) {
	var blocks []llvm.BasicBlock
	info := make(map[llvm.BasicBlock]*blockLiveness)
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		blocks = append(blocks, block)
		state := &blockLiveness{
			def:     make(valueSet),
			use:     make(valueSet),
			liveIn:  make(valueSet),
			liveOut: make(valueSet),
		}
		info[block] = state
		for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if instr.InstructionOpcode() != llvm.PHI {
				for operand := range localOperands(instr, candidates) {
					if _, defined := state.def[operand]; !defined {
						state.use[operand] = struct{}{}
					}
				}
			}
			if _, ok := candidates[instr]; ok {
				state.def[instr] = struct{}{}
			}
		}
	}

	changed := true
	for changed {
		changed = false
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]
			state := info[block]
			out := make(valueSet)
			terminator := block.LastInstruction()
			for successorIndex := 0; successorIndex < terminator.SuccessorsCount(); successorIndex++ {
				successor := terminator.Successor(successorIndex)
				unionInto(out, info[successor].liveIn)
				addPhiEdgeUses(out, successor, block, candidates)
			}
			in := cloneSet(state.use)
			for value := range out {
				if _, defined := state.def[value]; !defined {
					in[value] = struct{}{}
				}
			}
			if !equalSet(out, state.liveOut) || !equalSet(in, state.liveIn) {
				state.liveOut = out
				state.liveIn = in
				changed = true
			}
		}
	}
	return blocks, info
}

func addPhiEdgeUses(dst valueSet, successor, predecessor llvm.BasicBlock, candidates valueSet) {
	for phi := successor.FirstInstruction(); !phi.IsNil() && phi.InstructionOpcode() == llvm.PHI; phi = llvm.NextInstruction(phi) {
		for i := 0; i < phi.IncomingCount(); i++ {
			value := phi.IncomingValue(i)
			if phi.IncomingBlock(i) == predecessor {
				if _, ok := candidates[value]; ok {
					dst[value] = struct{}{}
				}
			}
		}
	}
}

func localOperands(instr llvm.Value, candidates valueSet) valueSet {
	operands := make(valueSet)
	addLocalOperands(operands, instr, candidates)
	return operands
}

func addLocalOperands(dst valueSet, instr llvm.Value, candidates valueSet) {
	for i := 0; i < instr.OperandsCount(); i++ {
		operand := instr.Operand(i)
		if _, ok := candidates[operand]; ok {
			dst[operand] = struct{}{}
		}
	}
}

func referencedAllocas(call llvm.Value, candidates valueSet) valueSet {
	allocas := make(valueSet)
	visited := make(valueSet)
	var visit func(llvm.Value)
	visit = func(value llvm.Value) {
		if _, ok := visited[value]; ok {
			return
		}
		visited[value] = struct{}{}
		if !value.IsAAllocaInst().IsNil() {
			allocas[value] = struct{}{}
			return
		}
		if _, ok := candidates[value]; !ok {
			return
		}
		if value.IsAInstruction().IsNil() {
			return
		}
		switch value.InstructionOpcode() {
		case llvm.Call, llvm.Load:
			return
		}
		for i := 0; i < value.OperandsCount(); i++ {
			visit(value.Operand(i))
		}
	}

	callee := call.CalledValue()
	for i := 0; i < call.OperandsCount(); i++ {
		operand := call.Operand(i)
		if operand != callee {
			visit(operand)
		}
	}
	return allocas
}

func resumeID(marker llvm.Value) (uint32, error) {
	fields := marker.MDNodeOperands()
	if len(fields) != 2 || fields[1].IsAConstantInt().IsNil() {
		return 0, fmt.Errorf("resumable call marker has no resume ID")
	}
	id := fields[1].ZExtValue()
	if id == 0 || id > maxResumeID {
		return 0, fmt.Errorf("invalid resume ID %d", id)
	}
	return uint32(id), nil
}

func cloneSet(src valueSet) valueSet {
	dst := make(valueSet, len(src))
	unionInto(dst, src)
	return dst
}

func unionInto(dst, src valueSet) {
	for value := range src {
		dst[value] = struct{}{}
	}
}

func equalSet(a, b valueSet) bool {
	if len(a) != len(b) {
		return false
	}
	for value := range a {
		if _, ok := b[value]; !ok {
			return false
		}
	}
	return true
}

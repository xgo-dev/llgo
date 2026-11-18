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

package goescape

import (
	"sort"

	"golang.org/x/tools/go/ssa"
)

// scopes derives iteration lifetimes from the CFG, including backwards gotos.
// Removing a component's dominating header exposes its nested cycles. Regions
// without a dominating header keep a conservative allocation policy; treating
// an irreducible cycle as an ordinary one-shot block would be unsound.
type scopes struct {
	invocation *lifetime
	blocks     map[*ssa.BasicBlock]*lifetime
	headers    map[*lifetime]*ssa.BasicBlock
	uncertain  map[*ssa.BasicBlock]bool
	budget     int
}

func functionScopes(fn *ssa.Function) *scopes {
	s := &scopes{
		invocation: new(lifetime),
		blocks:     make(map[*ssa.BasicBlock]*lifetime),
		headers:    make(map[*lifetime]*ssa.BasicBlock),
		uncertain:  make(map[*ssa.BasicBlock]bool),
		budget:     100000,
	}
	for _, block := range fn.Blocks {
		s.blocks[block] = s.invocation
	}
	s.refine(fn.Blocks, s.invocation)
	return s
}

func (s *scopes) refine(blocks []*ssa.BasicBlock, parent *lifetime) {
	s.budget -= len(blocks)
	if s.budget < 0 {
		for _, block := range blocks {
			s.uncertain[block] = true
		}
		return
	}
	for _, component := range cyclicComponents(blocks) {
		var header *ssa.BasicBlock
		for _, candidate := range component {
			dominates := true
			for _, block := range component {
				s.budget--
				if s.budget < 0 || !candidate.Dominates(block) {
					dominates = false
					break
				}
			}
			if dominates {
				header = candidate
				break
			}
		}
		region := &lifetime{parent: parent}
		for _, block := range component {
			s.blocks[block] = region
		}
		if header == nil {
			for _, block := range component {
				s.uncertain[block] = true
			}
			continue
		}
		s.headers[region] = header
		var remaining []*ssa.BasicBlock
		for _, block := range component {
			if block != header {
				remaining = append(remaining, block)
			}
		}
		s.refine(remaining, region)
	}
}

func (s *scopes) valueScope(value ssa.Value) *lifetime {
	instr, ok := value.(ssa.Instruction)
	if !ok || instr.Block() == nil {
		return s.invocation
	}
	region := s.blocks[instr.Block()]
	if _, phi := value.(*ssa.Phi); phi && s.headers[region] == instr.Block() {
		// A loop-header phi can keep the preceding iteration's object alive
		// while the next object is created. It outlives either iteration.
		return region.parent
	}
	return region
}

func cyclicComponents(blocks []*ssa.BasicBlock) [][]*ssa.BasicBlock {
	allowed := make(map[*ssa.BasicBlock]bool, len(blocks))
	for _, block := range blocks {
		allowed[block] = true
	}
	index, low := make(map[*ssa.BasicBlock]int), make(map[*ssa.BasicBlock]int)
	active := make(map[*ssa.BasicBlock]bool)
	var stack []*ssa.BasicBlock
	var components [][]*ssa.BasicBlock
	next := 1
	var visit func(*ssa.BasicBlock)
	visit = func(block *ssa.BasicBlock) {
		index[block], low[block] = next, next
		next++
		stack = append(stack, block)
		active[block] = true
		for _, succ := range block.Succs {
			if !allowed[succ] {
				continue
			}
			if index[succ] == 0 {
				visit(succ)
				if low[succ] < low[block] {
					low[block] = low[succ]
				}
			} else if active[succ] && index[succ] < low[block] {
				low[block] = index[succ]
			}
		}
		if low[block] != index[block] {
			return
		}
		var component []*ssa.BasicBlock
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			active[member] = false
			component = append(component, member)
			if member == block {
				break
			}
		}
		cyclic := len(component) > 1
		if !cyclic {
			for _, succ := range block.Succs {
				if succ == block {
					cyclic = true
				}
			}
		}
		if cyclic {
			sort.Slice(component, func(i, j int) bool { return component[i].Index < component[j].Index })
			components = append(components, component)
		}
	}
	for _, block := range blocks {
		if index[block] == 0 {
			visit(block)
		}
	}
	return components
}

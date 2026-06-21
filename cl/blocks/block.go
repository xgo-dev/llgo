/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package blocks

import (
	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

type Info struct {
	Kind llssa.DoAction
	Next int
}

// -----------------------------------------------------------------------------

type blockState struct {
	self   *ssa.BasicBlock
	preds  int
	succs  []int
	loop   bool // marks block during findLoop traversal (ordering purposes only)
	inLoop bool // block participates in a cycle (SCC); defer in this block may execute multiple times
	always bool
	reach  bool
	fdel   bool
}

func (p *blockState) kind() llssa.DoAction {
	// Defer semantics depend on whether the block can be re-entered (loop),
	// not whether it was selected as a "loop path" by the ordering algorithm.
	if p.inLoop {
		return llssa.DeferInLoop
	}
	if p.always {
		return llssa.DeferAlways
	}
	return llssa.DeferInCond
}

func newSuccs(succs []*ssa.BasicBlock) []int {
	ret := make([]int, len(succs))
	for i, blk := range succs {
		ret[i] = blk.Index
	}
	return ret
}

func findLoop(states []*blockState, path []int, seen []bool, iblk int) []int {
	path = append(path, iblk)
	seen[iblk] = true
	self := states[iblk]
	for _, succ := range self.succs {
		if s := states[succ]; s.fdel || s.loop {
			continue
		}
		if pos := find(path, succ); pos >= 0 {
			if pos > 0 {
				for _, i := range path[pos:] { // mark inner loop
					states[i].loop = true
				}
				continue
			}
			for _, i := range path {
				s := states[i]
				s.loop = true
				s.fdel = true
			}
			for _, s := range states { // clear inner loop mark
				if !s.fdel {
					s.loop = false
				}
			}
			return path
		}
		if seen[succ] {
			continue
		}
		if ret := findLoop(states, path, seen, succ); len(ret) > 0 {
			return ret
		}
	}
	return nil
}

// https://en.wikipedia.org/wiki/Topological_sorting
func Infos(blks []*ssa.BasicBlock) []Info {
	n := len(blks)
	order := make([]int, 1, n+1)
	order[0] = 0
	end, iend := 0, 0
	states := make([]*blockState, n)
	for i, blk := range blks {
		preds := len(blk.Preds)
		if preds == 0 && i != 0 {
			order = append(order, i)
		}
		if isEnd(blk) {
			end++
			iend = i
		}
		states[i] = &blockState{
			self:  blk,
			preds: preds,
			succs: newSuccs(blk.Succs),
		}
	}

	markInLoop(states)

	path := make([]int, 0, n)
	if states[0].preds != 0 {
		if loop := findLoop(states, path, make([]bool, n), 0); len(loop) > 0 {
			order = append(order, loop[1:]...)
		}
	} else {
		states[0].always = true
	}
	if end == 1 {
		states[iend].always = true
	}
	pos := 0

retry:
	for pos < len(order) {
		iblk := order[pos]
		pos++
		state := states[iblk]
		state.fdel = true
		for _, succ := range state.succs {
			s := states[succ]
			if s.fdel {
				continue
			}
			if s.preds--; s.preds == 0 {
				order = append(order, succ)
			} else {
				s.reach = true
			}
		}
	}
	if pos < n {
		for iblk, state := range states {
			if state.fdel || !state.reach {
				continue
			}
			if loop := findLoop(states, path, make([]bool, n), iblk); len(loop) > 0 {
				order = append(order, loop...)
				goto retry
			}
		}
		panic("unreachable")
	}
	order = append(order, -1)
	ret := make([]Info, n)
	for i := 0; i < n; i++ {
		iblk := order[i]
		ret[iblk] = Info{states[iblk].kind(), order[i+1]}
	}
	return ret
}

func isEnd(blk *ssa.BasicBlock) bool {
	// Note: skip recover block
	return len(blk.Succs) == 0 && (len(blk.Preds) > 0 || blk.Index == 0)
}

func find(path []int, fv int) int {
	for i, v := range path {
		if v == fv {
			return i
		}
	}
	return -1
}

// -----------------------------------------------------------------------------

// markInLoop marks blocks that belong to a cycle using SCC detection.
// This is used for defer classification: a defer in a loop can execute multiple
// times and must be handled with DeferInLoop semantics even if the block isn't
// on the specific "loop path" selected for compilation ordering.
func markInLoop(states []*blockState) {
	n := len(states)
	// Tarjan SCC.
	index := 0
	indices := make([]int, n)
	lowlink := make([]int, n)
	onStack := make([]bool, n)
	for i := range indices {
		indices[i] = -1
	}
	stack := make([]int, 0, n)

	var strongconnect func(v int)
	strongconnect = func(v int) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range states[v].succs {
			if indices[w] == -1 {
				strongconnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
				continue
			}
			if onStack[w] && indices[w] < lowlink[v] {
				lowlink[v] = indices[w]
			}
		}

		if lowlink[v] != indices[v] {
			return
		}

		// v is the root of an SCC; pop it.
		scc := make([]int, 0, 4)
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}

		inLoop := false
		if len(scc) > 1 {
			inLoop = true
		} else {
			// Single-node SCC is a loop only if it has a self-edge.
			only := scc[0]
			for _, succ := range states[only].succs {
				if succ == only {
					inLoop = true
					break
				}
			}
		}
		if inLoop {
			for _, i := range scc {
				states[i].inLoop = true
			}
		}
	}

	for v := 0; v < n; v++ {
		if indices[v] == -1 {
			strongconnect(v)
		}
	}
}

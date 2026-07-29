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

// Package safepointplan identifies control-flow edges that must poll for
// cooperative scheduling.
package safepointplan

import "golang.org/x/tools/go/ssa"

// Backedges returns block terminators that close a DFS cycle. Polling before
// these instructions intersects every cycle, including irreducible control
// flow, without adding a poll to every block in a loop.
func Backedges(fn *ssa.Function) map[ssa.Instruction]struct{} {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}

	const (
		unvisited uint8 = iota
		visiting
		visited
	)
	state := make([]uint8, len(fn.Blocks))
	polls := make(map[ssa.Instruction]struct{})
	var visit func(*ssa.BasicBlock)
	visit = func(block *ssa.BasicBlock) {
		state[block.Index] = visiting
		for _, succ := range block.Succs {
			switch state[succ.Index] {
			case unvisited:
				visit(succ)
			case visiting:
				if n := len(block.Instrs); n != 0 {
					polls[block.Instrs[n-1]] = struct{}{}
				}
			}
		}
		state[block.Index] = visited
	}

	for _, block := range fn.Blocks {
		if state[block.Index] == unvisited {
			visit(block)
		}
	}
	if len(polls) == 0 {
		return nil
	}
	return polls
}

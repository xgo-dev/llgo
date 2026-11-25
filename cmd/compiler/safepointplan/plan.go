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
	type frame struct {
		block    *ssa.BasicBlock
		nextSucc int
	}

	stack := make([]frame, 0, len(fn.Blocks))
	for _, root := range fn.Blocks {
		if state[root.Index] != unvisited {
			continue
		}
		state[root.Index] = visiting
		stack = append(stack, frame{block: root})
		for len(stack) != 0 {
			top := &stack[len(stack)-1]
			if top.nextSucc == len(top.block.Succs) {
				state[top.block.Index] = visited
				stack = stack[:len(stack)-1]
				continue
			}
			succ := top.block.Succs[top.nextSucc]
			top.nextSucc++
			switch state[succ.Index] {
			case unvisited:
				state[succ.Index] = visiting
				stack = append(stack, frame{block: succ})
			case visiting:
				if n := len(top.block.Instrs); n != 0 {
					polls[top.block.Instrs[n-1]] = struct{}{}
				}
			}
		}
	}
	if len(polls) == 0 {
		return nil
	}
	return polls
}

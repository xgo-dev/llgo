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
	"testing"

	"golang.org/x/tools/go/ssa"

	llssa "github.com/xgo-dev/llgo/ssa"
)

type graphCase struct {
	name          string
	edges         [][]int
	wantOrder     []int
	wantKinds     []llssa.DoAction
	findLoopFrom  int
	checkFindLoop bool
}

func TestInfos(t *testing.T) {
	tests := []graphCase{
		{
			name:      "single-exit diamond",
			edges:     [][]int{{1, 2}, {3}, {3}, nil},
			wantOrder: []int{0, 1, 2, 3},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInCond,
				llssa.DeferInCond,
				llssa.DeferAlways,
			},
		},
		{
			name:      "multiple exits",
			edges:     [][]int{{1, 2}, nil, nil},
			wantOrder: []int{0, 1, 2},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInCond,
				llssa.DeferInCond,
			},
		},
		{
			name:      "natural loop",
			edges:     [][]int{{1}, {2, 3}, {1}, nil},
			wantOrder: []int{0, 1, 2, 3},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferAlways,
			},
		},
		{
			// findLoop selects the 1 -> 2 -> 1 cycle for ordering. Block 3
			// is nevertheless in the same SCC and can also execute repeatedly.
			name:      "multi-node SCC with conditional exit",
			edges:     [][]int{{1}, {2, 3}, {1}, {1, 4}, nil},
			wantOrder: []int{0, 1, 2, 3, 4},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferAlways,
			},
		},
		{
			name:      "entry self-loop",
			edges:     [][]int{{0}},
			wantOrder: []int{0},
			wantKinds: []llssa.DoAction{llssa.DeferInLoop},
		},
		{
			name:      "nested body before outer exit",
			edges:     [][]int{{1}, {2, 3}, {4}, nil, {5, 6}, {4}, {1}},
			wantOrder: []int{0, 1, 2, 4, 6, 5, 3},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
			},
		},
		{
			// DFS visits 7 before 5, but closing the outer cycle leaves both
			// inner-body blocks pending. Append them in block-index order,
			// once each, before scheduling the outer exit (3).
			name:      "multiple nested bodies before outer exit",
			edges:     [][]int{{1}, {2, 3}, {4}, nil, {7, 6}, {4}, {1}, {5}},
			wantOrder: []int{0, 1, 2, 4, 6, 5, 7, 3},
			wantKinds: []llssa.DoAction{
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferAlways,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
				llssa.DeferInLoop,
			},
		},
		disconnectedSharedSubgraph(),
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			infos := Infos(blocksForGraph(t, test.edges))
			checkInfos(t, infos, test.wantOrder, test.wantKinds)

			if test.checkFindLoop {
				states := statesForGraph(test.edges)
				if loop := findLoop(states, nil, make([]bool, len(states)), test.findLoopFrom); len(loop) != 0 {
					t.Fatalf("findLoop returned a loop in an acyclic shared subgraph: %v", loop)
				}
			}
		})
	}
}

func disconnectedSharedSubgraph() graphCase {
	// Block 2 models the recover block: it is a second entry that is not
	// reachable from the ordinary entry block. The remaining 64-node DAG has
	// overlapping paths; searching it without a visited set takes exponential
	// time. Keep the graph generated so the regression test itself stays small.
	const (
		recoverBlock = 2
		sharedBlocks = 64
	)
	n := recoverBlock + sharedBlocks
	edges := make([][]int, n)
	edges[0] = []int{1}
	for i := recoverBlock; i < n; i++ {
		if i+1 < n {
			edges[i] = append(edges[i], i+1)
		}
		if i+2 < n {
			edges[i] = append(edges[i], i+2)
		}
	}

	// Zero-predecessor blocks are queued before successors of block 0. Once
	// blocks 0 and 2 have run, the two components advance in index order.
	wantOrder := []int{0, recoverBlock, 1}
	for i := recoverBlock + 1; i < n; i++ {
		wantOrder = append(wantOrder, i)
	}
	wantKinds := make([]llssa.DoAction, n)
	for i := range wantKinds {
		wantKinds[i] = llssa.DeferInCond
	}
	wantKinds[0] = llssa.DeferAlways

	return graphCase{
		name:          "disconnected recover and shared subgraph",
		edges:         edges,
		wantOrder:     wantOrder,
		wantKinds:     wantKinds,
		findLoopFrom:  recoverBlock,
		checkFindLoop: true,
	}
}

func blocksForGraph(t *testing.T, edges [][]int) []*ssa.BasicBlock {
	t.Helper()
	blocks := make([]*ssa.BasicBlock, len(edges))
	for i := range blocks {
		blocks[i] = &ssa.BasicBlock{Index: i}
	}
	for from, succs := range edges {
		for _, to := range succs {
			if to < 0 || to >= len(blocks) {
				t.Fatalf("edge %d -> %d is outside a %d-block graph", from, to, len(blocks))
			}
			blocks[from].Succs = append(blocks[from].Succs, blocks[to])
			blocks[to].Preds = append(blocks[to].Preds, blocks[from])
		}
	}
	return blocks
}

func statesForGraph(edges [][]int) []*blockState {
	states := make([]*blockState, len(edges))
	for i, succs := range edges {
		states[i] = &blockState{succs: append([]int(nil), succs...)}
	}
	return states
}

func checkInfos(t *testing.T, got []Info, wantOrder []int, wantKinds []llssa.DoAction) {
	t.Helper()
	if len(got) != len(wantKinds) || len(got) != len(wantOrder) {
		t.Fatalf("got %d infos; want %d kinds and %d ordered blocks", len(got), len(wantKinds), len(wantOrder))
	}

	wantNext := make([]int, len(got))
	assigned := make([]bool, len(got))
	for pos, block := range wantOrder {
		if block < 0 || block >= len(got) {
			t.Fatalf("wantOrder[%d] = %d is outside a %d-block graph", pos, block, len(got))
		}
		if assigned[block] {
			t.Fatalf("block %d occurs more than once in wantOrder", block)
		}
		assigned[block] = true
		wantNext[block] = -1
		if pos+1 < len(wantOrder) {
			wantNext[block] = wantOrder[pos+1]
		}
	}

	for block, info := range got {
		if info.Kind != wantKinds[block] || info.Next != wantNext[block] {
			t.Errorf("block %d: got {Kind:%v Next:%d}; want {Kind:%v Next:%d}",
				block, info.Kind, info.Next, wantKinds[block], wantNext[block])
		}
	}
}

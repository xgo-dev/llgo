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

package callgraph_test

import (
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

func TestCallgraphOperations(t *testing.T) {
	prog := ssa.NewProgram(token.NewFileSet(), 0)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fnA := prog.NewFunction("A", sig, "")
	fnB := prog.NewFunction("B", sig, "synthetic wrapper")
	fnC := prog.NewFunction("C", sig, "")

	blkA := fnA.NewBasicBlock("entry")
	callAB := &ssa.Call{Call: ssa.CallCommon{Value: fnB}}
	blkA.Emit(callAB, nil, token.Pos(10))
	blkA.Emit(&ssa.Return{}, nil, token.Pos(11))

	blkB := fnB.NewBasicBlock("entry")
	callBC := &ssa.Call{Call: ssa.CallCommon{Value: fnC}}
	blkB.Emit(callBC, nil, token.Pos(20))
	blkB.Emit(&ssa.Return{}, nil, token.Pos(21))

	g := callgraph.New(fnA)
	nA := g.CreateNode(fnA)
	nB := g.CreateNode(fnB)
	nC := g.CreateNode(fnC)
	callgraph.AddEdge(nA, callAB, nB)
	callgraph.AddEdge(nB, callBC, nC)

	if !callgraph.CalleesOf(nA)[nB] {
		t.Fatal("expected B in CalleesOf(A)")
	}
	path := callgraph.PathSearch(nA, func(n *callgraph.Node) bool { return n == nC })
	if len(path) != 2 || path[0].Callee != nB || path[1].Callee != nC {
		t.Fatalf("unexpected PathSearch result: %v", path)
	}
	if path[0].Pos() != token.Pos(10) || path[0].Description() != "static function call" {
		t.Fatalf("unexpected edge metadata: pos=%v desc=%q", path[0].Pos(), path[0].Description())
	}

	visited := 0
	if err := callgraph.GraphVisitEdges(g, func(e *callgraph.Edge) error {
		visited++
		return nil
	}); err != nil || visited != 2 {
		t.Fatalf("GraphVisitEdges visited %d edges, err=%v", visited, err)
	}

	g.DeleteSyntheticNodes()
	if _, exists := g.Nodes[fnB]; exists {
		t.Fatal("expected synthetic node B deleted")
	}
	if !callgraph.CalleesOf(nA)[nC] {
		t.Fatal("expected direct edge A -> C after deleting synthetic node B")
	}
}

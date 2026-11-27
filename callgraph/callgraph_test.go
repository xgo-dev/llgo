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
	"errors"
	"go/token"
	"strings"
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

	// Nil safety and string/description formatting.
	var nilGraph *callgraph.Graph
	nilGraph.DeleteNode(nA)
	g.DeleteNode(nil)
	nilGraph.DeleteSyntheticNodes()
	if len(callgraph.CalleesOf(nil)) != 0 {
		t.Fatal("expected empty CalleesOf(nil)")
	}
	if err := callgraph.GraphVisitEdges(nil, func(*callgraph.Edge) error { return nil }); err != nil {
		t.Fatalf("GraphVisitEdges(nil) = %v", err)
	}
	if got := callgraph.PathSearch(nil, func(*callgraph.Node) bool { return true }); got != nil {
		t.Fatalf("PathSearch(nil) = %v", got)
	}

	var nilNode *callgraph.Node
	if !strings.Contains(nilNode.String(), "n-1:<root>") {
		t.Fatalf("nil Node.String() = %q", nilNode.String())
	}
	rootOnly := callgraph.New(nil)
	if !strings.Contains(rootOnly.Root.String(), "<root>") {
		t.Fatalf("root Node.String() = %q", rootOnly.Root.String())
	}
	if !strings.Contains(nA.String(), ":A") {
		t.Fatalf("nA.String() = %q", nA.String())
	}

	synEdge := callgraph.Edge{Caller: nA, Callee: nC}
	if synEdge.Description() != "synthetic function call" || synEdge.Pos() != token.NoPos {
		t.Fatalf("synthetic edge = (%q, %v)", synEdge.Description(), synEdge.Pos())
	}
	if !strings.Contains(synEdge.String(), "-->") {
		t.Fatalf("Edge.String() = %q", synEdge.String())
	}
	goInstr := &ssa.Go{Call: ssa.CallCommon{Value: fnC}}
	deferInstr := &ssa.Defer{Call: ssa.CallCommon{Value: fnC}}
	if desc := (callgraph.Edge{Caller: nA, Site: goInstr, Callee: nC}).Description(); !strings.HasPrefix(desc, "concurrent ") {
		t.Fatalf("go edge description = %q", desc)
	}
	if desc := (callgraph.Edge{Caller: nA, Site: deferInstr, Callee: nC}).Description(); !strings.HasPrefix(desc, "deferred ") {
		t.Fatalf("defer edge description = %q", desc)
	}

	// GraphVisitEdges error propagation through nested calls and PathSearch backtracking / miss.
	fnD := prog.NewFunction("D", sig, "")
	fnE := prog.NewFunction("E", sig, "")
	nD := g.CreateNode(fnD)
	nE := g.CreateNode(fnE)
	callgraph.AddEdge(nA, nil, nD)
	callgraph.AddEdge(nD, nil, nE)
	wantErr := errors.New("stop")
	if err := callgraph.GraphVisitEdges(g, func(e *callgraph.Edge) error {
		if e.Callee == nE {
			return wantErr
		}
		return nil
	}); !errors.Is(err, wantErr) {
		t.Fatalf("GraphVisitEdges error = %v, want %v", err, wantErr)
	}
	if got := callgraph.PathSearch(nA, func(n *callgraph.Node) bool { return n == nE }); len(got) != 2 || got[1].Callee != nE {
		t.Fatalf("PathSearch to E after backtracking = %v", got)
	}
	if got := callgraph.PathSearch(nA, func(*callgraph.Node) bool { return false }); got != nil {
		t.Fatalf("PathSearch unreachable = %v, want nil", got)
	}
}

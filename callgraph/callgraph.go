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

// Package callgraph defines the call graph data structure and helper utilities
// over github.com/xgo-dev/llgo/ssa, aligning 1:1 with
// golang.org/x/tools/go/callgraph (pure Go 1.20+, zero CGo/LLVM/x/tools).
package callgraph

import (
	"fmt"
	"go/token"

	"github.com/xgo-dev/llgo/ssa"
)

// Graph represents a call graph.
type Graph struct {
	Root  *Node
	Nodes map[*ssa.Function]*Node
}

// New returns a new Graph with the specified (optional) root function.
func New(root *ssa.Function) *Graph {
	g := &Graph{Nodes: make(map[*ssa.Function]*Node)}
	g.Root = g.CreateNode(root)
	return g
}

// CreateNode returns the Node for fn, creating it if not present.
func (g *Graph) CreateNode(fn *ssa.Function) *Node {
	n, ok := g.Nodes[fn]
	if !ok {
		n = &Node{Func: fn, ID: len(g.Nodes)}
		g.Nodes[fn] = n
	}
	return n
}

// DeleteNode removes node n and its edges from the graph g.
func (g *Graph) DeleteNode(n *Node) {
	if g == nil || n == nil {
		return
	}
	for _, e := range n.In {
		if e.Caller != n {
			e.Caller.Out = removeEdgesTo(e.Caller.Out, n)
		}
	}
	for _, e := range n.Out {
		if e.Callee != n {
			e.Callee.In = removeEdgesFrom(e.Callee.In, n)
		}
	}
	n.In = nil
	n.Out = nil
	delete(g.Nodes, n.Func)
}

func removeEdgesTo(edges []*Edge, callee *Node) []*Edge {
	out := edges[:0]
	for _, e := range edges {
		if e.Callee != callee {
			out = append(out, e)
		}
	}
	return out
}

func removeEdgesFrom(edges []*Edge, caller *Node) []*Edge {
	out := edges[:0]
	for _, e := range edges {
		if e.Caller != caller {
			out = append(out, e)
		}
	}
	return out
}

// DeleteSyntheticNodes removes from call graph g all nodes for synthetic
// functions (except g.Root and package initializers), preserving topology.
func (g *Graph) DeleteSyntheticNodes() {
	if g == nil {
		return
	}
	var toDelete []*Node
	for fn, n := range g.Nodes {
		if n == g.Root || fn == nil || fn.Synthetic == "" || fn.Name() == "init" {
			continue
		}
		toDelete = append(toDelete, n)
	}
	for _, n := range toDelete {
		for _, in := range n.In {
			for _, out := range n.Out {
				if in.Caller != n && out.Callee != n {
					AddEdge(in.Caller, in.Site, out.Callee)
				}
			}
		}
		g.DeleteNode(n)
	}
}

// Node represents a node in a call graph.
type Node struct {
	Func *ssa.Function
	ID   int
	In   []*Edge
	Out  []*Edge
}

func (n *Node) String() string {
	if n == nil || n.Func == nil {
		return fmt.Sprintf("n%d:<root>", nodeID(n))
	}
	return fmt.Sprintf("n%d:%s", n.ID, n.Func.String())
}

func nodeID(n *Node) int {
	if n == nil {
		return -1
	}
	return n.ID
}

// Edge represents an edge in the call graph.
type Edge struct {
	Caller *Node
	Site   ssa.CallInstruction
	Callee *Node
}

func (e Edge) Description() string {
	var prefix string
	switch e.Site.(type) {
	case nil:
		return "synthetic function call"
	case *ssa.Go:
		prefix = "concurrent "
	case *ssa.Defer:
		prefix = "deferred "
	}
	return prefix + e.Site.Common().Description()
}

func (e Edge) Pos() token.Pos {
	if e.Site == nil {
		return token.NoPos
	}
	return e.Site.Pos()
}

func (e Edge) String() string {
	return fmt.Sprintf("%s --> %s", e.Caller, e.Callee)
}

// AddEdge adds the edge (caller, site, callee) to the call graph.
func AddEdge(caller *Node, site ssa.CallInstruction, callee *Node) {
	e := &Edge{Caller: caller, Site: site, Callee: callee}
	callee.In = append(callee.In, e)
	caller.Out = append(caller.Out, e)
}

// CalleesOf returns a new set containing all direct callees of the caller node.
func CalleesOf(caller *Node) map[*Node]bool {
	callees := make(map[*Node]bool)
	if caller == nil {
		return callees
	}
	for _, e := range caller.Out {
		callees[e.Callee] = true
	}
	return callees
}

// GraphVisitEdges visits all the edges in graph g in depth-first order.
func GraphVisitEdges(g *Graph, edge func(*Edge) error) error {
	if g == nil || g.Root == nil {
		return nil
	}
	seen := make(map[*Node]bool)
	var visit func(n *Node) error
	visit = func(n *Node) error {
		if !seen[n] {
			seen[n] = true
			for _, e := range n.Out {
				if err := visit(e.Callee); err != nil {
					return err
				}
				if err := edge(e); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(g.Root)
}

// PathSearch finds an arbitrary path starting at node start and ending at some
// node for which isEnd() returns true. On success, PathSearch returns the path
// as an ordered list of edges; on failure, it returns nil.
func PathSearch(start *Node, isEnd func(*Node) bool) []*Edge {
	if start == nil || isEnd == nil {
		return nil
	}
	stack := make([]*Edge, 0, 32)
	seen := make(map[*Node]bool)
	var search func(n *Node) []*Edge
	search = func(n *Node) []*Edge {
		if !seen[n] {
			seen[n] = true
			if isEnd(n) {
				res := make([]*Edge, len(stack))
				copy(res, stack)
				return res
			}
			for _, e := range n.Out {
				stack = append(stack, e)
				if found := search(e.Callee); found != nil {
					return found
				}
				stack = stack[:len(stack)-1]
			}
		}
		return nil
	}
	return search(start)
}

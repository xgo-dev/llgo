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

// Package callgraph constructs a pure-Go call graph and RTA-style reachability
// graph over llgo/ssa packages and cached exportdata.PackageMeta summaries
// (zero CGo, zero LLVM, zero golang.org/x/tools).
package callgraph

import (
	"sort"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
)

// EdgeKind classifies how a call edge is invoked.
type EdgeKind uint8

const (
	EdgeDirect    EdgeKind = 0
	EdgeClosure   EdgeKind = 1
	EdgeInterface EdgeKind = 2
)

// Edge represents a directed call or closure-reference edge.
type Edge struct {
	Caller string
	Callee string
	Kind   EdgeKind
}

// Node represents a function symbol in the call graph.
type Node struct {
	Name string
	Out  []Edge
	In   []Edge
}

// Graph holds the analyzed call graph and live interface-method dispatch table.
type Graph struct {
	Nodes          map[string]*Node
	Roots          []string
	Reachable      map[string]bool
	InterfaceCalls map[string]bool
	LiveTypes      map[string]bool
}

// ExtractEdges extracts deterministic summarized call edges from pkg.
func ExtractEdges(pkg *ssa.Package) []exportdata.CallEdge {
	if pkg == nil {
		return nil
	}
	seen := make(map[exportdata.CallEdge]bool)
	var edges []exportdata.CallEdge
	add := func(caller, callee string, kind EdgeKind) {
		if caller == "" || callee == "" {
			return
		}
		e := exportdata.CallEdge{Caller: caller, Callee: callee, Kind: uint8(kind)}
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}
	for _, fn := range pkg.Funcs {
		for _, blk := range fn.Blocks {
			for _, ins := range blk.Instrs {
				switch ins := ins.(type) {
				case *ssa.CallDirect:
					add(fn.Name, ins.Callee, EdgeDirect)
				case *ssa.MakeClosure:
					add(fn.Name, ins.FnName, EdgeClosure)
				case *ssa.CallInterface:
					add(fn.Name, ins.Method, EdgeInterface)
				}
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		if edges[i].Callee != edges[j].Callee {
			return edges[i].Callee < edges[j].Callee
		}
		return edges[i].Kind < edges[j].Kind
	})
	return edges
}

// Analyze builds a call graph combining live ssa.Packages (dirty packages) and
// cached exportdata.PackageMeta summaries (cache hits), then computes
// reachability from roots (resolving interface calls against methods defined on
// live types/functions whose suffix matches ".<Method>").
func Analyze(pkgs []*ssa.Package, metas []*exportdata.PackageMeta, roots []string) *Graph {
	g := &Graph{
		Nodes:          make(map[string]*Node),
		Reachable:      make(map[string]bool),
		InterfaceCalls: make(map[string]bool),
		LiveTypes:      make(map[string]bool),
	}

	ensure := func(name string) *Node {
		n := g.Nodes[name]
		if n == nil {
			n = &Node{Name: name}
			g.Nodes[name] = n
		}
		return n
	}

	edgeSeen := make(map[Edge]bool)
	addEdge := func(caller, callee string, kind EdgeKind) {
		if caller == "" || callee == "" {
			return
		}
		if kind == EdgeInterface {
			g.InterfaceCalls[callee] = true
		}
		e := Edge{Caller: caller, Callee: callee, Kind: kind}
		if edgeSeen[e] {
			return
		}
		edgeSeen[e] = true
		cn := ensure(caller)
		cn.Out = append(cn.Out, e)
		if kind != EdgeInterface {
			tn := ensure(callee)
			tn.In = append(tn.In, e)
		}
	}

	// Track candidate concrete methods by short method name for interface dispatch.
	methodsByName := make(map[string][]string)
	registerFunc := func(name string) {
		if name == "" {
			return
		}
		ensure(name)
		if dot := lastDot(name); dot >= 0 && dot+1 < len(name) {
			short := name[dot+1:]
			methodsByName[short] = append(methodsByName[short], name)
		}
	}

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, fn := range pkg.Funcs {
			registerFunc(fn.Name)
		}
		for _, e := range ExtractEdges(pkg) {
			addEdge(e.Caller, e.Callee, EdgeKind(e.Kind))
		}
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}
		for _, fn := range meta.Summary.DefinedFuncs {
			registerFunc(fn)
		}
		for _, lt := range meta.LiveTypes {
			g.LiveTypes[lt] = true
		}
		for _, e := range meta.CallEdges {
			addEdge(e.Caller, e.Callee, EdgeKind(e.Kind))
		}
		for _, m := range meta.Summary.InterfaceCalls {
			g.InterfaceCalls[m] = true
		}
	}

	// Sort edges deterministically.
	for _, n := range g.Nodes {
		sortEdges(n.Out)
		sortEdges(n.In)
	}
	for k, list := range methodsByName {
		sort.Strings(list)
		methodsByName[k] = dedupStrings(list)
	}

	// BFS reachability from roots (expanding interface dispatch when reached).
	var queue []string
	enqueue := func(name string) {
		if name == "" || g.Reachable[name] {
			return
		}
		g.Reachable[name] = true
		ensure(name)
		queue = append(queue, name)
	}
	sortedRoots := append([]string(nil), roots...)
	sort.Strings(sortedRoots)
	g.Roots = dedupStrings(sortedRoots)
	for _, r := range g.Roots {
		enqueue(r)
	}

	expandedIface := make(map[string]bool)
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		node := g.Nodes[cur]
		if node == nil {
			continue
		}
		for _, e := range node.Out {
			if e.Kind == EdgeInterface {
				if !expandedIface[e.Callee] {
					expandedIface[e.Callee] = true
					for _, impl := range methodsByName[e.Callee] {
						enqueue(impl)
					}
				}
				continue
			}
			enqueue(e.Callee)
		}
	}
	return g
}

func sortEdges(edges []Edge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		if edges[i].Callee != edges[j].Callee {
			return edges[i].Callee < edges[j].Callee
		}
		return edges[i].Kind < edges[j].Kind
	})
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

func dedupStrings(sorted []string) []string {
	if len(sorted) <= 1 {
		return sorted
	}
	out := sorted[:1]
	for _, s := range sorted[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}

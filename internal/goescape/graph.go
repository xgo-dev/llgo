// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in LICENSES/Go-BSD-3-Clause.txt.

// Package goescape analyzes Go storage lifetimes before LLVM lowering.
// The location graph and solver adapt Go 1.27.3's
// cmd/compile/internal/escape/{graph,solve,leaks}.go to a frontend-independent
// representation. SSA construction and backend placement are separate layers.
package goescape

// A lifetime identifies an invocation or one of its nested iteration scopes.
// The frontend creates separate scopes for loops; unrelated loop regions never
// imply that their objects can share a slot.
type lifetime struct {
	parent *lifetime
}

func (s *lifetime) contains(other *lifetime) bool {
	for ; other != nil; other = other.parent {
		if s == other {
			return true
		}
	}
	return false
}

type attributes uint8

const (
	escapes attributes = 1 << iota
	persists
	mutates
	calls
)

// A location is storage, not necessarily the pointer value referring to it.
// This distinction lets an interface box remain local while its payload flows
// to a caller, or a local pointer variable refer to an escaping object.
type location struct {
	name   string
	scope  *lifetime
	param  bool
	result int // one-based result index; zero for non-results
	attrs  attributes
	edges  []edge
	leaks  leaks
	queued bool
	walk   uint64
	derefs int
	reason string // first lifetime witness that forced this storage to escape
}

type edge struct {
	src    *location
	derefs int
}

// A zero leak means no flow; otherwise it encodes dereference depth + 1.
// Saturation lowers precision conservatively, rather than wrapping to no flow.
type leak uint16

func (l leak) depth() int { return int(l) - 1 }

func (l *leak) add(depth int) {
	if depth < 0 {
		panic("goescape: negative leak depth")
	}
	if depth > 65534 {
		depth = 65534
	}
	if *l == 0 || depth < l.depth() {
		*l = leak(depth + 1)
	}
}

type leaks struct {
	heap, mutator, callee leak
	results               []leak
}

func (l *leaks) addResult(index, depth int) {
	for len(l.results) <= index {
		l.results = append(l.results, 0)
	}
	l.results[index].add(depth)
}

// Each graph owns one function. Cross-function and closure flows use call
// summaries; unresolved external calls flow into the heap sink.
type graph struct {
	all                   []*location
	heap, mutator, callee location
	work                  []*location
	generation            uint64
	budget                int
}

func newGraph() *graph {
	return &graph{
		budget:  1_000_000,
		heap:    location{name: "heap", attrs: escapes | persists | mutates | calls},
		mutator: location{name: "mutator", attrs: mutates},
		callee:  location{name: "callee", attrs: calls},
	}
}

func (g *graph) location(name string, scope *lifetime) *location {
	l := &location{name: name, scope: scope}
	g.all = append(g.all, l)
	return l
}

func (g *graph) flow(dst, src *location, derefs int) {
	if dst == nil || src == nil || derefs < -1 {
		panic("goescape: invalid flow")
	}
	if dst == src && derefs >= 0 {
		return
	}
	if derefs > 65534 {
		derefs = 65534
	}
	dst.edges = append(dst.edges, edge{src: src, derefs: derefs})
}

func (g *graph) enqueue(l *location) {
	if !l.queued {
		l.queued = true
		g.work = append(g.work, l)
	}
}

func (g *graph) solve() bool {
	for _, l := range g.all {
		g.enqueue(l)
	}
	g.enqueue(&g.heap)
	g.enqueue(&g.mutator)
	g.enqueue(&g.callee)
	for len(g.work) != 0 {
		last := len(g.work) - 1
		root := g.work[last]
		g.work = g.work[:last]
		root.queued = false
		g.walk(root)
		if g.budget < 0 {
			return false
		}
	}
	return true
}

func (g *graph) walk(root *location) {
	g.generation++
	generation := g.generation
	root.walk, root.derefs = generation, 0
	pending := []*location{root}
	for len(pending) != 0 {
		g.budget--
		if g.budget < 0 {
			return
		}
		last := len(pending) - 1
		l := pending[last]
		pending = pending[:last]
		depth := l.derefs
		var add attributes
		if depth < 0 {
			// root = &l; l = x carries l's address, but only x's value.
			// Clamp here so address cycles cannot accumulate negative depth.
			depth = 0
			if g.outlives(root, l) {
				add |= escapes | persists | mutates | calls
				if l.reason == "" {
					switch {
					case root.reason != "":
						l.reason = root.reason
					case root.result != 0:
						l.reason = "returned address"
					case root.attrs&escapes != 0:
						l.reason = "stored in heap"
					default:
						l.reason = "cross-iteration retention"
					}
				}
			} else if root.attrs&persists != 0 {
				add |= persists
			}
		}
		if depth == 0 {
			add |= root.attrs & (mutates | calls)
		}
		if l.param {
			if g.outlives(root, l) {
				if root.result > 0 && root.attrs&escapes == 0 {
					l.leaks.addResult(root.result-1, depth)
				} else {
					l.leaks.heap.add(depth)
				}
			}
			if root.attrs&mutates != 0 {
				l.leaks.mutator.add(depth)
			}
			if root.attrs&calls != 0 {
				l.leaks.callee.add(depth)
			}
		}
		if add & ^l.attrs != 0 {
			l.attrs |= add
			g.enqueue(l)
		}
		for _, e := range l.edges {
			g.budget--
			if g.budget < 0 {
				return
			}
			d := depth + e.derefs
			if d > 65534 {
				d = 65534
			}
			if e.src.walk != generation || d < e.src.derefs {
				e.src.walk, e.src.derefs = generation, d
				pending = append(pending, e.src)
			}
		}
	}
}

func (g *graph) outlives(dst, src *location) bool {
	if dst.attrs&escapes != 0 {
		return true
	}
	if dst == &g.mutator || dst == &g.callee {
		return false
	}
	if dst.result != 0 {
		return true
	}
	return dst.scope != nil && src.scope != nil && dst.scope != src.scope && !src.scope.contains(dst.scope)
}

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

// Package escape computes pure-Go per-package and summary-driven cross-package
// escape analysis over llgo/ssa IR and exportdata.EscapeSummary.
package escape

import (
	"sort"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
)

const maxFixpointRounds = 16

// AnalyzePackage computes deterministic escape summaries for every defined
// function in pkg using imported package escape summaries for cross-package
// callees. Functions without an imported summary are treated fail-closed
// (parameters conservatively escape to heap).
func AnalyzePackage(pkg *ssa.Package, imports map[string]*exportdata.EscapeSummary) *exportdata.EscapeSummary {
	if pkg == nil {
		return &exportdata.EscapeSummary{}
	}
	known := make(map[string]exportdata.FuncEscape)
	for _, imp := range imports {
		if imp == nil {
			continue
		}
		for _, fe := range imp.Funcs {
			known[fe.Name] = fe
		}
	}

	// Initialize local function summaries optimistically (no escape) and iterate
	// monotonically to a bounded fixpoint over intraprocedural & local edges.
	local := make(map[string]exportdata.FuncEscape, len(pkg.Funcs))
	for _, fn := range pkg.Funcs {
		if fn.IsDeclaration() {
			// External declarations default fail-closed: all pointer-bearing params escape.
			params := make([]exportdata.ParamEscapeFlags, len(fn.Params))
			for i, p := range fn.Params {
				if p.Type() != nil && p.Type().HasPtrData() {
					params[i] = exportdata.ParamToHeap
				}
			}
			known[fn.Name] = exportdata.FuncEscape{Name: fn.Name, Params: params}
			continue
		}
		local[fn.Name] = exportdata.FuncEscape{
			Name:   fn.Name,
			Params: make([]exportdata.ParamEscapeFlags, len(fn.Params)),
		}
	}

	lookup := func(name string) (exportdata.FuncEscape, bool) {
		if fe, ok := local[name]; ok {
			return fe, true
		}
		fe, ok := known[name]
		return fe, ok
	}

	for round := 0; round < maxFixpointRounds; round++ {
		changed := false
		for _, fn := range pkg.Funcs {
			if fn.IsDeclaration() {
				continue
			}
			next := analyzeFunc(fn, lookup)
			prev := local[fn.Name]
			if !funcEscapeEqual(prev, next) {
				local[fn.Name] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	res := &exportdata.EscapeSummary{
		PkgPath: pkg.Path,
		Funcs:   make([]exportdata.FuncEscape, 0, len(pkg.Funcs)),
	}
	for _, fn := range pkg.Funcs {
		if fe, ok := local[fn.Name]; ok {
			res.Funcs = append(res.Funcs, fe)
		} else if fe, ok := known[fn.Name]; ok {
			res.Funcs = append(res.Funcs, fe)
		}
	}
	sort.Slice(res.Funcs, func(i, j int) bool { return res.Funcs[i].Name < res.Funcs[j].Name })
	return res
}

func analyzeFunc(fn *ssa.Function, lookup func(string) (exportdata.FuncEscape, bool)) exportdata.FuncEscape {
	// Track which SSA values derive from which parameters or local allocs.
	paramOrigin := make(map[ssa.Value][]int)
	allocOrigin := make(map[ssa.Value][]*ssa.Alloc)
	paramFlags := make([]exportdata.ParamEscapeFlags, len(fn.Params))
	allocEscapes := make(map[*ssa.Alloc]bool)

	for i, p := range fn.Params {
		paramOrigin[p] = []int{i}
	}

	addParamFlag := func(v ssa.Value, flag exportdata.ParamEscapeFlags) {
		for _, idx := range paramOrigin[v] {
			paramFlags[idx] |= flag
		}
		if flag&(exportdata.ParamToHeap|exportdata.ParamToReturn) != 0 {
			for _, a := range allocOrigin[v] {
				allocEscapes[a] = true
			}
		}
	}

	propagate := func(dst, src ssa.Value) {
		if ps := paramOrigin[src]; len(ps) > 0 {
			paramOrigin[dst] = appendUniqueInts(paramOrigin[dst], ps)
		}
		if as := allocOrigin[src]; len(as) > 0 {
			allocOrigin[dst] = appendUniqueAllocs(allocOrigin[dst], as)
		}
	}

	var allAllocs []*ssa.Alloc
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			if a, ok := ins.(*ssa.Alloc); ok {
				allAllocs = append(allAllocs, a)
				allocOrigin[a] = []*ssa.Alloc{a}
				if a.Heap {
					// Keep initial heap flag informational; escapes determines necessity.
				}
			}
		}
	}

	// Propagate value origins through dataflow ops (2 passes covers Phi back-edges).
	for pass := 0; pass < 2; pass++ {
		for _, blk := range fn.Blocks {
			for _, ins := range blk.Instrs {
				switch ins := ins.(type) {
				case *ssa.Phi:
					for _, e := range ins.Edges {
						propagate(ins, e.Val)
					}
				case *ssa.FieldAddr:
					propagate(ins, ins.Base)
				case *ssa.IndexAddr:
					propagate(ins, ins.Base)
				case *ssa.Convert:
					propagate(ins, ins.X)
				case *ssa.PtrCast:
					propagate(ins, ins.X)
				case *ssa.Extract:
					propagate(ins, ins.Tuple)
				case *ssa.CallDirect:
					if callee, ok := lookup(ins.Callee); ok {
						for idx, arg := range ins.Args {
							if idx < len(callee.Params) && callee.Params[idx]&exportdata.ParamToReturn != 0 {
								propagate(ins, arg)
							}
						}
					}
				}
			}
		}
	}

	// Evaluate escape sinks.
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.Return:
				for _, r := range ins.Results {
					addParamFlag(r, exportdata.ParamToReturn)
				}
			case *ssa.Store:
				// Storing into a global or escaping pointer leaks Val to heap.
				if _, isGlobal := ins.Addr.(*ssa.GlobalRef); isGlobal {
					addParamFlag(ins.Val, exportdata.ParamToHeap)
				} else if len(paramOrigin[ins.Addr]) > 0 {
					addParamFlag(ins.Val, exportdata.ParamToHeap)
				}
			case *ssa.WriteBarrier:
				addParamFlag(ins.Val, exportdata.ParamToHeap)
			case *ssa.MakeClosure:
				if ins.Env != nil {
					addParamFlag(ins.Env, exportdata.ParamToHeap)
				}
			case *ssa.Go:
				addParamFlag(ins.Callee, exportdata.ParamToHeap)
				for _, arg := range ins.Args {
					addParamFlag(arg, exportdata.ParamToHeap)
				}
			case *ssa.Defer:
				if ins.InLoop {
					addParamFlag(ins.Callee, exportdata.ParamToHeap)
					for _, arg := range ins.Args {
						addParamFlag(arg, exportdata.ParamToHeap)
					}
				}
			case *ssa.Panic:
				addParamFlag(ins.Val, exportdata.ParamToHeap)
			case *ssa.CallInterface:
				addParamFlag(ins.Iface, exportdata.ParamToHeap)
				for _, arg := range ins.Args {
					addParamFlag(arg, exportdata.ParamToHeap)
				}
			case *ssa.CallClosure:
				for _, arg := range ins.Args {
					addParamFlag(arg, exportdata.ParamToHeap)
				}
			case *ssa.CallCFunc:
				for _, arg := range ins.Args {
					addParamFlag(arg, exportdata.ParamToHeap)
				}
			case *ssa.CallDirect:
				callee, ok := lookup(ins.Callee)
				if !ok {
					// Unknown callee: fail-closed.
					for _, arg := range ins.Args {
						addParamFlag(arg, exportdata.ParamToHeap|exportdata.ParamToCall)
					}
					continue
				}
				for idx, arg := range ins.Args {
					if idx < len(callee.Params) {
						f := callee.Params[idx]
						if f&exportdata.ParamToHeap != 0 {
							addParamFlag(arg, exportdata.ParamToHeap)
						}
						if f != exportdata.ParamNoEscape {
							addParamFlag(arg, exportdata.ParamToCall)
						}
					}
				}
			}
		}
	}

	stackAllocs, heapAllocs := 0, 0
	for _, a := range allAllocs {
		if allocEscapes[a] {
			heapAllocs++
		} else {
			stackAllocs++
		}
	}

	return exportdata.FuncEscape{
		Name:        fn.Name,
		Params:      paramFlags,
		StackAllocs: stackAllocs,
		HeapAllocs:  heapAllocs,
	}
}

func funcEscapeEqual(a, b exportdata.FuncEscape) bool {
	if a.Name != b.Name || a.StackAllocs != b.StackAllocs || a.HeapAllocs != b.HeapAllocs || len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	return true
}

func appendUniqueInts(dst, src []int) []int {
	for _, v := range src {
		found := false
		for _, cur := range dst {
			if cur == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func appendUniqueAllocs(dst, src []*ssa.Alloc) []*ssa.Alloc {
	for _, v := range src {
		found := false
		for _, cur := range dst {
			if cur == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

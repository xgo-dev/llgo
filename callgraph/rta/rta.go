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

// Package rta computes the call graph and set of reachable functions and
// runtime types using Rapid Type Analysis (RTA) over github.com/xgo-dev/llgo/ssa,
// aligning 1:1 with golang.org/x/tools/go/callgraph/rta (pure Go 1.20+).
package rta

import (
	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llgo/types/typeutil"
)

// Result holds the results of Rapid Type Analysis, aligning 1:1 with
// golang.org/x/tools/go/callgraph/rta.Result.
type Result struct {
	CallGraph    *callgraph.Graph
	Reachable    map[*ssa.Function]struct{ AddrTaken bool }
	RuntimeTypes typeutil.Map
}

// Analyze performs Rapid Type Analysis, starting at the specified root
// functions. If buildCallGraph is true, Result.CallGraph will contain a call
// graph; otherwise, only the other fields are populated.
func Analyze(roots []*ssa.Function, buildCallGraph bool) *Result {
	r := &rta{
		result: &Result{
			Reachable: make(map[*ssa.Function]struct{ AddrTaken bool }),
		},
		addrTakenFuncsBySig: new(typeutil.Map),
		dynCallSites:        new(typeutil.Map),
		invokeSites:         new(typeutil.Map),
		concreteTypes:       new(typeutil.Map),
		interfaceTypes:      new(typeutil.Map),
	}
	if buildCallGraph {
		r.result.CallGraph = callgraph.New(nil)
	}
	for _, root := range roots {
		if root == nil {
			continue
		}
		if r.prog == nil && root.Prog != nil {
			r.prog = root.Prog
		}
		r.addReachable(root, false)
		if r.result.CallGraph != nil {
			callgraph.AddEdge(r.result.CallGraph.Root, nil, r.result.CallGraph.CreateNode(root))
		}
	}

	for len(r.worklist) > 0 {
		fn := r.worklist[len(r.worklist)-1]
		r.worklist = r.worklist[:len(r.worklist)-1]
		r.visitFunc(fn)
	}
	return r.result
}

type rta struct {
	result   *Result
	prog     *ssa.Program
	worklist []*ssa.Function

	addrTakenFuncsBySig *typeutil.Map // key: *types.Signature -> map[*ssa.Function]bool
	dynCallSites        *typeutil.Map // key: *types.Signature -> map[ssa.CallInstruction]bool
	invokeSites         *typeutil.Map // key: *types.Interface -> map[ssa.CallInstruction]bool
	concreteTypes       *typeutil.Map // key: types.Type -> bool
	interfaceTypes      *typeutil.Map // key: *types.Interface -> bool
}

func (r *rta) addEdge(site ssa.CallInstruction, callee *ssa.Function) {
	if callee == nil {
		return
	}
	r.addReachable(callee, false)
	if r.result.CallGraph != nil && site != nil && site.Parent() != nil {
		from := r.result.CallGraph.CreateNode(site.Parent())
		to := r.result.CallGraph.CreateNode(callee)
		callgraph.AddEdge(from, site, to)
	}
}

func (r *rta) addReachable(fn *ssa.Function, addrTaken bool) {
	if fn == nil {
		return
	}
	if r.prog == nil && fn.Prog != nil {
		r.prog = fn.Prog
	}
	prev, seen := r.result.Reachable[fn]
	if !seen {
		r.result.Reachable[fn] = struct{ AddrTaken bool }{AddrTaken: addrTaken}
		r.worklist = append(r.worklist, fn)
	} else if addrTaken && !prev.AddrTaken {
		prev.AddrTaken = true
		r.result.Reachable[fn] = prev
	}
	if addrTaken && fn.Signature != nil {
		r.addAddrTakenFunc(fn)
	}
}

func (r *rta) addAddrTakenFunc(fn *ssa.Function) {
	sig := fn.Signature
	set, _ := r.addrTakenFuncsBySig.At(sig).(map[*ssa.Function]bool)
	if set == nil {
		set = make(map[*ssa.Function]bool)
		r.addrTakenFuncsBySig.Set(sig, set)
	}
	if set[fn] {
		return
	}
	set[fn] = true
	if sites, _ := r.dynCallSites.At(sig).(map[ssa.CallInstruction]bool); sites != nil {
		for site := range sites {
			r.addEdge(site, fn)
		}
	}
}

func (r *rta) addDynCallSite(site ssa.CallInstruction, sig *types.Signature) {
	if sig == nil {
		return
	}
	sites, _ := r.dynCallSites.At(sig).(map[ssa.CallInstruction]bool)
	if sites == nil {
		sites = make(map[ssa.CallInstruction]bool)
		r.dynCallSites.Set(sig, sites)
	}
	if sites[site] {
		return
	}
	sites[site] = true
	if funcs, _ := r.addrTakenFuncsBySig.At(sig).(map[*ssa.Function]bool); funcs != nil {
		for fn := range funcs {
			r.addEdge(site, fn)
		}
	}
}

func (r *rta) addInvokeSite(site ssa.CallInstruction, iface *types.Interface) {
	if iface == nil {
		return
	}
	r.interfaceTypes.Set(iface, true)
	sites, _ := r.invokeSites.At(iface).(map[ssa.CallInstruction]bool)
	if sites == nil {
		sites = make(map[ssa.CallInstruction]bool)
		r.invokeSites.Set(iface, sites)
	}
	if sites[site] {
		return
	}
	sites[site] = true
	r.concreteTypes.Iterate(func(C types.Type, _ interface{}) {
		if types.Implements(C, iface) {
			r.resolveInvoke(site, C)
		}
	})
}

func (r *rta) addRuntimeType(T types.Type, skip bool) {
	if T == nil {
		return
	}
	if prev, ok := r.result.RuntimeTypes.At(T).(bool); ok && (!prev || skip) {
		return
	}
	r.result.RuntimeTypes.Set(T, skip)

	if _, isIface := T.Underlying().(*types.Interface); isIface {
		return
	}
	if r.concreteTypes.At(T) != nil {
		return
	}
	r.concreteTypes.Set(T, true)

	// Check every seen interface against this new concrete type.
	r.interfaceTypes.Iterate(func(k types.Type, _ interface{}) {
		iface, ok := k.(*types.Interface)
		if !ok || !types.Implements(T, iface) {
			return
		}
		if sites, _ := r.invokeSites.At(iface).(map[ssa.CallInstruction]bool); sites != nil {
			for site := range sites {
				r.resolveInvoke(site, T)
			}
		}
	})

	// If T is accessible to reflection, mark its exported methods reachable.
	if !skip && r.prog != nil {
		mset := r.prog.MethodSets.MethodSet(T)
		for i := 0; i < mset.Len(); i++ {
			sel := mset.At(i)
			if sel.Obj().Exported() {
				if fn := r.prog.MethodValue(sel); fn != nil {
					r.addReachable(fn, true)
				}
			}
		}
	}
}

func (r *rta) resolveInvoke(site ssa.CallInstruction, concrete types.Type) {
	if r.prog == nil || site == nil || site.Common().Method == nil {
		return
	}
	m := site.Common().Method
	if fn := r.prog.LookupMethod(concrete, m.Pkg(), m.Name()); fn != nil {
		r.addEdge(site, fn)
	}
}

func (r *rta) visitFunc(fn *ssa.Function) {
	var rands []*ssa.Value
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			// Track runtime types entering interfaces.
			switch ins := ins.(type) {
			case *ssa.MakeInterface:
				if ins.X != nil {
					r.addRuntimeType(ins.X.Type(), false)
				}
			case *ssa.MakeClosure:
				if target, ok := ins.Fn.(*ssa.Function); ok {
					r.addReachable(target, true)
				}
			}

			// Handle call sites (Call, Go, Defer).
			if site, ok := ins.(ssa.CallInstruction); ok {
				common := site.Common()
				if common.IsInvoke() {
					if common.Value != nil && common.Value.Type() != nil {
						if iface, ok := common.Value.Type().Underlying().(*types.Interface); ok {
							r.addInvokeSite(site, iface)
						}
					}
				} else if callee := common.StaticCallee(); callee != nil {
					r.addEdge(site, callee)
				} else if _, isBuiltin := common.Value.(*ssa.Builtin); !isBuiltin {
					r.addDynCallSite(site, common.Signature())
				}
			}

			// Any *ssa.Function referenced as a non-callee operand is address-taken.
			rands = ins.Operands(rands[:0])
			var staticCallee *ssa.Function
			if site, ok := ins.(ssa.CallInstruction); ok && !site.Common().IsInvoke() {
				staticCallee = site.Common().StaticCallee()
			}
			for _, opPtr := range rands {
				if opPtr == nil || *opPtr == nil {
					continue
				}
				if target, ok := (*opPtr).(*ssa.Function); ok && target != staticCallee {
					r.addReachable(target, true)
				}
			}
		}
	}
}

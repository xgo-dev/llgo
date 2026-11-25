// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file (LICENSES/Go-BSD-3-Clause.txt).

// Package rta provides Rapid Type Analysis (RTA) for Go over
// github.com/xgo-dev/llgo/ssa, aligning 1:1 with
// golang.org/x/tools/go/callgraph/rta in pure Go 1.20+.
package rta

import (
	"fmt"
	"hash/crc32"

	"github.com/xgo-dev/llgo/callgraph"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
	"github.com/xgo-dev/llgo/types/typeutil"
)

// A Result holds the results of Rapid Type Analysis, which includes the
// set of reachable functions/methods, runtime types, and the call graph.
type Result struct {
	CallGraph    *callgraph.Graph
	Reachable    map[*ssa.Function]struct{ AddrTaken bool }
	RuntimeTypes typeutil.Map
}

type rta struct {
	result           *Result
	prog             *ssa.Program
	reflectValueCall *ssa.Function
	worklist         []*ssa.Function

	addrTakenFuncsBySig typeutil.Map
	dynCallSites        typeutil.Map
	invokeSites         typeutil.Map

	concreteTypes typeutil.Map
	concreteList  []*concreteTypeInfo

	interfaceTypes typeutil.Map
	interfaceList  []*interfaceTypeInfo
}

type concreteTypeInfo struct {
	C          types.Type
	fprint     uint64
	implements []*types.Interface
}

type interfaceTypeInfo struct {
	I               *types.Interface
	fprint          uint64
	implementations []types.Type
}

func (r *rta) addReachable(f *ssa.Function, addrTaken bool) {
	if f == nil {
		return
	}
	reachable := r.result.Reachable
	n := len(reachable)
	v := reachable[f]
	if addrTaken {
		v.AddrTaken = true
	}
	reachable[f] = v
	if len(reachable) > n {
		r.worklist = append(r.worklist, f)
	}
}

func (r *rta) addEdge(caller *ssa.Function, site ssa.CallInstruction, callee *ssa.Function, addrTaken bool) {
	if callee == nil {
		return
	}
	r.addReachable(callee, addrTaken)
	if g := r.result.CallGraph; g != nil {
		if caller == nil {
			panic(site)
		}
		from := g.CreateNode(caller)
		to := g.CreateNode(callee)
		callgraph.AddEdge(from, site, to)
	}
}

func (r *rta) visitAddrTakenFunc(f *ssa.Function) {
	if f == nil || f.Signature == nil {
		return
	}
	S := f.Signature
	funcs, _ := r.addrTakenFuncsBySig.At(S).(map[*ssa.Function]bool)
	if funcs == nil {
		funcs = make(map[*ssa.Function]bool)
		r.addrTakenFuncsBySig.Set(S, funcs)
	}
	if !funcs[f] {
		funcs[f] = true
		sites, _ := r.dynCallSites.At(S).([]ssa.CallInstruction)
		for _, site := range sites {
			r.addEdge(site.Parent(), site, f, true)
		}
		if r.reflectValueCall != nil {
			var site ssa.CallInstruction
			r.addEdge(r.reflectValueCall, site, f, true)
		}
	}
}

func (r *rta) visitDynCall(site ssa.CallInstruction) {
	S := site.Common().Signature()
	if S == nil {
		return
	}
	sites, _ := r.dynCallSites.At(S).([]ssa.CallInstruction)
	r.dynCallSites.Set(S, append(sites, site))
	funcs, _ := r.addrTakenFuncsBySig.At(S).(map[*ssa.Function]bool)
	for g := range funcs {
		r.addEdge(site.Parent(), site, g, true)
	}
}

func (r *rta) addInvokeEdge(site ssa.CallInstruction, C types.Type) {
	if r.prog == nil {
		return
	}
	imethod := site.Common().Method
	if cmethod := r.prog.LookupMethod(C, imethod.Pkg(), imethod.Name()); cmethod != nil {
		r.addEdge(site.Parent(), site, cmethod, true)
	}
}

func (r *rta) visitInvoke(site ssa.CallInstruction) {
	I := site.Common().Value.Type().Underlying().(*types.Interface)
	sites, _ := r.invokeSites.At(I).([]ssa.CallInstruction)
	r.invokeSites.Set(I, append(sites, site))
	for _, C := range r.implementations(I) {
		r.addInvokeEdge(site, C)
	}
}

func (r *rta) visitFunc(f *ssa.Function) {
	var space [32]*ssa.Value
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			rands := instr.Operands(space[:0])

			switch instr := instr.(type) {
			case ssa.CallInstruction:
				call := instr.Common()
				if call.IsInvoke() {
					r.visitInvoke(instr)
				} else if g := call.StaticCallee(); g != nil {
					r.addEdge(f, instr, g, false)
				} else if _, ok := call.Value.(*ssa.Builtin); !ok {
					r.visitDynCall(instr)
				}
				if len(rands) > 0 {
					rands = rands[1:]
				}
			case *ssa.MakeInterface:
				r.addRuntimeType(instr.X.Type())
			}

			for _, op := range rands {
				if op != nil {
					if g, ok := (*op).(*ssa.Function); ok {
						r.visitAddrTakenFunc(g)
					}
				}
			}
		}
	}
}

// Analyze performs Rapid Type Analysis, starting at the specified root
// functions. It returns nil if no roots were specified.
func Analyze(roots []*ssa.Function, buildCallGraph bool) *Result {
	if len(roots) == 0 {
		return nil
	}

	r := &rta{
		result: &Result{Reachable: make(map[*ssa.Function]struct{ AddrTaken bool })},
		prog:   roots[0].Prog,
	}

	if buildCallGraph {
		r.result.CallGraph = callgraph.New(roots[0])
	}

	if r.prog != nil {
		if reflectPkg := r.prog.ImportedPackage("reflect"); reflectPkg != nil {
			if reflectValue, ok := reflectPkg.Members["Value"].(*ssa.Type); ok {
				r.reflectValueCall = r.prog.LookupMethod(reflectValue.Object().Type(), reflectPkg.Pkg, "Call")
			}
		}
	}

	hasher := typeutil.MakeHasher()
	r.result.RuntimeTypes.SetHasher(hasher)
	r.addrTakenFuncsBySig.SetHasher(hasher)
	r.dynCallSites.SetHasher(hasher)
	r.invokeSites.SetHasher(hasher)
	r.concreteTypes.SetHasher(hasher)
	r.interfaceTypes.SetHasher(hasher)

	for _, root := range roots {
		r.addReachable(root, false)
	}

	var shadow []*ssa.Function
	for len(r.worklist) > 0 {
		shadow, r.worklist = r.worklist, shadow[:0]
		for _, f := range shadow {
			r.visitFunc(f)
		}
	}
	return r.result
}

func (r *rta) interfaces(C types.Type) []*types.Interface {
	var cinfo *concreteTypeInfo
	if v := r.concreteTypes.At(C); v != nil {
		cinfo = v.(*concreteTypeInfo)
	} else {
		cinfo = &concreteTypeInfo{
			C:      C,
			fprint: fingerprint(r.prog.MethodSets.MethodSet(C)),
		}
		r.concreteTypes.Set(C, cinfo)
		r.concreteList = append(r.concreteList, cinfo)

		for _, iinfo := range r.interfaceList {
			if implements(cinfo, iinfo) {
				iinfo.implementations = append(iinfo.implementations, C)
				cinfo.implements = append(cinfo.implements, iinfo.I)
			}
		}
	}
	return cinfo.implements
}

func (r *rta) implementations(I *types.Interface) []types.Type {
	var iinfo *interfaceTypeInfo
	if v := r.interfaceTypes.At(I); v != nil {
		iinfo = v.(*interfaceTypeInfo)
	} else {
		iinfo = &interfaceTypeInfo{
			I:      I,
			fprint: fingerprint(r.prog.MethodSets.MethodSet(I)),
		}
		r.interfaceTypes.Set(I, iinfo)
		r.interfaceList = append(r.interfaceList, iinfo)

		for _, cinfo := range r.concreteList {
			if implements(cinfo, iinfo) {
				cinfo.implements = append(cinfo.implements, I)
				iinfo.implementations = append(iinfo.implementations, cinfo.C)
			}
		}
	}
	return iinfo.implementations
}

func (r *rta) addRuntimeType(T types.Type) {
	if r.prog == nil || T == nil {
		return
	}
	methodSetOf := r.prog.MethodSets.MethodSet
	forEachElement(methodSetOf, T, func(T types.Type, access bool) bool {
		if prevInaccess, ok := r.result.RuntimeTypes.At(T).(bool); ok {
			if prevInaccess && access {
				r.result.RuntimeTypes.Set(T, false)
			}
			return true
		}
		r.result.RuntimeTypes.Set(T, !access)

		if !types.IsInterface(T) {
			mset := methodSetOf(T)
			for i := 0; i < mset.Len(); i++ {
				sel := mset.At(i)
				obj := sel.Obj()
				if !obj.Exported() {
					continue
				}
				if obj.Type().(*types.Signature).TypeParams() != nil {
					continue
				}
				r.addReachable(r.prog.MethodValue(sel), true)
			}

			for _, I := range r.interfaces(T) {
				sites, _ := r.invokeSites.At(I).([]ssa.CallInstruction)
				for _, site := range sites {
					r.addInvokeEdge(site, T)
				}
			}
		}

		return false
	})
}

func forEachElement(methodSetOf func(types.Type) *types.MethodSet, T types.Type, f func(T types.Type, access bool) bool) {
	var visit func(T types.Type, access bool)
	visit = func(T types.Type, access bool) {
		if f(T, access) {
			return
		}
		tmset := methodSetOf(T)
		for i := 0; i < tmset.Len(); i++ {
			method := tmset.At(i)
			sig := method.Type().(*types.Signature)
			if sig.TypeParams() != nil {
				continue
			}
			visit(sig.Params(), false)
			visit(sig.Results(), false)
		}
		switch T := T.(type) {
		case *types.Alias:
			visit(types.Unalias(T), access)
		case *types.Basic, *types.SIMDVec:
			// nop
		case *types.Interface:
			// nop---handled by recursion over method set.
		case *types.Pointer:
			visit(T.Elem(), true)
		case *types.Slice:
			visit(T.Elem(), true)
		case *types.Chan:
			visit(T.Elem(), true)
		case *types.Map:
			visit(T.Key(), true)
			visit(T.Elem(), true)
		case *types.Signature:
			if T.Recv() != nil {
				panic(fmt.Sprintf("Signature %s has Recv %s", T, T.Recv()))
			}
			visit(T.Params(), false)
			visit(T.Results(), false)
		case *types.Named:
			visit(types.NewPointer(T), true)
			visit(T.Underlying(), false)
		case *types.Array:
			visit(T.Elem(), true)
		case *types.Struct:
			for i, n := 0, T.NumFields(); i < n; i++ {
				visit(T.Field(i).Type(), true)
			}
		case *types.Tuple:
			for i, n := 0, T.Len(); i < n; i++ {
				visit(T.At(i).Type(), true)
			}
		case *types.TypeParam, *types.Union:
			panic(fmt.Sprintf("forEachElement called on type containing %T", T))
		default:
			panic(fmt.Sprintf("forEachElement called on unexpected type %T", T))
		}
	}
	visit(T, true)
}

func fingerprint(mset *types.MethodSet) uint64 {
	var space [64]byte
	var mask uint64
	for i := 0; i < mset.Len(); i++ {
		method := mset.At(i).Obj()
		sig := method.Type().(*types.Signature)
		if sig.TypeParams() != nil {
			continue
		}
		sum := crc32.ChecksumIEEE(fmt.Appendf(space[:0], "%s/%d/%d",
			method.Id(),
			sig.Params().Len(),
			sig.Results().Len()))
		mask |= 1 << (sum % 64)
	}
	return mask
}

func implements(cinfo *concreteTypeInfo, iinfo *interfaceTypeInfo) (got bool) {
	return iinfo.fprint & ^cinfo.fprint == 0 && types.Implements(cinfo.C, iinfo.I)
}

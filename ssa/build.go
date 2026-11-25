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

package ssa

import (
	"go/token"

	"github.com/xgo-dev/llgo/types"
)

// NewBuiltin creates a Builtin value with name and signature.
func NewBuiltin(name string, sig *types.Signature) *Builtin {
	return &Builtin{name: name, sig: sig}
}

// AddGlobal adds a package-level Global variable to p and registers it in p.Members.
func (p *Package) AddGlobal(name string, obj *types.Var, ptrType types.Type) *Global {
	var pos token.Pos
	if obj != nil {
		pos = obj.Pos()
	}
	g := &Global{
		Pkg:    p,
		name:   name,
		object: obj,
		typ:    ptrType,
		pos:    pos,
	}
	p.Members[name] = g
	return g
}

// AddFunction adds a Function to p (and p.Prog maps if present).
func (p *Package) AddFunction(fn *Function, obj *types.Func) {
	if fn == nil {
		return
	}
	fn.Pkg = p
	if p != nil {
		fn.Prog = p.Prog
	}
	if obj != nil {
		fn.object = obj
		fn.pos = obj.Pos()
	}
	if p != nil && fn.parent == nil && (fn.Signature == nil || fn.Signature.Recv() == nil) {
		p.Members[fn.name] = fn
	}
	if p != nil && p.Prog != nil {
		if obj != nil {
			p.Prog.funcsByObj[obj] = fn
		}
		if fn.Signature != nil && fn.Signature.Recv() != nil {
			var pkg *types.Package
			if p.Pkg != nil {
				pkg = p.Pkg
			}
			p.Prog.methods[methodKey(fn.Signature.Recv().Type(), pkg, fn.name)] = fn
		}
	}
}

// AddType adds a Type member to p.Members.
func (p *Package) AddType(obj *types.TypeName) *Type {
	if p == nil || obj == nil {
		return nil
	}
	t := &Type{pkg: p, object: obj}
	p.Members[obj.Name()] = t
	return t
}

// AddConst adds a NamedConst member to p.Members.
func (p *Package) AddConst(obj *types.Const, val *Const) *NamedConst {
	if p == nil || obj == nil {
		return nil
	}
	c := &NamedConst{pkg: p, object: obj, Value: val}
	p.Members[obj.Name()] = c
	return c
}

// RegisterMethod associates a method implementation fn with concrete receiver type T.
func (prog *Program) RegisterMethod(T types.Type, pkg *types.Package, name string, fn *Function) {
	if prog == nil || T == nil || fn == nil {
		return
	}
	prog.methods[methodKey(T, pkg, name)] = fn
}

// AddRuntimeType records a concrete type that may be stored into an interface at runtime.
func (prog *Program) AddRuntimeType(T types.Type) {
	if prog == nil || T == nil {
		return
	}
	prog.runtimeTypes.Set(T, true)
}

// NewBasicBlock appends a new BasicBlock to f.
func (f *Function) NewBasicBlock(comment string) *BasicBlock {
	b := &BasicBlock{
		Index:   len(f.Blocks),
		Comment: comment,
		parent:  f,
	}
	f.Blocks = append(f.Blocks, b)
	return b
}

// AddParam appends a formal Parameter to f.Params.
func (f *Function) AddParam(name string, obj types.Object, typ types.Type) *Parameter {
	p := &Parameter{
		name:   name,
		object: obj,
		typ:    typ,
		parent: f,
	}
	f.Params = append(f.Params, p)
	return p
}

// AddFreeVar appends a captured FreeVar to f.FreeVars.
func (f *Function) AddFreeVar(name string, typ types.Type, pos token.Pos) *FreeVar {
	fv := &FreeVar{
		name:   name,
		typ:    typ,
		pos:    pos,
		parent: f,
	}
	f.FreeVars = append(f.FreeVars, fv)
	return fv
}

// AddAnonFunc appends a nested anonymous Function to f.AnonFuncs.
func (f *Function) AddAnonFunc(anon *Function) {
	if f == nil || anon == nil {
		return
	}
	anon.parent = f
	anon.Pkg = f.Pkg
	anon.Prog = f.Prog
	f.AnonFuncs = append(f.AnonFuncs, anon)
}

// AddEdge adds a directed CFG edge b -> succ, updating both b.Succs and succ.Preds.
func (b *BasicBlock) AddEdge(succ *BasicBlock) {
	if b == nil || succ == nil {
		return
	}
	b.Succs = append(b.Succs, succ)
	succ.Preds = append(succ.Preds, b)
}

// Emit appends ins to b.Instrs, assigning its block, source position, register
// type, and virtual register name.
func (b *BasicBlock) Emit(ins Instruction, typ types.Type, pos token.Pos) {
	if s, ok := ins.(instrSetter); ok {
		s.setBlock(b)
		s.setPos(pos)
	}
	if r, ok := ins.(regSetter); ok {
		r.setType(typ)
		if b.parent != nil {
			r.setNum(b.parent.nextReg)
			b.parent.nextReg++
		}
	}
	if a, ok := ins.(*Alloc); ok && !a.Heap && b.parent != nil {
		b.parent.Locals = append(b.parent.Locals, a)
	}
	if c, ok := ins.(CallInstruction); ok {
		c.Common().pos = pos
	}
	b.Instrs = append(b.Instrs, ins)
}

// Finish recomputes value Referrers and immediate dominators for f after all
// blocks, edges, and Phi operands have been populated.
func (f *Function) Finish() {
	if f == nil || len(f.Blocks) == 0 {
		return
	}
	// Reset Referrers.
	for _, p := range f.Params {
		p.referrers = p.referrers[:0]
	}
	for _, fv := range f.FreeVars {
		fv.referrers = fv.referrers[:0]
	}
	for _, anon := range f.AnonFuncs {
		anon.referrers = anon.referrers[:0]
	}
	for _, b := range f.Blocks {
		for _, ins := range b.Instrs {
			if v, ok := ins.(Value); ok {
				if refs := v.Referrers(); refs != nil {
					*refs = (*refs)[:0]
				}
			}
		}
	}
	// Populate Referrers from every instruction's Operands.
	var rands []*Value
	for _, b := range f.Blocks {
		for _, ins := range b.Instrs {
			rands = ins.Operands(rands[:0])
			for _, op := range rands {
				if op == nil || *op == nil {
					continue
				}
				if refs := (*op).Referrers(); refs != nil {
					*refs = append(*refs, ins)
				}
			}
		}
	}
	computeDominators(f)
}

// computeDominators computes immediate dominators (Cooper-Harvey-Kennedy) and
// preorder/postorder DFS intervals for O(1) BasicBlock.Dominates queries.
func computeDominators(f *Function) {
	for _, b := range f.Blocks {
		b.idom = nil
		b.dominees = nil
		b.domPre = 0
		b.domPost = 0
	}
	if len(f.Blocks) == 0 {
		return
	}

	// Compute reverse postorder (RPO) from entry block f.Blocks[0].
	visited := make(map[*BasicBlock]bool, len(f.Blocks))
	var post []*BasicBlock
	var dfs func(*BasicBlock)
	dfs = func(b *BasicBlock) {
		visited[b] = true
		for _, s := range b.Succs {
			if !visited[s] {
				dfs(s)
			}
		}
		post = append(post, b)
	}
	dfs(f.Blocks[0])

	n := len(post)
	rpo := make([]*BasicBlock, n)
	rpoNum := make(map[*BasicBlock]int, n)
	for i := 0; i < n; i++ {
		b := post[n-1-i]
		rpo[i] = b
		rpoNum[b] = i
	}

	idom := make([]int, n)
	for i := range idom {
		idom[i] = -1
	}
	idom[0] = 0

	intersect := func(a, b int) int {
		for a != b {
			for a > b {
				a = idom[a]
			}
			for b > a {
				b = idom[b]
			}
		}
		return a
	}

	changed := true
	for changed {
		changed = false
		for i := 1; i < n; i++ {
			b := rpo[i]
			newIdom := -1
			for _, p := range b.Preds {
				pIdx, ok := rpoNum[p]
				if !ok || idom[pIdx] == -1 {
					continue
				}
				if newIdom == -1 {
					newIdom = pIdx
				} else {
					newIdom = intersect(pIdx, newIdom)
				}
			}
			if newIdom != -1 && idom[i] != newIdom {
				idom[i] = newIdom
				changed = true
			}
		}
	}

	for i := 1; i < n; i++ {
		if idom[i] >= 0 {
			parent := rpo[idom[i]]
			child := rpo[i]
			child.idom = parent
			parent.dominees = append(parent.dominees, child)
		}
	}

	// Assign Euler tour timestamps for O(1) Dominates.
	timer := 0
	var tour func(*BasicBlock)
	tour = func(b *BasicBlock) {
		timer++
		b.domPre = timer
		for _, child := range b.dominees {
			tour(child)
		}
		b.domPost = timer
	}
	tour(rpo[0])
}

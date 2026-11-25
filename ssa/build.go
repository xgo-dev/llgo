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
	if obj != nil && p.objects != nil {
		p.objects[obj] = g
	}
	return g
}

// AddFunction adds a Function to p (and p.Prog method/object tables if present).
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
		if p != nil && p.objects != nil {
			p.objects[obj] = fn
		}
	}
	if p != nil && fn.parent == nil && (fn.Signature == nil || fn.Signature.Recv() == nil) {
		p.Members[fn.name] = fn
	}
	if p != nil && p.Prog != nil && fn.Signature != nil && fn.Signature.Recv() != nil {
		var pkg *types.Package
		if obj != nil {
			pkg = obj.Pkg()
		} else if p.Pkg != nil {
			pkg = p.Pkg
		}
		p.Prog.RegisterMethod(fn.Signature.Recv().Type(), pkg, fn.name, fn)
	}
}

// AddType adds a Type member to p.Members.
func (p *Package) AddType(obj *types.TypeName) *Type {
	if p == nil || obj == nil {
		return nil
	}
	t := &Type{pkg: p, object: obj}
	p.Members[obj.Name()] = t
	if p.objects != nil {
		p.objects[obj] = t
	}
	return t
}

// AddConst adds a NamedConst member to p.Members.
func (p *Package) AddConst(obj *types.Const, val *Const) *NamedConst {
	if p == nil || obj == nil {
		return nil
	}
	c := &NamedConst{pkg: p, object: obj, Value: val}
	p.Members[obj.Name()] = c
	if p.objects != nil {
		p.objects[obj] = c
	}
	return c
}

// RegisterMethod associates a method implementation fn with concrete receiver type T.
func (prog *Program) RegisterMethod(T types.Type, pkg *types.Package, name string, fn *Function) {
	if prog == nil || T == nil || fn == nil {
		return
	}
	key := methodKeyOf(pkg, name)
	prog.methodsMu.Lock()
	defer prog.methodsMu.Unlock()
	mset, ok := prog.methodSets.At(T).(*methodSet)
	if !ok {
		mset = &methodSet{mapping: make(map[methodKey]*Function)}
		prog.methodSets.Set(T, mset)
	}
	mset.mapping[key] = fn
}

// AddRuntimeType records a concrete type that may be stored into an interface at runtime.
func (prog *Program) AddRuntimeType(T types.Type) {
	if prog == nil || T == nil {
		return
	}
	prog.makeInterfaceTypesMu.Lock()
	if prog.makeInterfaceTypes == nil {
		prog.makeInterfaceTypes = make(map[types.Type]unit)
	}
	prog.makeInterfaceTypes[T] = unit{}
	prog.makeInterfaceTypesMu.Unlock()
}

// NewBasicBlock appends a new BasicBlock to f.
func (f *Function) NewBasicBlock(comment string) *BasicBlock {
	b := &BasicBlock{
		Index:   len(f.Blocks),
		Comment: comment,
		parent:  f,
	}
	b.Succs = b.succs2[:0]
	f.Blocks = append(f.Blocks, b)
	return b
}

// AddParam appends a formal Parameter to f.Params.
func (f *Function) AddParam(name string, obj types.Object, typ types.Type) *Parameter {
	v, _ := obj.(*types.Var)
	if v == nil {
		v = types.NewParam(token.NoPos, nil, name, typ)
	}
	p := &Parameter{
		name:   name,
		object: v,
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
	anon.anonIdx = int32(len(f.AnonFuncs))
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
	ins.setBlock(b)
	setInstrPos(ins, pos)
	if v, ok := ins.(interface {
		setType(types.Type)
		setNum(int)
	}); ok {
		v.setType(typ)
		v.setNum(len(b.Instrs))
	}
	if a, ok := ins.(*Alloc); ok && !a.Heap && b.parent != nil {
		b.parent.Locals = append(b.parent.Locals, a)
	}
	if c, ok := ins.(CallInstruction); ok && c.Common().pos == token.NoPos {
		c.Common().pos = pos
	}
	b.Instrs = append(b.Instrs, ins)
}

func setInstrPos(ins Instruction, pos token.Pos) {
	if r, ok := ins.(interface{ setPos(token.Pos) }); ok {
		r.setPos(pos)
		return
	}
	switch i := ins.(type) {
	case *Store:
		i.pos = pos
	case *Return:
		i.pos = pos
	case *Panic:
		i.pos = pos
	case *Send:
		i.pos = pos
	case *Go:
		i.pos = pos
	case *Defer:
		i.pos = pos
	case *MapUpdate:
		i.pos = pos
	case *DebugRef:
		// DebugRef.Pos() comes from Expr.Pos().
	}
}

// Finish recomputes register numbering, value Referrers, and immediate
// dominators for f after all blocks, edges, and Phi operands have been populated.
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
	numberRegisters(f)
	buildReferrers(f)
	buildDomTree(f)
}

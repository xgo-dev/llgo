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
	"fmt"
	"go/constant"
	"go/token"

	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

// NewPackage creates a new pure-Go IR Package for spec.
func NewPackage(path, name string, spec target.Spec) *Package {
	if spec.WordSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	return &Package{
		Path:    path,
		Name:    name,
		Target:  spec,
		Exports: make(map[string]string),
	}
}

// NewGlobal adds and returns a package-level Global.
func (p *Package) NewGlobal(name string, typ types.Type) *Global {
	g := &Global{
		Name:       name,
		Type:       typ,
		HasPtrData: typ != nil && typ.HasPtrData(),
	}
	p.Globals = append(p.Globals, g)
	return g
}

// NewFunction adds and returns a Function with formal parameters derived from sig.
func (p *Package) NewFunction(name string, sig *types.Signature) *Function {
	fn := &Function{
		Pkg:    p,
		Name:   name,
		Sig:    sig,
		HasEnv: sig != nil && sig.IsClosure(),
	}
	if sig != nil {
		idx := 0
		if recv := sig.Recv(); recv != nil {
			fn.Params = append(fn.Params, &Parameter{
				baseValue: baseValue{id: fn.allocID(), typ: recv.Type()},
				ParamName: recv.Name(),
				Index:     idx,
			})
			idx++
		}
		params := sig.Params()
		for i := 0; i < params.Len(); i++ {
			param := params.At(i)
			fn.Params = append(fn.Params, &Parameter{
				baseValue: baseValue{id: fn.allocID(), typ: param.Type()},
				ParamName: param.Name(),
				Index:     idx,
			})
			idx++
		}
	}
	p.Funcs = append(p.Funcs, fn)
	return fn
}

func (f *Function) allocID() ValueID {
	f.nextID++
	return f.nextID
}

// NewBasicBlock appends a new basic block to f.
func (f *Function) NewBasicBlock(label string) *BasicBlock {
	idx := len(f.Blocks)
	if label == "" {
		label = fmt.Sprintf("bb%d", idx)
	}
	blk := &BasicBlock{Func: f, Index: idx, Label: label}
	f.Blocks = append(f.Blocks, blk)
	return blk
}

// Builder emits instructions into a Function's basic blocks.
type Builder struct {
	fn  *Function
	blk *BasicBlock
	pos token.Pos
}

// NewBuilder creates a Builder positioned at blk.
func NewBuilder(blk *BasicBlock) *Builder {
	return &Builder{fn: blk.Func, blk: blk}
}

// SetBlock positions the builder at the end of blk.
func (b *Builder) SetBlock(blk *BasicBlock) {
	b.fn = blk.Func
	b.blk = blk
}

// SetPos sets the source position stamped onto subsequently emitted instructions.
func (b *Builder) SetPos(pos token.Pos) {
	b.pos = pos
}

func (b *Builder) emit(ins Instruction) {
	ins.setBlock(b.blk)
	if b.pos.IsValid() {
		ins.setPos(b.pos)
	}
	b.blk.Instrs = append(b.blk.Instrs, ins)
}

func (b *Builder) Const(typ types.Type, val constant.Value) *Const {
	return &Const{baseValue: baseValue{id: b.fn.allocID(), typ: typ}, Value: val}
}

func (b *Builder) Extract(tuple Value, index int) *Extract {
	var elemType types.Type
	if tuple != nil && tuple.Type() != nil {
		if tup, ok := tuple.Type().(*types.Tuple); ok && index >= 0 && index < tup.Len() {
			elemType = tup.At(index).Type()
		}
	}
	ins := &Extract{baseValue: baseValue{id: b.fn.allocID(), typ: elemType}, Tuple: tuple, Index: index}
	b.emit(ins)
	return ins
}

func (b *Builder) DebugRef(x Value, obj types.Object, isAddr bool) *DebugRef {
	ins := &DebugRef{X: x, Object: obj, IsAddr: isAddr}
	b.emit(ins)
	return ins
}

func (b *Builder) Phi(typ types.Type, edges ...PhiEdge) *Phi {
	ins := &Phi{baseValue: baseValue{id: b.fn.allocID(), typ: typ}, Edges: edges}
	b.emit(ins)
	return ins
}

func (b *Builder) BinOp(op token.Token, typ types.Type, x, y Value) *BinOp {
	ins := &BinOp{baseValue: baseValue{id: b.fn.allocID(), typ: typ}, Op: op, X: x, Y: y}
	b.emit(ins)
	return ins
}

func (b *Builder) Alloc(elem types.Type, heap, noScan bool) *Alloc {
	ins := &Alloc{
		baseValue: baseValue{id: b.fn.allocID(), typ: types.NewPointerWithKind(elem, types.GoPtr)},
		Elem:      elem,
		Heap:      heap,
		NoScan:    noScan,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) Load(addr Value, elem types.Type) *Load {
	ins := &Load{baseValue: baseValue{id: b.fn.allocID(), typ: elem}, Addr: addr}
	b.emit(ins)
	return ins
}

func (b *Builder) Store(addr, val Value) *Store {
	ins := &Store{Addr: addr, Val: val}
	b.emit(ins)
	return ins
}

func (b *Builder) WriteBarrier(dst, val Value, kind BarrierKind) *WriteBarrier {
	ins := &WriteBarrier{Dst: dst, Val: val, Kind: kind}
	b.emit(ins)
	return ins
}

func (b *Builder) Safepoint(roots ...Value) *Safepoint {
	ins := &Safepoint{LiveRoots: roots}
	b.emit(ins)
	return ins
}

func (b *Builder) GCRoot(slot Value, live bool) *GCRoot {
	ins := &GCRoot{Slot: slot, Live: live}
	b.emit(ins)
	return ins
}

func (b *Builder) KeepAlive(val Value) *KeepAlive {
	ins := &KeepAlive{Val: val}
	b.emit(ins)
	return ins
}

func (b *Builder) PtrCast(x Value, targetPtr *types.Pointer) *PtrCast {
	ins := &PtrCast{baseValue: baseValue{id: b.fn.allocID(), typ: targetPtr}, X: x}
	b.emit(ins)
	return ins
}

func (b *Builder) MakeClosure(sig *types.Signature, fnName string, env Value) *MakeClosure {
	cloSig := sig
	if sig != nil && !sig.IsClosure() {
		cloSig = sig.WithCallConv(types.Go)
	}
	ins := &MakeClosure{
		baseValue: baseValue{id: b.fn.allocID(), typ: cloSig},
		FnName:    fnName,
		Env:       env,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) CallDirect(callee string, cc types.CallConv, ret types.Type, args ...Value) *CallDirect {
	ins := &CallDirect{
		baseValue: baseValue{id: b.fn.allocID(), typ: ret},
		Callee:    callee,
		CallConv:  cc,
		Args:      args,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) CallClosure(closure Value, ret types.Type, args ...Value) *CallClosure {
	ins := &CallClosure{
		baseValue: baseValue{id: b.fn.allocID(), typ: ret},
		Closure:   closure,
		Args:      args,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) CallCFunc(callee Value, cc types.CallConv, ret types.Type, args ...Value) *CallCFunc {
	ins := &CallCFunc{
		baseValue: baseValue{id: b.fn.allocID(), typ: ret},
		Callee:    callee,
		CallConv:  cc,
		Args:      args,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) CallInterface(iface Value, method string, methodIndex int, ret types.Type, args ...Value) *CallInterface {
	ins := &CallInterface{
		baseValue:   baseValue{id: b.fn.allocID(), typ: ret},
		Iface:       iface,
		Method:      method,
		MethodIndex: methodIndex,
		Args:        args,
	}
	b.emit(ins)
	return ins
}

func (b *Builder) Defer(callee Value, inLoop bool, args ...Value) *Defer {
	ins := &Defer{Callee: callee, Args: args, InLoop: inLoop}
	b.emit(ins)
	return ins
}

func (b *Builder) RunDefers() *RunDefers {
	ins := &RunDefers{}
	b.emit(ins)
	return ins
}

func (b *Builder) Jump(target *BasicBlock) *Jump {
	ins := &Jump{Target: target}
	b.emit(ins)
	return ins
}

func (b *Builder) Branch(cond Value, trueBlk, falseBlk *BasicBlock) *Branch {
	ins := &Branch{Cond: cond, True: trueBlk, False: falseBlk}
	b.emit(ins)
	return ins
}

func (b *Builder) Return(results ...Value) *Return {
	ins := &Return{Results: results}
	b.emit(ins)
	return ins
}

func (b *Builder) Unreachable() *Unreachable {
	ins := &Unreachable{}
	b.emit(ins)
	return ins
}

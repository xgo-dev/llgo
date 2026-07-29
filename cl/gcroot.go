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

package cl

import (
	"go/token"
	"go/types"

	"github.com/goplus/llgo/internal/gcrootplan"
	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func (p *context) prepareGCRoots(fn *ssa.Function, hasClosureContext bool) {
	p.gcRoots = nil
	p.gcClosureRoot = llssa.Nil
	if !p.prog.GCRootsEnabled() {
		return
	}

	planned := gcrootplan.Plan(fn, func(value ssa.Value) bool {
		switch value.(type) {
		case *ssa.FreeVar:
			return false
		}
		typ := p.type_(value.Type(), llssa.InGo)
		return p.prog.GCRootCount(typ) != 0
	}, p.isGCSafepoint)
	if p.safepointEntry {
		for _, param := range fn.Params {
			typ := p.type_(param.Type(), llssa.InGo)
			if p.prog.GCRootCount(typ) != 0 {
				planned[param] = struct{}{}
			}
		}
	}
	counts := make(map[ssa.Value]int, len(planned))
	total := 0
	count := func(value ssa.Value) {
		if _, ok := planned[value]; !ok {
			return
		}
		typ := p.type_(value.Type(), llssa.InGo)
		if n := p.prog.GCRootCount(typ); n != 0 {
			counts[value] = n
			total += n
		}
	}
	for _, param := range fn.Params {
		count(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				count(value)
			}
		}
	}
	hasClosureRoot := hasClosureContext && p.functionHasGCSafepoint(fn)
	if hasClosureRoot {
		total++
	}
	allSlots := p.fn.NewGCRoots(total)
	next := 0
	roots := make(map[ssa.Value][]llssa.Expr, len(counts))
	assign := func(value ssa.Value) {
		if n := counts[value]; n != 0 {
			roots[value] = allSlots[next : next+n]
			next += n
		}
	}
	for _, param := range fn.Params {
		assign(param)
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				assign(value)
			}
		}
	}
	p.gcRoots = roots
	if hasClosureRoot {
		p.gcClosureRoot = allSlots[next]
	}
}

func (p *context) initGCRoots(b llssa.Builder, fn *ssa.Function) {
	if len(p.gcRoots) == 0 && p.gcClosureRoot.IsNil() {
		return
	}
	b.SetBlockEx(p.fn.Block(0), llssa.AtEnd, true)
	for i, param := range fn.Params {
		if _, ok := p.gcRoots[param]; ok {
			p.publishGCRoot(b, param, b.Param(i))
		}
	}
	if !p.gcClosureRoot.IsNil() {
		b.SetGCRoot(p.gcClosureRoot, p.fn.ClosureContextParam())
	}
}

func functionHasGCSafepoint(fn *ssa.Function) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if gcSafepoint(instr) {
				return true
			}
		}
	}
	return false
}

func (p *context) functionHasGCSafepoint(fn *ssa.Function) bool {
	if p.safepointEntry {
		return true
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if p.isGCSafepoint(instr) {
				return true
			}
		}
	}
	return false
}

func (p *context) isGCSafepoint(instr ssa.Instruction) bool {
	return gcSafepoint(instr) || p.isCooperativeSafepoint(instr)
}

// gcSafepoint mirrors the operations whose LLGo lowering can call the runtime.
// Unknown instructions stay conservative.
func gcSafepoint(instr ssa.Instruction) bool {
	switch instr := instr.(type) {
	case *ssa.Phi, *ssa.DebugRef, *ssa.Extract, *ssa.Field, *ssa.FieldAddr,
		*ssa.Index, *ssa.IndexAddr, *ssa.If, *ssa.Jump, *ssa.Return,
		*ssa.Slice, *ssa.SliceToArrayPointer, *ssa.Store, *ssa.ChangeType:
		return false
	case *ssa.BinOp:
		return gcBinOpSafepoint(instr)
	case *ssa.UnOp:
		return instr.Op == token.ARROW
	case *ssa.Convert:
		return gcConversionSafepoint(instr.X.Type(), instr.Type())
	case *ssa.Call:
		if builtin, ok := instr.Call.Value.(*ssa.Builtin); ok {
			switch builtin.Name() {
			case "cap", "complex", "imag", "len", "real":
				return false
			}
		}
		return true
	default:
		return true
	}
}

func gcBinOpSafepoint(instr *ssa.BinOp) bool {
	switch basicKind(instr.X.Type()) {
	case types.String, types.UntypedString:
		return true
	}
	_, isInterface := types.Unalias(instr.X.Type()).Underlying().(*types.Interface)
	return isInterface
}

func gcConversionSafepoint(src, dst types.Type) bool {
	return isStringOrSlice(src) || isStringOrSlice(dst)
}

func isStringOrSlice(typ types.Type) bool {
	switch typ := types.Unalias(typ).Underlying().(type) {
	case *types.Slice:
		return true
	case *types.Basic:
		return typ.Info()&types.IsString != 0
	default:
		return false
	}
}

func basicKind(typ types.Type) types.BasicKind {
	if basic, ok := types.Unalias(typ).Underlying().(*types.Basic); ok {
		return basic.Kind()
	}
	return types.Invalid
}

func (p *context) publishGCRoot(b llssa.Builder, value ssa.Value, expr llssa.Expr) {
	slots, ok := p.gcRoots[value]
	if !ok || expr.IsNil() {
		return
	}
	roots := b.GCRootPointers(expr)
	if len(roots) != len(slots) {
		panic("cl: inconsistent GC root layout")
	}
	for i, root := range roots {
		b.SetGCRoot(slots[i], root)
	}
}

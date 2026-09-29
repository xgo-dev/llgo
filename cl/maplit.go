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
	"go/constant"
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// maxUnrolledMapUpdates matches cmd/compile's maplit: more than 25 static
// entries become a counted loop of mapassign over arrays of keys and values.
const maxUnrolledMapUpdates = 25

type mapLitKind uint8

const (
	mapLitGeneric mapLitKind = iota
	mapLitStaticComplit
)

type mapLitPlan struct {
	kind     mapLitKind
	updates  []*ssa.MapUpdate
	structTy types.Type
	skip     []ssa.Instruction
}

func collectLargeMapLits(fn *ssa.Function) map[*ssa.MapUpdate]*mapLitPlan {
	if fn == nil {
		return nil
	}
	grouped := mapUpdatesInBlockOrder(fn)
	out := make(map[*ssa.MapUpdate]*mapLitPlan)
	for makeMap, updates := range grouped {
		if len(updates) <= maxUnrolledMapUpdates {
			continue
		}
		ok := true
		for _, u := range updates {
			if _, isConst := u.Key.(*ssa.Const); !isConst {
				ok = false
				break
			}
		}
		if !ok || !mapLitSafeToDelay(makeMap, updates) {
			continue
		}
		plan := &mapLitPlan{updates: updates}
		if classifyStaticComplit(updates, plan) {
			plan.kind = mapLitStaticComplit
		} else if allConstMapValues(updates) {
			plan.kind = mapLitGeneric
		} else {
			// Runtime values would be spilled into alloca [N x T] and stored
			// as first-class aggregates. LLVM default<Os> SLP then scalarizes
			// that array, as with ixgo/pkg/unicode's map[string]reflect.Value.
			continue
		}
		for _, u := range updates {
			out[u] = plan
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (p *context) applyMapLitPlans(plans map[*ssa.MapUpdate]*mapLitPlan) {
	if len(plans) == 0 {
		return
	}
	p.ensureStaticInitState()
	seen := make(map[*mapLitPlan]none)
	for _, plan := range plans {
		if _, dup := seen[plan]; dup {
			continue
		}
		seen[plan] = none{}
		if plan.kind == mapLitStaticComplit {
			for _, instr := range plan.skip {
				p.staticInitInstrs[instr] = none{}
			}
		}
		for _, u := range plan.updates[:len(plan.updates)-1] {
			p.staticInitInstrs[u] = none{}
		}
	}
}

func mapLitSafeToDelay(makeMap *ssa.MakeMap, updates []*ssa.MapUpdate) bool {
	last := updates[len(updates)-1]
	lastBlk := last.Block()
	lastIdx := instrIndex(last)
	if lastBlk == nil || lastIdx < 0 {
		return false
	}
	refs, ok := nonDebugReferrers(makeMap)
	if !ok {
		return false
	}
	inPlan := make(map[*ssa.MapUpdate]none, len(updates))
	for _, u := range updates {
		inPlan[u] = none{}
	}
	for _, ref := range refs {
		switch r := ref.(type) {
		case *ssa.MapUpdate:
			if _, ok := inPlan[r]; !ok {
				return false
			}
		case *ssa.Store:
			if r.Val != makeMap {
				return false
			}
		default:
			if !instrIsAfter(ref, lastBlk, lastIdx) {
				return false
			}
		}
	}
	return true
}

func instrIndex(instr ssa.Instruction) int {
	block := instr.Block()
	if block == nil {
		return -1
	}
	for i, in := range block.Instrs {
		if in == instr {
			return i
		}
	}
	return -1
}

func instrIsAfter(instr ssa.Instruction, afterBlock *ssa.BasicBlock, afterIndex int) bool {
	block := instr.Block()
	if block == nil {
		return false
	}
	if block != afterBlock {
		return block.Index > afterBlock.Index
	}
	idx := instrIndex(instr)
	return idx > afterIndex
}

func allConstMapValues(updates []*ssa.MapUpdate) bool {
	for _, u := range updates {
		if _, ok := u.Value.(*ssa.Const); !ok {
			return false
		}
	}
	return true
}

func classifyStaticComplit(updates []*ssa.MapUpdate, plan *mapLitPlan) bool {
	var skip []ssa.Instruction
	for _, u := range updates {
		load, alloc, fields, instrs, ok := mapValueComplit(u.Value)
		if !ok {
			return false
		}
		lrefs, ok := nonDebugReferrers(load)
		if !ok || len(lrefs) != 1 || lrefs[0] != u {
			return false
		}
		if plan.structTy == nil {
			plan.structTy = alloc.Type().(*types.Pointer).Elem()
		}
		st, ok := plan.structTy.Underlying().(*types.Struct)
		if !ok {
			return false
		}
		for i := 0; i < st.NumFields(); i++ {
			fv, ok := fields[i]
			if !ok {
				return false
			}
			if _, isConst := fv.(*ssa.Const); isConst {
				continue
			}
			if _, ok := constantMakeCall(fv); ok {
				continue
			}
			return false
		}
		skip = append(skip, instrs...)
	}
	plan.skip = skip
	return true
}

func mapValueComplit(v ssa.Value) (*ssa.UnOp, *ssa.Alloc, map[int]ssa.Value, []ssa.Instruction, bool) {
	load, ok := v.(*ssa.UnOp)
	if !ok || load.Op != token.MUL {
		return nil, nil, nil, nil, false
	}
	alloc, ok := load.X.(*ssa.Alloc)
	if !ok {
		return nil, nil, nil, nil, false
	}
	refs, ok := nonDebugReferrers(alloc)
	if !ok {
		return nil, nil, nil, nil, false
	}
	fields := make(map[int]ssa.Value)
	instrs := []ssa.Instruction{alloc, load}
	sawLoad := false
	for _, ref := range refs {
		switch r := ref.(type) {
		case *ssa.FieldAddr:
			instrs = append(instrs, r)
			srefs, ok := nonDebugReferrers(r)
			if !ok || len(srefs) != 1 {
				return nil, nil, nil, nil, false
			}
			st, ok := srefs[0].(*ssa.Store)
			if !ok {
				return nil, nil, nil, nil, false
			}
			instrs = append(instrs, st)
			fields[r.Field] = st.Val
			extra, ok := skipStoredValue(st)
			if !ok {
				return nil, nil, nil, nil, false
			}
			instrs = append(instrs, extra...)
		case *ssa.UnOp:
			if r != load {
				return nil, nil, nil, nil, false
			}
			sawLoad = true
		default:
			return nil, nil, nil, nil, false
		}
	}
	if !sawLoad || len(fields) == 0 {
		return nil, nil, nil, nil, false
	}
	return load, alloc, fields, instrs, true
}

func skipStoredValue(st *ssa.Store) ([]ssa.Instruction, bool) {
	var extra []ssa.Instruction
	user := ssa.Instruction(st)
	v := st.Val
	for {
		switch x := v.(type) {
		case *ssa.Call:
			if !singleUser(v, user) {
				return nil, false
			}
			return append(extra, x), true
		case *ssa.ChangeInterface:
			if !singleUser(v, user) {
				return nil, false
			}
			extra = append(extra, x)
			user, v = x, x.X
		case *ssa.ChangeType:
			if !singleUser(v, user) {
				return nil, false
			}
			extra = append(extra, x)
			user, v = x, x.X
		case *ssa.MakeInterface:
			if !singleUser(v, user) {
				return nil, false
			}
			extra = append(extra, x)
			user, v = x, x.X
		default:
			return extra, true
		}
	}
}

func singleUser(v ssa.Value, user ssa.Instruction) bool {
	refs, ok := nonDebugReferrers(v)
	return ok && len(refs) == 1 && refs[0] == user
}

func constantMakeCall(v ssa.Value) (*ssa.Call, bool) {
	for {
		switch x := v.(type) {
		case *ssa.ChangeInterface:
			v = x.X
		case *ssa.ChangeType:
			v = x.X
		case *ssa.MakeInterface:
			v = x.X
		default:
			call, ok := v.(*ssa.Call)
			if !ok || call.Call.IsInvoke() {
				return nil, false
			}
			fn, ok := call.Call.Value.(*ssa.Function)
			if !ok {
				return nil, false
			}
			param, _, ok := analyzeTrivialIfaceBox(fn)
			if !ok {
				return nil, false
			}
			idx := -1
			for i, p := range fn.Params {
				if p == param {
					idx = i
					break
				}
			}
			if idx < 0 || idx >= len(call.Call.Args) {
				return nil, false
			}
			if _, ok := call.Call.Args[idx].(*ssa.Const); !ok {
				return nil, false
			}
			return call, true
		}
	}
}

func (p *context) compileMapLitConst(b llssa.Builder, c *ssa.Const) llssa.Expr {
	t := types.Default(c.Type())
	if basic, ok := t.Underlying().(*types.Basic); ok && basic.Kind() == types.String {
		return b.Pkg.ConstString(constant.StringVal(c.Value))
	}
	bg := llssa.InGo
	if p.inCFunc {
		bg = llssa.InC
	}
	return b.Const(c.Value, p.type_(t, bg))
}

func (p *context) compileMapLitComplit(b llssa.Builder, v ssa.Value, structTy types.Type) (llssa.Expr, bool) {
	_, _, fields, _, ok := mapValueComplit(v)
	if !ok {
		return llssa.Expr{}, false
	}
	st := structTy.Underlying().(*types.Struct)
	flds := make([]llssa.Expr, st.NumFields())
	for i := 0; i < st.NumFields(); i++ {
		fv, ok := fields[i]
		if !ok {
			return llssa.Expr{}, false
		}
		if c, ok := fv.(*ssa.Const); ok {
			flds[i] = p.compileMapLitConst(b, c)
			continue
		}
		call, ok := constantMakeCall(fv)
		if !ok {
			return llssa.Expr{}, false
		}
		folded, ok := p.foldConstantMakeValue(b, call.Call.Value.(*ssa.Function), &call.Call)
		if !ok {
			return llssa.Expr{}, false
		}
		flds[i] = folded
	}
	return b.StructLit(p.type_(structTy, llssa.InGo), flds), true
}

func (p *context) compileMapLitUpdate(b llssa.Builder, v *ssa.MapUpdate) bool {
	if p.mapLitLoops == nil {
		return false
	}
	plan, ok := p.mapLitLoops[v]
	if !ok {
		return false
	}
	if v != plan.updates[len(plan.updates)-1] {
		return true
	}
	p.recordPanicSite(b, v.Pos())
	m := p.compileValue(b, v.Map)
	n := len(plan.updates)
	keys := make([]llssa.Expr, n)
	for i, u := range plan.updates {
		keys[i] = p.compileMapLitConst(b, u.Key.(*ssa.Const))
	}
	vals := make([]llssa.Expr, n)
	if plan.kind == mapLitStaticComplit {
		for i, u := range plan.updates {
			val, ok := p.compileMapLitComplit(b, u.Value, plan.structTy)
			if !ok {
				return false
			}
			vals[i] = val
		}
		b.MapLitLoop(m, keys, vals)
		return true
	}
	for i, u := range plan.updates {
		if expr, ok := p.staticMapSliceValues[u]; ok {
			vals[i] = expr
			continue
		}
		if c, ok := u.Value.(*ssa.Const); ok {
			vals[i] = p.compileMapLitConst(b, c)
			continue
		}
		vals[i] = p.compileValue(b, u.Value)
	}
	b.MapLitLoop(m, keys, vals)
	return true
}

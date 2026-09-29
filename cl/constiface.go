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
	llssa "github.com/xgo-dev/llgo/ssa"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// foldConstantMakeValue inlines a trivial "box a parameter as an interface"
// function when the argument is a compile-time constant. The callee is
// recognized by its SSA body (ChangeType/ChangeInterface of a param, then
// MakeInterface, then return), not by name. That matches cmd/compile
// inlining constructors such as constant.MakeInt64. Convert is excluded:
// it changes the value (int32→string, narrowing) and cannot reuse the
// argument's raw constant.
func (p *context) foldConstantMakeValue(b llssa.Builder, fn *ssa.Function, call *ssa.CallCommon) (llssa.Expr, bool) {
	param, concrete, ok := analyzeTrivialIfaceBox(fn)
	if !ok {
		return llssa.Expr{}, false
	}
	idx := -1
	for i, p := range fn.Params {
		if p == param {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(call.Args) {
		return llssa.Expr{}, false
	}
	c, ok := call.Args[idx].(*ssa.Const)
	if !ok || c.Value == nil {
		return llssa.Expr{}, false
	}
	x := b.Const(c.Value, p.type_(concrete, llssa.InGo))
	retTy := p.type_(call.Signature().Results().At(0).Type(), llssa.InGo)
	return b.MakeInterface(retTy, x), true
}

// analyzeTrivialIfaceBox reports that fn is a single-block function whose
// result is MakeInterface of a parameter (optionally after ChangeType
// /ChangeInterface). The concrete type is the value being boxed.
func analyzeTrivialIfaceBox(fn *ssa.Function) (*ssa.Parameter, types.Type, bool) {
	if fn == nil || fn.Signature.Recv() != nil || fn.Recover != nil || fn.Synthetic != "" {
		return nil, nil, false
	}
	if hasNoInlineDirective(fn) {
		return nil, nil, false
	}
	if len(fn.Blocks) != 1 {
		return nil, nil, false
	}
	blk := fn.Blocks[0]
	var ret *ssa.Return
	for _, instr := range blk.Instrs {
		r, ok := instr.(*ssa.Return)
		if !ok {
			continue
		}
		if ret != nil {
			return nil, nil, false
		}
		ret = r
	}
	if ret == nil || len(ret.Results) != 1 {
		return nil, nil, false
	}
	mi, ok := ret.Results[0].(*ssa.MakeInterface)
	if !ok {
		return nil, nil, false
	}
	chain := map[ssa.Instruction]bool{ret: true, mi: true}
	v := mi.X
	for {
		switch t := v.(type) {
		case *ssa.Parameter:
			if blockHasExtraInstrs(blk, chain) {
				return nil, nil, false
			}
			return t, mi.X.Type(), true
		case *ssa.ChangeType:
			chain[t] = true
			v = t.X
		case *ssa.ChangeInterface:
			chain[t] = true
			v = t.X
		default:
			return nil, nil, false
		}
	}
}

func blockHasExtraInstrs(blk *ssa.BasicBlock, chain map[ssa.Instruction]bool) bool {
	for _, instr := range blk.Instrs {
		if _, ok := instr.(*ssa.DebugRef); ok {
			continue
		}
		if !chain[instr] {
			return true
		}
	}
	return false
}

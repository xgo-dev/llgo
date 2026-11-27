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

package ssaopt

import (
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// NilCheckPass is the single owner of pointer-method and value-method receiver
// nil-check lowering decisions. It consumes first-class RecvNilCheck source
// facts recorded by llgo/ssa and emits read-only NilCheckPlan entries verified
// by VerifyPlan.
//
// At -O0, every semantically required check is preserved (skipping only
// syntactically non-nil addresses such as &local, &global, or ssa:wrapnilchk)
// to maintain exact panic evaluation order. At >= -O1, dominating non-nil facts
// (prior checked dereferences, FieldAddr bases, and dominating p != nil
// branches) can prove checks redundant with an explicit proof anchor.
type NilCheckPass struct{}

func (NilCheckPass) Name() string { return "nilcheck" }

func (NilCheckPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}

	// Pass 1: collect value-receiver dereference loads and pointer-method
	// receiver pointer loads that must be guarded before the load itself.
	var methodDeref map[*ssa.UnOp]struct{}
	var recvDeref map[*ssa.UnOp]token.Pos
	markMethodDeref := func(arg *ssa.UnOp, ok bool) {
		if !ok {
			return
		}
		if methodDeref == nil {
			methodDeref = make(map[*ssa.UnOp]struct{})
		}
		methodDeref[arg] = struct{}{}
	}
	var markReceiverLoads func(ssa.Value, token.Pos)
	markReceiverLoads = func(value ssa.Value, pos token.Pos) {
		switch value := value.(type) {
		case *ssa.UnOp:
			if value.Op != token.MUL {
				return
			}
			if !IsKnownNonNilAddr(value.X) && !IsWrapNilCheckCall(value.X) {
				if recvDeref == nil {
					recvDeref = make(map[*ssa.UnOp]token.Pos)
				}
				recvDeref[value] = pos
			}
			markReceiverLoads(value.X, pos)
		case *ssa.FieldAddr:
			markReceiverLoads(value.X, pos)
		}
	}

	// Collect per-block non-nil proofs along the dominator tree (for >= -O1).
	domProofs := collectDomNonNilProofs(fn, cfg.Optimizing())

	var nilChecks map[ssa.Instruction]NilCheckPlan
	recordNilCheck := func(ins ssa.Instruction, recv ssa.Value, pos token.Pos, address bool) {
		if nilChecks == nil {
			nilChecks = make(map[ssa.Instruction]NilCheckPlan)
		}
		// Syntactic non-nil guarantees apply at all opt levels (including -O0).
		if kind, ok := syntacticNonNil(recv); ok {
			nilChecks[ins] = NilCheckPlan{
				Elided:    true,
				Pos:       pos,
				ProofKind: kind,
			}
			return
		}
		if load, ok := recv.(*ssa.UnOp); ok && !address {
			if _, protected := recvDeref[load]; protected {
				nilChecks[ins] = NilCheckPlan{
					Elided:    true,
					Pos:       pos,
					ProofKind: NilProofAlreadyProtectedLoad,
				}
				return
			}
		}
		hasUncheckedBase := MethodReceiverHasUncheckedBase(recv)
		if !address && !hasUncheckedBase {
			// Plain p.M() on a pointer receiver without deref/FieldAddr does not
			// require a nil check in Go.
			return
		}
		// At >= -O1, check if a dominating non-nil proof covers recv (and any
		// FieldAddr/UnOp base).
		if cfg.Optimizing() {
			if kind, anchor, ok := domProofs.lookup(recv, ins); ok && (!hasUncheckedBase || baseCovered(recv, ins, domProofs)) {
				nilChecks[ins] = NilCheckPlan{
					Elided:      true,
					Pos:         pos,
					ProofKind:   kind,
					ProofAnchor: anchor,
				}
				return
			}
		}
		nilChecks[ins] = NilCheckPlan{
			NeedBaseCheck: hasUncheckedBase,
			NeedAddrCheck: address,
			Pos:           pos,
		}
	}

	// First scan: collect load-site guards so Call/MakeClosure sites know which
	// *ssa.UnOp loads are already protected at the load.
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.Call:
				scanCallReceiverLoads(fn, &ins.Call, markMethodDeref, markReceiverLoads)
			case *ssa.Defer:
				scanCallReceiverLoads(fn, &ins.Call, markMethodDeref, markReceiverLoads)
			case *ssa.Go:
				scanCallReceiverLoads(fn, &ins.Call, markMethodDeref, markReceiverLoads)
			case *ssa.MakeClosure:
				if bound, ok := ins.Fn.(*ssa.Function); ok {
					markMethodDeref(boundValueReceiverNilDerefArg(bound, ins.Bindings))
				}
				if len(ins.Bindings) > 0 && ins.RecvNilCheck != nil {
					markReceiverLoads(ins.Bindings[0], ins.RecvNilCheck.Pos)
				}
			}
		}
	}

	// Second scan: emit NilCheckPlan for every Call/Defer/Go/MakeClosure site.
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.Call:
				planCallNilCheck(fn, ins, &ins.Call, recordNilCheck)
			case *ssa.Defer:
				planCallNilCheck(fn, ins, &ins.Call, recordNilCheck)
			case *ssa.Go:
				planCallNilCheck(fn, ins, &ins.Call, recordNilCheck)
			case *ssa.MakeClosure:
				if len(ins.Bindings) > 0 && ins.RecvNilCheck != nil {
					recordNilCheck(ins, ins.Bindings[0], ins.RecvNilCheck.Pos, ins.RecvNilCheck.Address)
				}
			}
		}
	}

	plan.NilChecks = nilChecks
	plan.MethodNilDerefChecks = methodDeref
	plan.RecvNilDerefChecks = recvDeref
	return nil
}

func scanCallReceiverLoads(
	owner *ssa.Function,
	call *ssa.CallCommon,
	markMethodDeref func(*ssa.UnOp, bool),
	markReceiverLoads func(ssa.Value, token.Pos),
) {
	if callee, ok := call.Value.(*ssa.Function); ok {
		markMethodDeref(valueReceiverNilDerefArg(callee, call.Args))
	}
	if len(call.Args) == 0 {
		return
	}
	if IsPointerMethodWrapperCall(owner, call) {
		markReceiverLoads(call.Args[0], owner.Pos())
	} else if call.RecvNilCheck != nil {
		markReceiverLoads(call.Args[0], call.RecvNilCheck.Pos)
	}
}

func planCallNilCheck(
	owner *ssa.Function,
	ins ssa.Instruction,
	call *ssa.CallCommon,
	record func(ssa.Instruction, ssa.Value, token.Pos, bool),
) {
	if len(call.Args) == 0 {
		return
	}
	if IsPointerMethodWrapperCall(owner, call) {
		record(ins, call.Args[0], owner.Pos(), false)
		return
	}
	if call.RecvNilCheck != nil {
		record(ins, call.Args[0], call.RecvNilCheck.Pos, call.RecvNilCheck.Address)
	}
}

// IsKnownNonNilAddr reports whether v is syntactically guaranteed to be a
// non-nil address (stack/heap Alloc, package Global, or chain of FieldAddr/
// IndexAddr rooted at one).
func IsKnownNonNilAddr(v ssa.Value) bool {
	for {
		switch x := v.(type) {
		case *ssa.Alloc, *ssa.Global:
			return true
		case *ssa.FieldAddr:
			v = x.X
		case *ssa.IndexAddr:
			if isPointerGoType(x.X.Type()) {
				v = x.X
			} else {
				return false
			}
		default:
			return false
		}
	}
}

// IsWrapNilCheckCall reports whether v is the result of ssa:wrapnilchk.
func IsWrapNilCheckCall(v ssa.Value) bool {
	call, ok := v.(*ssa.Call)
	if !ok {
		return false
	}
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	return ok && builtin.Name() == "ssa:wrapnilchk"
}

func syntacticNonNil(v ssa.Value) (NilProofKind, bool) {
	if IsWrapNilCheckCall(v) {
		return NilProofWrapNilChk, true
	}
	cur := v
	for {
		switch x := cur.(type) {
		case *ssa.Alloc:
			return NilProofAlloc, true
		case *ssa.Global:
			return NilProofGlobal, true
		case *ssa.FieldAddr:
			cur = x.X
		case *ssa.IndexAddr:
			if isPointerGoType(x.X.Type()) {
				cur = x.X
			} else {
				return NilProofNone, false
			}
		default:
			return NilProofNone, false
		}
	}
}

// IsPointerMethodWrapperCall reports whether call inside owner is a forwarded
// call to a pointer-receiver method inside a synthetic wrapper or thunk.
func IsPointerMethodWrapperCall(owner *ssa.Function, call *ssa.CallCommon) bool {
	if !IsMethodReceiverWrapper(owner) {
		return false
	}
	fn := call.StaticCallee()
	return fn != nil && fn.Signature.Recv() != nil && isPointerGoType(fn.Signature.Recv().Type())
}

// IsMethodReceiverWrapper reports whether fn is a synthetic method wrapper or thunk.
func IsMethodReceiverWrapper(fn *ssa.Function) bool {
	return fn != nil && (strings.HasPrefix(fn.Synthetic, "wrapper for ") || strings.HasPrefix(fn.Synthetic, "thunk for "))
}

// MethodReceiverHasUncheckedBase reports whether receiver is a pointer load or
// FieldAddr whose base pointer is not syntactically known non-nil.
func MethodReceiverHasUncheckedBase(receiver ssa.Value) bool {
	switch receiver := receiver.(type) {
	case *ssa.UnOp:
		return receiver.Op == token.MUL && !IsKnownNonNilAddr(receiver.X) && !IsWrapNilCheckCall(receiver.X)
	case *ssa.FieldAddr:
		return !IsKnownNonNilAddr(receiver.X) && !IsWrapNilCheckCall(receiver.X)
	}
	return false
}

func valueReceiverNilDerefArg(fn *ssa.Function, args []ssa.Value) (*ssa.UnOp, bool) {
	if fn == nil || len(args) == 0 {
		return nil, false
	}
	recv := fn.Signature.Recv()
	if recv == nil || isPointerGoType(recv.Type()) {
		return nil, false
	}
	arg, ok := args[0].(*ssa.UnOp)
	if !ok || arg.Op != token.MUL || IsKnownNonNilAddr(arg.X) || IsWrapNilCheckCall(arg.X) {
		return nil, false
	}
	return arg, true
}

func boundValueReceiverNilDerefArg(fn *ssa.Function, bindings []ssa.Value) (*ssa.UnOp, bool) {
	if fn == nil || len(fn.FreeVars) == 0 || len(bindings) == 0 {
		return nil, false
	}
	if !strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") {
		return nil, false
	}
	if isPointerGoType(fn.FreeVars[0].Type()) {
		return nil, false
	}
	arg, ok := bindings[0].(*ssa.UnOp)
	if !ok || arg.Op != token.MUL || IsKnownNonNilAddr(arg.X) || IsWrapNilCheckCall(arg.X) {
		return nil, false
	}
	return arg, true
}

func isPointerGoType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	_, ok := types.Unalias(typ).Underlying().(*types.Pointer)
	return ok
}

type nonNilProof struct {
	kind   NilProofKind
	anchor ssa.Instruction
	blk    *ssa.BasicBlock
	idx    int
}

type domNonNilTable struct {
	enabled  bool
	hasDom   bool
	instrPos map[ssa.Instruction]struct {
		blk *ssa.BasicBlock
		idx int
	}
	proofs    map[ssa.Value][]nonNilProof
	edgeProof map[*ssa.BasicBlock]map[ssa.Value]nonNilProof
}

func collectDomNonNilProofs(fn *ssa.Function, optimizing bool) *domNonNilTable {
	t := &domNonNilTable{enabled: optimizing}
	if !optimizing || fn == nil || len(fn.Blocks) == 0 {
		return t
	}
	t.hasDom = fn.Blocks[0].Dominates(fn.Blocks[0])
	if !t.hasDom {
		return t
	}
	t.instrPos = make(map[ssa.Instruction]struct {
		blk *ssa.BasicBlock
		idx int
	}, 64)
	t.proofs = make(map[ssa.Value][]nonNilProof)
	t.edgeProof = make(map[*ssa.BasicBlock]map[ssa.Value]nonNilProof)

	addProof := func(v ssa.Value, p nonNilProof) {
		if v == nil || !isPointerGoType(v.Type()) {
			return
		}
		t.proofs[v] = append(t.proofs[v], p)
	}

	for _, blk := range fn.Blocks {
		for idx, ins := range blk.Instrs {
			t.instrPos[ins] = struct {
				blk *ssa.BasicBlock
				idx int
			}{blk: blk, idx: idx}
			switch ins := ins.(type) {
			case *ssa.UnOp:
				// A dereference *p proves p != nil for all dominated instructions
				// (Go panics on nil dereference before continuing).
				if ins.Op == token.MUL && isPointerGoType(ins.X.Type()) {
					addProof(ins.X, nonNilProof{kind: NilProofDomCheck, anchor: ins, blk: blk, idx: idx})
				}
			case *ssa.Store:
				if isPointerGoType(ins.Addr.Type()) {
					addProof(ins.Addr, nonNilProof{kind: NilProofDomCheck, anchor: ins, blk: blk, idx: idx})
				}
			case *ssa.Call:
				// A prior receiver nil-checked call proves its receiver non-nil.
				if ins.Call.RecvNilCheck != nil && ins.Call.RecvNilCheck.Address && len(ins.Call.Args) > 0 {
					addProof(ins.Call.Args[0], nonNilProof{kind: NilProofDomCheck, anchor: ins, blk: blk, idx: idx})
				}
			case *ssa.If:
				// Check for `if p != nil` / `if p == nil` dominating a single-predecessor successor.
				bin, ok := ins.Cond.(*ssa.BinOp)
				if !ok || (bin.Op != token.NEQ && bin.Op != token.EQL) {
					continue
				}
				var ptr ssa.Value
				if c, ok := bin.Y.(*ssa.Const); ok && c.IsNil() && isPointerGoType(bin.X.Type()) {
					ptr = bin.X
				} else if c, ok := bin.X.(*ssa.Const); ok && c.IsNil() && isPointerGoType(bin.Y.Type()) {
					ptr = bin.Y
				}
				if ptr == nil || len(blk.Succs) != 2 {
					continue
				}
				nonNilSucc := blk.Succs[0]
				if bin.Op == token.EQL {
					nonNilSucc = blk.Succs[1]
				}
				if len(nonNilSucc.Preds) == 1 {
					if t.edgeProof[nonNilSucc] == nil {
						t.edgeProof[nonNilSucc] = make(map[ssa.Value]nonNilProof)
					}
					t.edgeProof[nonNilSucc][ptr] = nonNilProof{
						kind:   NilProofDomCond,
						anchor: ins,
						blk:    blk,
						idx:    idx,
					}
				}
			}
		}
	}
	return t
}

func (t *domNonNilTable) lookup(v ssa.Value, target ssa.Instruction) (NilProofKind, ssa.Instruction, bool) {
	if !t.enabled || !t.hasDom || v == nil {
		return NilProofNone, nil, false
	}
	tp, ok := t.instrPos[target]
	if !ok {
		return NilProofNone, nil, false
	}
	for _, p := range t.proofs[v] {
		if p.blk == tp.blk {
			if p.idx < tp.idx {
				return p.kind, p.anchor, true
			}
		} else if p.blk.Dominates(tp.blk) {
			return p.kind, p.anchor, true
		}
	}
	for domBlk := tp.blk; domBlk != nil; domBlk = domBlk.Idom() {
		if ep, ok := t.edgeProof[domBlk][v]; ok {
			return ep.kind, ep.anchor, true
		}
	}
	return NilProofNone, nil, false
}

func baseCovered(recv ssa.Value, target ssa.Instruction, domProofs *domNonNilTable) bool {
	switch r := recv.(type) {
	case *ssa.FieldAddr:
		if IsKnownNonNilAddr(r.X) || IsWrapNilCheckCall(r.X) {
			return true
		}
		_, _, ok := domProofs.lookup(r.X, target)
		return ok && baseCovered(r.X, target, domProofs)
	case *ssa.UnOp:
		if r.Op != token.MUL {
			return true
		}
		if IsKnownNonNilAddr(r.X) || IsWrapNilCheckCall(r.X) {
			return true
		}
		_, _, ok := domProofs.lookup(r.X, target)
		return ok
	default:
		return true
	}
}

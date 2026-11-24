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

package ir

import (
	"fmt"

	lltypes "github.com/xgo-dev/llgo/types"
)

// VerifyPackage checks structural, CFG, calling-convention, and GC-safety
// invariants across every function in pkg.
func VerifyPackage(pkg *Package) error {
	if pkg == nil {
		return fmt.Errorf("ir: nil package")
	}
	seenFuncs := make(map[string]bool, len(pkg.Funcs))
	for _, fn := range pkg.Funcs {
		if fn.Name == "" {
			return fmt.Errorf("ir: unnamed function in package %s", pkg.Path)
		}
		if seenFuncs[fn.Name] {
			return fmt.Errorf("ir: duplicate function %q in package %s", fn.Name, pkg.Path)
		}
		seenFuncs[fn.Name] = true
		if err := VerifyFunction(fn); err != nil {
			return err
		}
	}
	return nil
}

// VerifyFunction verifies a single function's CFG, block terminators, Phi
// predecessors, SSA value ownership, and LLGo-specific instruction invariants.
func VerifyFunction(fn *Function) error {
	if fn == nil {
		return fmt.Errorf("ir: nil function")
	}
	if fn.Sig == nil {
		return fmt.Errorf("ir: function %s missing signature", fn.Name)
	}
	if fn.IsDeclaration() {
		return nil
	}
	blkSet := make(map[*BasicBlock]bool, len(fn.Blocks))
	for _, blk := range fn.Blocks {
		if blk == nil {
			return fmt.Errorf("ir: %s: nil basic block", fn.Name)
		}
		if blkSet[blk] {
			return fmt.Errorf("ir: %s: duplicate basic block %s", fn.Name, blk.Label)
		}
		blkSet[blk] = true
	}
	paramSet := make(map[*Parameter]bool, len(fn.Params))
	for _, p := range fn.Params {
		if p == nil {
			return fmt.Errorf("ir: %s: nil parameter", fn.Name)
		}
		if p.Type() == nil {
			return fmt.Errorf("ir: %s: parameter %s has nil type", fn.Name, p.Name())
		}
		paramSet[p] = true
	}

	// Pass 1: validate block structure, build CFG predecessor map, and index
	// value-producing instructions owned by fn.
	preds := make(map[*BasicBlock]map[*BasicBlock]bool, len(fn.Blocks))
	instrOwner := make(map[Value]*BasicBlock)
	for _, blk := range fn.Blocks {
		if len(blk.Instrs) == 0 {
			return fmt.Errorf("ir: %s.%s: empty basic block", fn.Name, blk.Label)
		}
		seenNonPhi := false
		for idx, ins := range blk.Instrs {
			if ins == nil {
				return fmt.Errorf("ir: %s.%s: nil instruction at index %d", fn.Name, blk.Label, idx)
			}
			if _, isPhi := ins.(*Phi); isPhi {
				if seenNonPhi {
					return fmt.Errorf("ir: %s.%s: phi instruction after non-phi instruction", fn.Name, blk.Label)
				}
			} else {
				seenNonPhi = true
			}
			if v, ok := ins.(Value); ok {
				instrOwner[v] = blk
			}
			isLast := idx == len(blk.Instrs)-1
			term, isTerm := ins.(Terminator)
			if isTerm && !isLast {
				return fmt.Errorf("ir: %s.%s: terminator %T before end of block", fn.Name, blk.Label, ins)
			}
			if !isTerm && isLast {
				return fmt.Errorf("ir: %s.%s: missing terminator at end of block", fn.Name, blk.Label)
			}
			if isTerm {
				for _, succ := range term.Successors() {
					if succ == nil || !blkSet[succ] {
						return fmt.Errorf("ir: %s.%s: invalid successor block in %T", fn.Name, blk.Label, ins)
					}
					if preds[succ] == nil {
						preds[succ] = make(map[*BasicBlock]bool)
					}
					preds[succ][blk] = true
				}
			}
		}
	}

	// Pass 2: validate operand ownership/typing, Phi predecessors, and
	// instruction-specific invariants.
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			for _, op := range ins.Operands() {
				if op == nil {
					return fmt.Errorf("ir: %s.%s: nil operand in %T", fn.Name, blk.Label, ins)
				}
				if op.Type() == nil {
					return fmt.Errorf("ir: %s.%s: untyped operand %q in %T", fn.Name, blk.Label, op.Name(), ins)
				}
				if err := verifyValueOwner(fn, blk, op, paramSet, instrOwner); err != nil {
					return err
				}
			}
			if err := verifyInstr(fn, blk, ins, blkSet, preds[blk]); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyValueOwner(fn *Function, blk *BasicBlock, op Value, paramSet map[*Parameter]bool, instrOwner map[Value]*BasicBlock) error {
	switch v := op.(type) {
	case *Parameter:
		if !paramSet[v] {
			return fmt.Errorf("ir: %s.%s: operand param %s does not belong to function %s", fn.Name, blk.Label, v.Name(), fn.Name)
		}
	case *Const:
		// Compile-time constants are value literals with no owning function.
	case *GlobalRef:
		if v.Global == nil {
			return fmt.Errorf("ir: %s.%s: global reference with nil Global", fn.Name, blk.Label)
		}
	default:
		if _, ok := instrOwner[op]; !ok {
			return fmt.Errorf("ir: %s.%s: operand %s does not belong to function %s", fn.Name, blk.Label, op.Name(), fn.Name)
		}
	}
	return nil
}

func verifyInstr(fn *Function, blk *BasicBlock, ins Instruction, blkSet map[*BasicBlock]bool, blkPreds map[*BasicBlock]bool) error {
	switch ins := ins.(type) {
	case *Phi:
		if len(ins.Edges) == 0 {
			return fmt.Errorf("ir: %s.%s: phi has no incoming edges", fn.Name, blk.Label)
		}
		if len(ins.Edges) != len(blkPreds) {
			return fmt.Errorf("ir: %s.%s: phi has %d incoming edges, want %d CFG predecessors", fn.Name, blk.Label, len(ins.Edges), len(blkPreds))
		}
		seenPred := make(map[*BasicBlock]bool, len(ins.Edges))
		for _, edge := range ins.Edges {
			if edge.Pred == nil || !blkSet[edge.Pred] {
				return fmt.Errorf("ir: %s.%s: phi incoming block does not belong to function %s", fn.Name, blk.Label, fn.Name)
			}
			if !blkPreds[edge.Pred] {
				return fmt.Errorf("ir: %s.%s: phi incoming block %s is not a CFG predecessor", fn.Name, blk.Label, edge.Pred.Label)
			}
			if seenPred[edge.Pred] {
				return fmt.Errorf("ir: %s.%s: duplicate phi predecessor %s", fn.Name, blk.Label, edge.Pred.Label)
			}
			seenPred[edge.Pred] = true
		}
	case *Alloc:
		if ins.Elem == nil {
			return fmt.Errorf("ir: %s.%s: alloc with nil element type", fn.Name, blk.Label)
		}
		if ins.NoScan && ins.Elem.HasPtrData() {
			return fmt.Errorf("ir: %s.%s: alloc.noscan on GC-pointer-bearing type %s", fn.Name, blk.Label, ins.Elem)
		}
	case *WriteBarrier:
		dstType := ins.Dst.Type()
		if dstType == nil {
			return fmt.Errorf("ir: %s.%s: write barrier destination has nil type", fn.Name, blk.Label)
		}
		ptr, ok := dstType.Underlying().(*lltypes.Pointer)
		if !ok || ptr.Kind != lltypes.GoPtr {
			return fmt.Errorf("ir: %s.%s: write barrier destination %s must be a GoPtr", fn.Name, blk.Label, dstType)
		}
	case *CallClosure:
		cloType := ins.Closure.Type()
		if cloType == nil {
			return fmt.Errorf("ir: %s.%s: call.closure callee has nil type", fn.Name, blk.Label)
		}
		if _, ok := cloType.Underlying().(*lltypes.Closure); !ok {
			return fmt.Errorf("ir: %s.%s: call.closure callee %s is not a Closure", fn.Name, blk.Label, cloType)
		}
	case *CallCFunc:
		if !ins.CallConv.IsNative() {
			return fmt.Errorf("ir: %s.%s: call.cfunc requires CDecl or Stdcall, got %s", fn.Name, blk.Label, ins.CallConv)
		}
	case *Return:
		if len(ins.Results) != len(fn.Sig.Results) {
			return fmt.Errorf("ir: %s.%s: return arity %d != signature results %d", fn.Name, blk.Label, len(ins.Results), len(fn.Sig.Results))
		}
	}
	return nil
}

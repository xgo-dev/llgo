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
	"sort"

	"github.com/xgo-dev/llgo/types"
)

// VerifyPackage checks structural, CFG, SSA value ownership, and GC-safety
// invariants across every Function reachable from pkg.Members.
func VerifyPackage(pkg *Package) error {
	if pkg == nil {
		return fmt.Errorf("ssa: nil package")
	}
	names := make([]string, 0, len(pkg.Members))
	for name := range pkg.Members {
		names = append(names, name)
	}
	sort.Strings(names)
	var verifyTree func(fn *Function) error
	verifyTree = func(fn *Function) error {
		if err := VerifyFunction(fn); err != nil {
			return err
		}
		for _, anon := range fn.AnonFuncs {
			if err := verifyTree(anon); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range names {
		if fn, ok := pkg.Members[name].(*Function); ok {
			if err := verifyTree(fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// VerifyFunction verifies a single function's CFG, block terminators, Phi
// predecessors, SSA value ownership, and instruction invariants.
func VerifyFunction(fn *Function) error {
	if fn == nil {
		return fmt.Errorf("ssa: nil function")
	}
	if fn.Signature == nil {
		return fmt.Errorf("ssa: function %s missing signature", fn.String())
	}
	if len(fn.Blocks) == 0 {
		return nil
	}

	blkSet := make(map[*BasicBlock]bool, len(fn.Blocks))
	for i, blk := range fn.Blocks {
		if blk == nil {
			return fmt.Errorf("ssa: %s: nil basic block at %d", fn.String(), i)
		}
		if blkSet[blk] {
			return fmt.Errorf("ssa: %s: duplicate basic block %s", fn.String(), blk.String())
		}
		blkSet[blk] = true
	}

	localValues := make(map[Value]bool, len(fn.Params)+len(fn.FreeVars)+len(fn.AnonFuncs)+32)
	for _, p := range fn.Params {
		if p == nil || p.Type() == nil {
			return fmt.Errorf("ssa: %s: invalid parameter", fn.String())
		}
		localValues[p] = true
	}
	for _, fv := range fn.FreeVars {
		if fv == nil || fv.Type() == nil {
			return fmt.Errorf("ssa: %s: invalid freevar", fn.String())
		}
		localValues[fv] = true
	}
	for _, anon := range fn.AnonFuncs {
		localValues[anon] = true
	}

	// Pass 1: validate block terminators, Succs/Preds symmetry, Phi placement,
	// and collect all instruction-defined values.
	for _, blk := range fn.Blocks {
		if len(blk.Instrs) == 0 {
			return fmt.Errorf("ssa: %s.%s: empty basic block", fn.String(), blk.String())
		}
		seenNonPhi := false
		for idx, ins := range blk.Instrs {
			if ins == nil {
				return fmt.Errorf("ssa: %s.%s: nil instruction at %d", fn.String(), blk.String(), idx)
			}
			if _, isPhi := ins.(*Phi); isPhi {
				if seenNonPhi {
					return fmt.Errorf("ssa: %s.%s: phi instruction after non-phi instruction", fn.String(), blk.String())
				}
			} else {
				seenNonPhi = true
			}
			if v, ok := ins.(Value); ok {
				localValues[v] = true
			}
			isLast := idx == len(blk.Instrs)-1
			term := isTerminator(ins)
			if term && !isLast {
				return fmt.Errorf("ssa: %s.%s: terminator %T before end of block", fn.String(), blk.String(), ins)
			}
			if !term && isLast {
				return fmt.Errorf("ssa: %s.%s: missing terminator at end of block", fn.String(), blk.String())
			}
		}
		for _, succ := range blk.Succs {
			if !blkSet[succ] {
				return fmt.Errorf("ssa: %s.%s: successor does not belong to function", fn.String(), blk.String())
			}
		}
		for _, pred := range blk.Preds {
			if !blkSet[pred] {
				return fmt.Errorf("ssa: %s.%s: predecessor does not belong to function", fn.String(), blk.String())
			}
		}
		switch blk.Instrs[len(blk.Instrs)-1].(type) {
		case *Jump:
			if len(blk.Succs) != 1 {
				return fmt.Errorf("ssa: %s.%s: jump block has %d successors, want 1", fn.String(), blk.String(), len(blk.Succs))
			}
		case *If:
			if len(blk.Succs) != 2 {
				return fmt.Errorf("ssa: %s.%s: if block has %d successors, want 2", fn.String(), blk.String(), len(blk.Succs))
			}
		case *Return, *Panic:
			if len(blk.Succs) != 0 {
				return fmt.Errorf("ssa: %s.%s: return/panic block has %d successors, want 0", fn.String(), blk.String(), len(blk.Succs))
			}
		}
	}

	// Pass 2: validate operands and instruction-specific invariants.
	var rands []*Value
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			rands = ins.Operands(rands[:0])
			for _, opPtr := range rands {
				if opPtr == nil || *opPtr == nil {
					return fmt.Errorf("ssa: %s.%s: nil operand in %T", fn.String(), blk.String(), ins)
				}
				op := *opPtr
				if op.Type() == nil {
					return fmt.Errorf("ssa: %s.%s: untyped operand %q in %T", fn.String(), blk.String(), op.Name(), ins)
				}
				switch op.(type) {
				case *Const, *Global, *Function, *Builtin:
					// Package/program-level values are valid operands across functions.
				default:
					if !localValues[op] {
						return fmt.Errorf("ssa: %s.%s: operand %s does not belong to function %s", fn.String(), blk.String(), op.Name(), fn.String())
					}
				}
			}
			if err := verifyInstr(fn, blk, ins); err != nil {
				return err
			}
		}
	}
	return nil
}

func isTerminator(ins Instruction) bool {
	switch ins.(type) {
	case *Jump, *If, *Return, *Panic:
		return true
	default:
		return false
	}
}

func verifyInstr(fn *Function, blk *BasicBlock, ins Instruction) error {
	switch ins := ins.(type) {
	case *Phi:
		if len(ins.Edges) != len(blk.Preds) {
			return fmt.Errorf("ssa: %s.%s: phi has %d edges, want %d predecessors", fn.String(), blk.String(), len(ins.Edges), len(blk.Preds))
		}
	case *Alloc:
		ptr, ok := ins.Type().(*types.Pointer)
		if !ok || ptr.Elem() == nil {
			return fmt.Errorf("ssa: %s.%s: alloc must have pointer type, got %v", fn.String(), blk.String(), ins.Type())
		}
		if ins.NoScan && ptr.Elem().HasPtrData() {
			return fmt.Errorf("ssa: %s.%s: alloc.noscan on pointer-bearing type %s", fn.String(), blk.String(), ptr.Elem())
		}
	case *Extract:
		tup, ok := ins.Tuple.Type().(*types.Tuple)
		if !ok {
			return fmt.Errorf("ssa: %s.%s: extract operand %s has non-tuple type %s", fn.String(), blk.String(), ins.Tuple.Name(), ins.Tuple.Type())
		}
		if ins.Index < 0 || ins.Index >= tup.Len() {
			return fmt.Errorf("ssa: %s.%s: extract index %d out of range [0,%d)", fn.String(), blk.String(), ins.Index, tup.Len())
		}
	case *Return:
		if len(ins.Results) != fn.Signature.Results().Len() {
			return fmt.Errorf("ssa: %s.%s: return arity %d != signature results %d", fn.String(), blk.String(), len(ins.Results), fn.Signature.Results().Len())
		}
	}
	return nil
}

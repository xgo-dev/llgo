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
		if err := VerifyFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

// VerifyFunc verifies a single function's CFG, block terminators, and
// LLGo-specific instruction invariants.
func VerifyFunc(fn *Func) error {
	if fn == nil {
		return fmt.Errorf("ir: nil function")
	}
	if fn.Sig == nil {
		return fmt.Errorf("ir: function %s missing signature", fn.Name)
	}
	if fn.IsDeclaration() {
		return nil
	}
	blkSet := make(map[*Block]bool, len(fn.Blocks))
	for _, blk := range fn.Blocks {
		blkSet[blk] = true
	}
	for _, blk := range fn.Blocks {
		if len(blk.Instrs) == 0 {
			return fmt.Errorf("ir: %s.%s: empty basic block", fn.Name, blk.Label)
		}
		for idx, ins := range blk.Instrs {
			isLast := idx == len(blk.Instrs)-1
			term, isTerm := ins.(Terminator)
			if isTerm && !isLast {
				return fmt.Errorf("ir: %s.%s: terminator %q before end of block", fn.Name, blk.Label, ins)
			}
			if !isTerm && isLast {
				return fmt.Errorf("ir: %s.%s: missing terminator at end of block", fn.Name, blk.Label)
			}
			if isTerm {
				for _, succ := range term.Successors() {
					if succ == nil || !blkSet[succ] {
						return fmt.Errorf("ir: %s.%s: invalid successor block in %q", fn.Name, blk.Label, ins)
					}
				}
			}
			for _, op := range ins.Operands() {
				if op == nil {
					return fmt.Errorf("ir: %s.%s: nil operand in %T", fn.Name, blk.Label, ins)
				}
				if op.Type() == nil {
					return fmt.Errorf("ir: %s.%s: untyped operand %q in %T", fn.Name, blk.Label, op.Name(), ins)
				}
			}
			if err := verifyInstr(fn, blk, ins); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyInstr(fn *Func, blk *Block, ins Instr) error {
	switch ins := ins.(type) {
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

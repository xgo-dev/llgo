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

package ssa_test

import (
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

// newStackBudgetProgram starts one in-process program whose goroutine stack
// size is stackSize (0 leaves it unset, the hosted default).
func newStackBudgetProgram(t *testing.T, stackSize uint64) (ssa.Program, ssa.Package) {
	t.Helper()
	prog := ssatest.NewProgram(t, nil)
	t.Cleanup(prog.Dispose)
	if stackSize != 0 {
		prog.SetPthreadStackSize(stackSize)
	}
	pkg := prog.NewPackage("stackmake", "stackmake")
	return prog, pkg
}

func int64SliceType(prog ssa.Program) ssa.Type {
	return prog.Type(types.NewSlice(types.Typ[types.Int64]), ssa.GoDecl)
}

func int64ArrayType(prog ssa.Program, n int64) ssa.Type {
	return prog.Type(types.NewArray(types.Typ[types.Int64], n), ssa.GoDecl)
}

// TestStackVarBudgetFollowsFixedGoroutineStacks pins both stack-residency
// gates on fixed-size goroutine stacks: the make backing-array eligibility,
// the stack-variable alloca limit, and the map bucket limit. The baremetal
// fiber scheduler injects the target default-stack-size through
// SetPthreadStackSize, so a pi1000-class make([]int64, 3501) must fall back
// to the heap helpers there instead of reserving a 28 KiB alloca on a 2 KiB
// fiber stack. The budget leaves the fiber overhead (guard margin, entry
// chain, fixed frames) unclaimed: with a 2048-byte stack the usable budget
// is 1792 bytes, so a budget-exact 2048-byte array must also go to the heap.
// Hosted targets - no injected size, or the large test default - keep their
// historical 64/128 KiB limits.
func TestStackVarBudgetFollowsFixedGoroutineStacks(t *testing.T) {
	const capacity = 3501 // 3501 * 8 bytes = 28008, the pi1000-class alloc

	makeIR := func(stackSize uint64, cap int64) string {
		_, pkg := newStackBudgetProgram(t, stackSize)
		fn := pkg.NewFunc("makeit", ssa.NoArgsNoRet, ssa.GoDecl)
		b := fn.MakeBody(1)
		slice := int64SliceType(pkg.Prog)
		length := pkg.Prog.IntVal(uint64(cap), pkg.Prog.Int())
		b.MakeSliceStackCapacity(slice, length, length, cap)
		b.Return()
		b.EndBuild()
		return pkg.String()
	}

	allocIR := func(stackSize uint64, n int64) string {
		_, pkg := newStackBudgetProgram(t, stackSize)
		fn := pkg.NewFunc("localloc", ssa.NoArgsNoRet, ssa.GoDecl)
		b := fn.MakeBody(1)
		b.AllocEx(int64ArrayType(pkg.Prog, n), false, true)
		b.Return()
		b.EndBuild()
		return pkg.String()
	}

	if ir := makeIR(0, capacity); !strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("unset stack size lost the stack buffer path:\n%s", ir)
	}
	if ir := makeIR(32<<20, capacity); !strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("large injected stack size lost the stack buffer path:\n%s", ir)
	}
	if ir := makeIR(2048, capacity); strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("2 KiB fiber stack still reserved a %d-byte stack buffer:\n%s", capacity*8, ir)
	} else if !strings.Contains(ir, "runtime.MakeSlice") {
		t.Errorf("overflowed stack make did not fall back to the heap helper:\n%s", ir)
	}
	// 2048 - 256 overhead = 1792 usable bytes: a budget-exact 2048-byte array
	// must fall to the heap, 224 elements (1792 bytes) stay on the stack, and
	// one element more (1800 bytes) must not.
	if ir := makeIR(2048, 256); strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("budget-exact stack make consumed the whole fiber stack:\n%s", ir)
	}
	if ir := makeIR(2048, 224); !strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("budget-exact stack make lost the stack buffer path:\n%s", ir)
	}
	if ir := makeIR(2048, 225); strings.Contains(ir, "MakeSliceBuffer") {
		t.Errorf("stack make one element over the budget kept the stack buffer:\n%s", ir)
	}

	if ir := allocIR(0, capacity); !strings.Contains(ir, "alloca") {
		t.Errorf("unset stack size moved a %d-byte local off the stack:\n%s", capacity*8, ir)
	}
	if ir := allocIR(32<<20, capacity); !strings.Contains(ir, "alloca") {
		t.Errorf("large injected stack size moved a %d-byte local off the stack:\n%s", capacity*8, ir)
	}
	if ir := allocIR(2048, capacity); strings.Contains(ir, "alloca") || !strings.Contains(ir, "AllocAtomic") {
		t.Errorf("2 KiB fiber stack kept a %d-byte local alloca:\n%s", capacity*8, ir)
	}
	if ir := allocIR(2048, 256); strings.Contains(ir, "alloca") {
		t.Errorf("budget-exact stack local consumed the whole fiber stack:\n%s", ir)
	}
	// 224 * 8 bytes == 1792 usable bytes exactly stays a stack local.
	if ir := allocIR(2048, 224); !strings.Contains(ir, "alloca") {
		t.Errorf("budget-exact stack local lost stack storage:\n%s", ir)
	}
	if ir := allocIR(2048, 225); strings.Contains(ir, "alloca") {
		t.Errorf("stack local one element over the budget kept its alloca:\n%s", ir)
	}
}

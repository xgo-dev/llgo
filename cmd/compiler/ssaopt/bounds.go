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
	"go/constant"
	"go/token"
	"math"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/types"
)

// BoundsPass analyzes slice, array, string indexing, slicing, and
// slice-to-array-pointer conversions and emits read-only BoundsPlan decisions.
//
// Safety invariants:
//   - At -O0, only DisableBoundsChecks or constant-in-range accesses elide
//     bounds checks (matching existing llvmgen constant folding and keeping
//     panic evaluation order exact).
//   - At >= -O1, dominator-tree facts (branch guards, canonical loop induction
//     variables, bitmask/widening ranges, and dominating checked accesses) may
//     elide redundant bounds checks with a verified ProofAnchor.
//   - *array nil dereference checks are NEVER controlled by BoundsPlan; llvmgen
//     always preserves AssertNilDeref independently of CheckBounds.
//   - Wide indices on 32-bit targets (e.g. int64/uint64 on Wasm32/386) are
//     checked in full width before eliding; narrowing conversions are never
//     unwrapped.
type BoundsPass struct{}

func (BoundsPass) Name() string { return "bounds" }

func (BoundsPass) Run(fn *ssa.Function, cfg Config, plan *FunctionPlan) error {
	if fn == nil || len(fn.Blocks) == 0 {
		return nil
	}

	wordBits := 64
	if cfg.TypeCtx != nil && cfg.TypeCtx.Spec().WordSize > 0 {
		wordBits = int(cfg.TypeCtx.Spec().WordSize) * 8
	}

	var facts *boundsFacts
	if cfg.Optimizing() && !cfg.DisableBoundsChecks {
		facts = collectBoundsFacts(fn, wordBits)
	}

	var bounds map[ssa.Instruction]BoundsPlan
	record := func(ins ssa.Instruction, bp BoundsPlan) {
		if bounds == nil {
			bounds = make(map[ssa.Instruction]BoundsPlan)
		}
		bounds[ins] = bp
	}

	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.IndexAddr:
				record(ins, planIndexBounds(ins, ins.X, ins.Index, cfg, facts, wordBits))
			case *ssa.Index:
				record(ins, planIndexBounds(ins, ins.X, ins.Index, cfg, facts, wordBits))
			case *ssa.Slice:
				record(ins, planSliceBounds(ins, cfg, facts, wordBits))
			case *ssa.SliceToArrayPointer:
				record(ins, planSliceToArrayPtrBounds(ins, cfg, facts))
			}
		}
	}

	plan.Bounds = bounds
	return nil
}

func planIndexBounds(ins ssa.Instruction, base, idx ssa.Value, cfg Config, facts *boundsFacts, wordBits int) BoundsPlan {
	if cfg.DisableBoundsChecks {
		return BoundsPlan{CheckBounds: false}
	}
	// Constant-in-range against statically known length (valid at all opt levels).
	if clen, ok := staticValueLen(base); ok {
		if iv, ok := constNonNegInt64(idx); ok && iv < clen {
			return BoundsPlan{CheckBounds: false}
		}
	}
	if !cfg.Optimizing() || facts == nil {
		return BoundsPlan{CheckBounds: true}
	}
	if anchor, ok := facts.proveIndexInBounds(ins, base, idx); ok {
		return BoundsPlan{CheckBounds: false, ProofAnchor: anchor}
	}
	return BoundsPlan{CheckBounds: true}
}

func planSliceToArrayPtrBounds(ins *ssa.SliceToArrayPointer, cfg Config, facts *boundsFacts) BoundsPlan {
	if cfg.DisableBoundsChecks {
		return BoundsPlan{CheckBounds: false}
	}
	ptr, ok := types.Unalias(ins.Type()).Underlying().(*types.Pointer)
	if !ok {
		return BoundsPlan{CheckBounds: true}
	}
	arr, ok := types.Unalias(ptr.Elem()).Underlying().(*types.Array)
	if !ok {
		return BoundsPlan{CheckBounds: true}
	}
	need := arr.Len()
	if need == 0 {
		return BoundsPlan{CheckBounds: false}
	}
	if clen, ok := staticValueLen(ins.X); ok && clen >= need {
		return BoundsPlan{CheckBounds: false}
	}
	if !cfg.Optimizing() || facts == nil {
		return BoundsPlan{CheckBounds: true}
	}
	if minL, anchor, ok := facts.knownMinLen(ins, ins.X); ok && minL >= need {
		return BoundsPlan{CheckBounds: false, ProofAnchor: anchor}
	}
	return BoundsPlan{CheckBounds: true}
}

func planSliceBounds(ins *ssa.Slice, cfg Config, facts *boundsFacts, wordBits int) BoundsPlan {
	if cfg.DisableBoundsChecks {
		return BoundsPlan{CheckBounds: false}
	}
	// s[:] on any slice, string, or *array is always in bounds (note: *array
	// nil deref is still checked unconditionally by llvmgen).
	if ins.Low == nil && ins.High == nil && ins.Max == nil {
		return BoundsPlan{CheckBounds: false}
	}

	// Constant bounds against static length/capacity (valid at all opt levels).
	if clen, ccap, ok := staticValueLenCap(ins.X); ok {
		upper := clen
		if isSliceType(ins.X.Type()) {
			upper = ccap
		}
		lowVal, lowOK := int64(0), true
		if ins.Low != nil {
			lowVal, lowOK = constNonNegInt64(ins.Low)
		}
		highVal, highOK := clen, true
		if ins.High != nil {
			highVal, highOK = constNonNegInt64(ins.High)
		}
		maxVal, maxOK := upper, true
		if ins.Max != nil {
			maxVal, maxOK = constNonNegInt64(ins.Max)
		}
		if lowOK && highOK && maxOK &&
			0 <= lowVal && lowVal <= highVal && highVal <= maxVal && maxVal <= upper &&
			fitsNativeSigned(lowVal, wordBits) && fitsNativeSigned(highVal, wordBits) && fitsNativeSigned(maxVal, wordBits) {
			return BoundsPlan{CheckBounds: false}
		}
	}

	if !cfg.Optimizing() || facts == nil || ins.Max != nil {
		return BoundsPlan{CheckBounds: true}
	}

	// Two-index slicing proofs at >= -O1:
	// Case 1: s[low:] (High == nil, Max == nil) when 0 <= low <= len(s).
	if ins.Low != nil && ins.High == nil {
		if anchor, ok := facts.proveInZeroToLenInclusive(ins, ins.X, ins.Low); ok {
			return BoundsPlan{CheckBounds: false, ProofAnchor: anchor}
		}
	}
	// Case 2: s[:high] (Low == nil, Max == nil) when 0 <= high <= len(s).
	if ins.Low == nil && ins.High != nil {
		if anchor, ok := facts.proveInZeroToLenInclusive(ins, ins.X, ins.High); ok {
			return BoundsPlan{CheckBounds: false, ProofAnchor: anchor}
		}
	}
	// Case 3: s[c1:c2] where 0 <= c1 <= c2 <= knownMinLen(s).
	if ins.Low != nil && ins.High != nil {
		lowC, lowOK := constNonNegInt64(ins.Low)
		highC, highOK := constNonNegInt64(ins.High)
		if lowOK && highOK && lowC <= highC &&
			fitsNativeSigned(lowC, wordBits) && fitsNativeSigned(highC, wordBits) {
			if minL, anchor, ok := facts.knownMinLen(ins, ins.X); ok && highC <= minL {
				return BoundsPlan{CheckBounds: false, ProofAnchor: anchor}
			}
		}
	}
	return BoundsPlan{CheckBounds: true}
}

// staticValueLen returns the compile-time constant length of v if known.
func staticValueLen(v ssa.Value) (int64, bool) {
	clen, _, ok := staticValueLenCap(v)
	return clen, ok
}

func staticValueLenCap(v ssa.Value) (int64, int64, bool) {
	if v == nil || v.Type() == nil {
		return 0, 0, false
	}
	switch t := types.Unalias(v.Type()).Underlying().(type) {
	case *types.Array:
		return t.Len(), t.Len(), true
	case *types.Pointer:
		if arr, ok := types.Unalias(t.Elem()).Underlying().(*types.Array); ok {
			return arr.Len(), arr.Len(), true
		}
	case *types.Basic:
		if t.Kind() == types.String {
			if c, ok := v.(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.String {
				n := int64(len(constant.StringVal(c.Value)))
				return n, n, true
			}
		}
	case *types.Slice:
		switch s := v.(type) {
		case *ssa.MakeSlice:
			l, okL := constNonNegInt64(s.Len)
			c, okC := constNonNegInt64(s.Cap)
			if okL && okC && l <= c {
				return l, c, true
			}
		case *ssa.Slice:
			if s.Low == nil && s.High == nil && s.Max == nil {
				return staticValueLenCap(s.X)
			}
		}
	}
	return 0, 0, false
}

func isSliceType(t types.Type) bool {
	if t == nil {
		return false
	}
	_, ok := types.Unalias(t).Underlying().(*types.Slice)
	return ok
}

func constInt64(v ssa.Value) (int64, bool) {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil || c.Value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(c.Value)
}

func constNonNegInt64(v ssa.Value) (int64, bool) {
	c, ok := v.(*ssa.Const)
	if !ok || c.Value == nil || c.Value.Kind() != constant.Int {
		return 0, false
	}
	if constant.Sign(c.Value) < 0 {
		return 0, false
	}
	if n, exact := constant.Int64Val(c.Value); exact && n >= 0 {
		return n, true
	}
	return 0, false
}

func fitsNativeSigned(v int64, wordBits int) bool {
	if v < 0 {
		return false
	}
	if wordBits <= 32 {
		return v <= math.MaxInt32
	}
	return true
}

func intTypeInfo(t types.Type, wordBits int) (bits int, signed bool, ok bool) {
	if t == nil {
		return 0, false, false
	}
	b, isBasic := types.Unalias(t).Underlying().(*types.Basic)
	if !isBasic {
		return 0, false, false
	}
	switch b.Kind() {
	case types.Int8:
		return 8, true, true
	case types.Int16:
		return 16, true, true
	case types.Int32:
		return 32, true, true
	case types.Int64:
		return 64, true, true
	case types.Int:
		return wordBits, true, true
	case types.Uint8:
		return 8, false, true
	case types.Uint16:
		return 16, false, true
	case types.Uint32:
		return 32, false, true
	case types.Uint64:
		return 64, false, true
	case types.Uint, types.Uintptr:
		return wordBits, false, true
	default:
		return 0, false, false
	}
}

func isLenCallOf(v ssa.Value) (ssa.Value, bool) {
	call, ok := v.(*ssa.Call)
	if !ok || len(call.Call.Args) != 1 {
		return nil, false
	}
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "len" {
		return nil, false
	}
	return call.Call.Args[0], true
}

func isCapCallOf(v ssa.Value) (ssa.Value, bool) {
	call, ok := v.(*ssa.Call)
	if !ok || len(call.Call.Args) != 1 {
		return nil, false
	}
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "cap" {
		return nil, false
	}
	return call.Call.Args[0], true
}

type valBaseKey struct {
	val  ssa.Value
	base ssa.Value
}

type anchoredFact struct {
	anchor ssa.Instruction
	blk    *ssa.BasicBlock
	idx    int
}

type constBoundFact struct {
	limit  int64 // exclusive upper bound for ltConst, inclusive lower bound for minLen
	anchor ssa.Instruction
	blk    *ssa.BasicBlock
	idx    int
}

type boundsFacts struct {
	hasDom   bool
	wordBits int
	instrPos map[ssa.Instruction]struct {
		blk *ssa.BasicBlock
		idx int
	}

	// Instruction-level dominating facts (hold after blk.Instrs[idx] in all dominated blocks).
	nonNegInstr  map[ssa.Value][]anchoredFact
	ltLenInstr   map[valBaseKey][]anchoredFact
	leLenInstr   map[valBaseKey][]anchoredFact
	minLenInstr  map[ssa.Value][]constBoundFact
	ltConstInstr map[ssa.Value][]constBoundFact

	// Edge-level facts (hold in single-predecessor successor blk and all blocks it dominates).
	nonNegEdge  map[*ssa.BasicBlock]map[ssa.Value]ssa.Instruction
	ltLenEdge   map[*ssa.BasicBlock]map[valBaseKey]ssa.Instruction
	leLenEdge   map[*ssa.BasicBlock]map[valBaseKey]ssa.Instruction
	minLenEdge  map[*ssa.BasicBlock]map[ssa.Value]constBoundFact
	ltConstEdge map[*ssa.BasicBlock]map[ssa.Value]constBoundFact

	// Canonical non-negative loop induction variable phis (anchor = loop guard *ssa.If).
	indVarNonNeg map[*ssa.Phi]ssa.Instruction
}

func collectBoundsFacts(fn *ssa.Function, wordBits int) *boundsFacts {
	f := &boundsFacts{
		wordBits: wordBits,
		instrPos: make(map[ssa.Instruction]struct {
			blk *ssa.BasicBlock
			idx int
		}, 64),
		nonNegInstr:  make(map[ssa.Value][]anchoredFact),
		ltLenInstr:   make(map[valBaseKey][]anchoredFact),
		leLenInstr:   make(map[valBaseKey][]anchoredFact),
		minLenInstr:  make(map[ssa.Value][]constBoundFact),
		ltConstInstr: make(map[ssa.Value][]constBoundFact),
		nonNegEdge:   make(map[*ssa.BasicBlock]map[ssa.Value]ssa.Instruction),
		ltLenEdge:    make(map[*ssa.BasicBlock]map[valBaseKey]ssa.Instruction),
		leLenEdge:    make(map[*ssa.BasicBlock]map[valBaseKey]ssa.Instruction),
		minLenEdge:   make(map[*ssa.BasicBlock]map[ssa.Value]constBoundFact),
		ltConstEdge:  make(map[*ssa.BasicBlock]map[ssa.Value]constBoundFact),
		indVarNonNeg: make(map[*ssa.Phi]ssa.Instruction),
	}
	if fn == nil || len(fn.Blocks) == 0 {
		return f
	}
	f.hasDom = fn.Blocks[0].Dominates(fn.Blocks[0])
	if !f.hasDom {
		return f
	}

	addNonNegEdge := func(succ *ssa.BasicBlock, v ssa.Value, anchor ssa.Instruction) {
		if len(succ.Preds) != 1 || v == nil {
			return
		}
		if f.nonNegEdge[succ] == nil {
			f.nonNegEdge[succ] = make(map[ssa.Value]ssa.Instruction)
		}
		f.nonNegEdge[succ][v] = anchor
	}
	addLtLenEdge := func(succ *ssa.BasicBlock, idx, base ssa.Value, anchor ssa.Instruction) {
		if len(succ.Preds) != 1 || idx == nil || base == nil {
			return
		}
		if f.ltLenEdge[succ] == nil {
			f.ltLenEdge[succ] = make(map[valBaseKey]ssa.Instruction)
		}
		f.ltLenEdge[succ][valBaseKey{idx, base}] = anchor
		if f.leLenEdge[succ] == nil {
			f.leLenEdge[succ] = make(map[valBaseKey]ssa.Instruction)
		}
		f.leLenEdge[succ][valBaseKey{idx, base}] = anchor
	}
	addLeLenEdge := func(succ *ssa.BasicBlock, idx, base ssa.Value, anchor ssa.Instruction) {
		if len(succ.Preds) != 1 || idx == nil || base == nil {
			return
		}
		if f.leLenEdge[succ] == nil {
			f.leLenEdge[succ] = make(map[valBaseKey]ssa.Instruction)
		}
		f.leLenEdge[succ][valBaseKey{idx, base}] = anchor
	}
	addMinLenEdge := func(succ *ssa.BasicBlock, base ssa.Value, minL int64, anchor ssa.Instruction) {
		if len(succ.Preds) != 1 || base == nil || minL <= 0 {
			return
		}
		if f.minLenEdge[succ] == nil {
			f.minLenEdge[succ] = make(map[ssa.Value]constBoundFact)
		}
		if prev, ok := f.minLenEdge[succ][base]; !ok || minL > prev.limit {
			f.minLenEdge[succ][base] = constBoundFact{limit: minL, anchor: anchor}
		}
	}
	addLtConstEdge := func(succ *ssa.BasicBlock, v ssa.Value, limit int64, anchor ssa.Instruction) {
		if len(succ.Preds) != 1 || v == nil || limit <= 0 {
			return
		}
		if f.ltConstEdge[succ] == nil {
			f.ltConstEdge[succ] = make(map[ssa.Value]constBoundFact)
		}
		if prev, ok := f.ltConstEdge[succ][v]; !ok || limit < prev.limit {
			f.ltConstEdge[succ][v] = constBoundFact{limit: limit, anchor: anchor}
		}
	}

	recordCmp := func(succ *ssa.BasicBlock, op token.Token, lhs, rhs ssa.Value, anchor ssa.Instruction) {
		// Normalize GTR/GEQ into LSS/LEQ by swapping operands.
		switch op {
		case token.GTR:
			op = token.LSS
			lhs, rhs = rhs, lhs
		case token.GEQ:
			op = token.LEQ
			lhs, rhs = rhs, lhs
		}

		// Pattern: lhs < rhs or lhs <= rhs or lhs == rhs
		switch op {
		case token.LSS:
			// c < rhs -> rhs >= c + 1
			if c, ok := constInt64(lhs); ok {
				if c >= -1 {
					addNonNegEdge(succ, rhs, anchor)
				}
				if c >= 0 && c < math.MaxInt64 {
					if base, ok := isLenCallOf(rhs); ok {
						addMinLenEdge(succ, base, c+1, anchor)
					}
				}
			}
			// lhs < c
			if c, ok := constNonNegInt64(rhs); ok && c > 0 {
				addLtConstEdge(succ, lhs, c, anchor)
			}
			// lhs < len(base)
			if base, ok := isLenCallOf(rhs); ok {
				addLtLenEdge(succ, lhs, base, anchor)
			}
		case token.LEQ:
			// c <= rhs
			if c, ok := constInt64(lhs); ok {
				if c >= 0 {
					addNonNegEdge(succ, rhs, anchor)
					if base, ok := isLenCallOf(rhs); ok {
						addMinLenEdge(succ, base, c, anchor)
					}
				}
			}
			// lhs <= c
			if c, ok := constNonNegInt64(rhs); ok && c < math.MaxInt64 {
				addLtConstEdge(succ, lhs, c+1, anchor)
			}
			// lhs <= len(base)
			if base, ok := isLenCallOf(rhs); ok {
				addLeLenEdge(succ, lhs, base, anchor)
			}
		case token.EQL:
			// len(base) == c or c == len(base)
			if c, ok := constNonNegInt64(rhs); ok {
				addNonNegEdge(succ, lhs, anchor)
				if c < math.MaxInt64 {
					addLtConstEdge(succ, lhs, c+1, anchor)
				}
				if base, ok := isLenCallOf(lhs); ok {
					addMinLenEdge(succ, base, c, anchor)
				}
			}
			if c, ok := constNonNegInt64(lhs); ok {
				addNonNegEdge(succ, rhs, anchor)
				if c < math.MaxInt64 {
					addLtConstEdge(succ, rhs, c+1, anchor)
				}
				if base, ok := isLenCallOf(rhs); ok {
					addMinLenEdge(succ, base, c, anchor)
				}
			}
			if base, ok := isLenCallOf(rhs); ok {
				addLeLenEdge(succ, lhs, base, anchor)
			}
			if base, ok := isLenCallOf(lhs); ok {
				addLeLenEdge(succ, rhs, base, anchor)
			}
		}
	}

	for _, blk := range fn.Blocks {
		for idx, ins := range blk.Instrs {
			f.instrPos[ins] = struct {
				blk *ssa.BasicBlock
				idx int
			}{blk: blk, idx: idx}

			switch ins := ins.(type) {
			case *ssa.IndexAddr:
				f.nonNegInstr[ins.Index] = append(f.nonNegInstr[ins.Index], anchoredFact{anchor: ins, blk: blk, idx: idx})
				f.ltLenInstr[valBaseKey{ins.Index, ins.X}] = append(f.ltLenInstr[valBaseKey{ins.Index, ins.X}], anchoredFact{anchor: ins, blk: blk, idx: idx})
				f.leLenInstr[valBaseKey{ins.Index, ins.X}] = append(f.leLenInstr[valBaseKey{ins.Index, ins.X}], anchoredFact{anchor: ins, blk: blk, idx: idx})
				if c, ok := constNonNegInt64(ins.Index); ok && c < math.MaxInt64 {
					f.minLenInstr[ins.X] = append(f.minLenInstr[ins.X], constBoundFact{limit: c + 1, anchor: ins, blk: blk, idx: idx})
				}
			case *ssa.Index:
				f.nonNegInstr[ins.Index] = append(f.nonNegInstr[ins.Index], anchoredFact{anchor: ins, blk: blk, idx: idx})
				f.ltLenInstr[valBaseKey{ins.Index, ins.X}] = append(f.ltLenInstr[valBaseKey{ins.Index, ins.X}], anchoredFact{anchor: ins, blk: blk, idx: idx})
				f.leLenInstr[valBaseKey{ins.Index, ins.X}] = append(f.leLenInstr[valBaseKey{ins.Index, ins.X}], anchoredFact{anchor: ins, blk: blk, idx: idx})
				if c, ok := constNonNegInt64(ins.Index); ok && c < math.MaxInt64 {
					f.minLenInstr[ins.X] = append(f.minLenInstr[ins.X], constBoundFact{limit: c + 1, anchor: ins, blk: blk, idx: idx})
				}
			case *ssa.SliceToArrayPointer:
				if ptr, ok := types.Unalias(ins.Type()).Underlying().(*types.Pointer); ok {
					if arr, ok := types.Unalias(ptr.Elem()).Underlying().(*types.Array); ok && arr.Len() > 0 {
						f.minLenInstr[ins.X] = append(f.minLenInstr[ins.X], constBoundFact{limit: arr.Len(), anchor: ins, blk: blk, idx: idx})
					}
				}
			case *ssa.If:
				if len(blk.Succs) != 2 {
					continue
				}
				bin, ok := ins.Cond.(*ssa.BinOp)
				if !ok {
					continue
				}
				switch bin.Op {
				case token.LSS, token.LEQ, token.GTR, token.GEQ, token.EQL, token.NEQ:
					recordCmp(blk.Succs[0], bin.Op, bin.X, bin.Y, ins)
					if negOp, ok := negateCmpOp(bin.Op); ok {
						recordCmp(blk.Succs[1], negOp, bin.X, bin.Y, ins)
					}
				}
			}
		}
	}

	// Detect canonical non-negative loop induction variables:
	//   phi = Phi(init >= 0, Add(phi, step > 0))
	// where the backedge block is dominated by a strict upper-bound guard
	// phi < len(s) (or phi < c with c <= maxSigned - step), guaranteeing that
	// phi + step cannot overflow and phi stays >= 0 on every iteration.
	f.detectInductionVars(fn)

	return f
}

func (f *boundsFacts) detectInductionVars(fn *ssa.Function) {
	maxSigned := int64(math.MaxInt64)
	if f.wordBits <= 32 {
		maxSigned = math.MaxInt32
	}
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			phi, ok := ins.(*ssa.Phi)
			if !ok || len(phi.Edges) != 2 || len(blk.Preds) != 2 {
				continue
			}
			bits, signed, isInt := intTypeInfo(phi.Type(), f.wordBits)
			if !isInt || !signed {
				continue
			}
			typeMax := maxSigned
			if bits < 64 {
				typeMax = (int64(1) << (bits - 1)) - 1
			}
			for initIdx := 0; initIdx < 2; initIdx++ {
				stepIdx := 1 - initIdx
				initVal := phi.Edges[initIdx]
				stepVal := phi.Edges[stepIdx]
				backPred := blk.Preds[stepIdx]

				// Init value must be proven non-negative (typically constant 0).
				initConst, ok := constNonNegInt64(initVal)
				if !ok || initConst < 0 {
					continue
				}
				// Step must be phi + positive constant.
				add, ok := stepVal.(*ssa.BinOp)
				if !ok || add.Op != token.ADD {
					continue
				}
				var stepConst int64
				var stepOK bool
				if add.X == phi {
					stepConst, stepOK = constNonNegInt64(add.Y)
				} else if add.Y == phi {
					stepConst, stepOK = constNonNegInt64(add.X)
				}
				if !stepOK || stepConst <= 0 {
					continue
				}
				// Backedge predecessor must be dominated by a strict upper bound
				// phi < len(s) (when step == 1, len(s) <= typeMax so phi+1 <= typeMax)
				// or phi < C with C <= typeMax - (step - 1).
				if guard, ok := f.findGuardDominatingBlock(backPred, phi, stepConst, typeMax); ok {
					f.indVarNonNeg[phi] = guard
					break
				}
			}
		}
	}
}

func (f *boundsFacts) findGuardDominatingBlock(blk *ssa.BasicBlock, phi *ssa.Phi, step, typeMax int64) (ssa.Instruction, bool) {
	for dom := blk; dom != nil; dom = dom.Idom() {
		if step == 1 {
			for k, anchor := range f.ltLenEdge[dom] {
				if k.val == phi {
					return anchor, true
				}
			}
		}
		if cb, ok := f.ltConstEdge[dom][phi]; ok {
			// phi < cb.limit => phi <= cb.limit - 1 => phi + step <= cb.limit - 1 + step.
			if cb.limit-1 <= typeMax-step {
				return cb.anchor, true
			}
		}
	}
	return nil, false
}

func negateCmpOp(op token.Token) (token.Token, bool) {
	switch op {
	case token.LSS:
		return token.GEQ, true
	case token.LEQ:
		return token.GTR, true
	case token.GTR:
		return token.LEQ, true
	case token.GEQ:
		return token.LSS, true
	case token.EQL:
		return token.NEQ, true
	case token.NEQ:
		return token.EQL, true
	default:
		return token.ILLEGAL, false
	}
}

func (f *boundsFacts) strictlyDominates(proofBlk *ssa.BasicBlock, proofIdx int, target ssa.Instruction) bool {
	tp, ok := f.instrPos[target]
	if !ok {
		return false
	}
	if proofBlk == tp.blk {
		return proofIdx < tp.idx
	}
	return proofBlk.Dominates(tp.blk)
}

func (f *boundsFacts) proveNonNeg(target ssa.Instruction, v ssa.Value) (ssa.Instruction, bool) {
	if v == nil {
		return nil, false
	}
	// Unsigned types are inherently non-negative.
	if _, signed, ok := intTypeInfo(v.Type(), f.wordBits); ok && !signed {
		return nil, true
	}
	if _, ok := constNonNegInt64(v); ok {
		return nil, true
	}
	if _, ok := isLenCallOf(v); ok {
		return nil, true
	}
	if _, ok := isCapCallOf(v); ok {
		return nil, true
	}
	if conv, ok := v.(*ssa.Convert); ok {
		srcBits, srcSigned, okS := intTypeInfo(conv.X.Type(), f.wordBits)
		dstBits, _, okD := intTypeInfo(conv.Type(), f.wordBits)
		if okS && okD && !srcSigned && srcBits < dstBits {
			return nil, true
		}
		if okS && okD && srcBits <= dstBits {
			if a, ok := f.proveNonNeg(target, conv.X); ok {
				return a, true
			}
		}
	}
	if bin, ok := v.(*ssa.BinOp); ok {
		switch bin.Op {
		case token.AND:
			if c, ok := constNonNegInt64(bin.Y); ok && c >= 0 {
				return nil, true
			}
			if c, ok := constNonNegInt64(bin.X); ok && c >= 0 {
				return nil, true
			}
		case token.SHR:
			if a, ok := f.proveNonNeg(target, bin.X); ok {
				return a, true
			}
		case token.REM:
			if c, ok := constNonNegInt64(bin.Y); ok && c > 0 {
				if a, ok := f.proveNonNeg(target, bin.X); ok {
					return a, true
				}
			}
		}
	}
	if phi, ok := v.(*ssa.Phi); ok {
		if guard, ok := f.indVarNonNeg[phi]; ok {
			// Verify guard dominates target before relying on it as ProofAnchor.
			if gp, ok := f.instrPos[guard]; ok && f.strictlyDominates(gp.blk, gp.idx, target) {
				return guard, true
			}
			// Even if the guard itself is inside the loop body (not dominating phi's
			// header), the induction invariant guarantees phi >= 0 everywhere phi is live!
			return nil, true
		}
	}
	for _, af := range f.nonNegInstr[v] {
		if f.strictlyDominates(af.blk, af.idx, target) {
			return af.anchor, true
		}
	}
	if tp, ok := f.instrPos[target]; ok {
		for dom := tp.blk; dom != nil; dom = dom.Idom() {
			if anchor, ok := f.nonNegEdge[dom][v]; ok {
				return anchor, true
			}
		}
	}
	return nil, false
}

// intrinsicLtConst returns an exclusive upper bound C (meaning v < C) that
// holds from v's definition alone (e.g. constant, bitmask, widening from uint8,
// modulo by positive constant).
func (f *boundsFacts) intrinsicLtConst(target ssa.Instruction, v ssa.Value) (int64, bool) {
	if c, ok := constNonNegInt64(v); ok && c < math.MaxInt64 {
		return c + 1, true
	}
	if bits, signed, ok := intTypeInfo(v.Type(), f.wordBits); ok && !signed && bits <= 16 {
		return int64(1) << bits, true
	}
	if conv, ok := v.(*ssa.Convert); ok {
		srcBits, srcSigned, okS := intTypeInfo(conv.X.Type(), f.wordBits)
		dstBits, _, okD := intTypeInfo(conv.Type(), f.wordBits)
		if okS && okD && !srcSigned && srcBits < dstBits && srcBits <= 30 {
			return int64(1) << srcBits, true
		}
		if okS && okD && srcBits <= dstBits {
			if _, ok := f.proveNonNeg(target, conv.X); ok {
				if lim, ok := f.intrinsicLtConst(target, conv.X); ok {
					return lim, true
				}
			}
		}
	}
	if bin, ok := v.(*ssa.BinOp); ok {
		switch bin.Op {
		case token.AND:
			if c, ok := constNonNegInt64(bin.Y); ok && c < math.MaxInt64 {
				return c + 1, true
			}
			if c, ok := constNonNegInt64(bin.X); ok && c < math.MaxInt64 {
				return c + 1, true
			}
		case token.REM:
			if c, ok := constNonNegInt64(bin.Y); ok && c > 0 {
				if _, ok := f.proveNonNeg(target, bin.X); ok {
					return c, true
				}
			}
		}
	}
	return 0, false
}

func (f *boundsFacts) knownLtConst(target ssa.Instruction, v ssa.Value) (int64, ssa.Instruction, bool) {
	best := int64(math.MaxInt64)
	var bestAnchor ssa.Instruction
	found := false
	if lim, ok := f.intrinsicLtConst(target, v); ok {
		best = lim
		found = true
	}
	for _, cf := range f.ltConstInstr[v] {
		if f.strictlyDominates(cf.blk, cf.idx, target) && (!found || cf.limit < best) {
			best = cf.limit
			bestAnchor = cf.anchor
			found = true
		}
	}
	if tp, ok := f.instrPos[target]; ok {
		for dom := tp.blk; dom != nil; dom = dom.Idom() {
			if cf, ok := f.ltConstEdge[dom][v]; ok && (!found || cf.limit < best) {
				best = cf.limit
				bestAnchor = cf.anchor
				found = true
			}
		}
	}
	return best, bestAnchor, found
}

func (f *boundsFacts) knownMinLen(target ssa.Instruction, base ssa.Value) (int64, ssa.Instruction, bool) {
	best := int64(0)
	var bestAnchor ssa.Instruction
	found := false
	if clen, ok := staticValueLen(base); ok {
		best = clen
		found = true
	}
	for _, cf := range f.minLenInstr[base] {
		if f.strictlyDominates(cf.blk, cf.idx, target) && cf.limit > best {
			best = cf.limit
			bestAnchor = cf.anchor
			found = true
		}
	}
	if tp, ok := f.instrPos[target]; ok {
		for dom := tp.blk; dom != nil; dom = dom.Idom() {
			if cf, ok := f.minLenEdge[dom][base]; ok && cf.limit > best {
				best = cf.limit
				bestAnchor = cf.anchor
				found = true
			}
		}
	}
	return best, bestAnchor, found
}

func (f *boundsFacts) lookupLtLen(target ssa.Instruction, idx, base ssa.Value) (ssa.Instruction, bool) {
	k := valBaseKey{idx, base}
	for _, af := range f.ltLenInstr[k] {
		if f.strictlyDominates(af.blk, af.idx, target) {
			return af.anchor, true
		}
	}
	if tp, ok := f.instrPos[target]; ok {
		for dom := tp.blk; dom != nil; dom = dom.Idom() {
			if anchor, ok := f.ltLenEdge[dom][k]; ok {
				return anchor, true
			}
		}
	}
	return nil, false
}

func (f *boundsFacts) lookupLeLen(target ssa.Instruction, idx, base ssa.Value) (ssa.Instruction, bool) {
	k := valBaseKey{idx, base}
	for _, af := range f.leLenInstr[k] {
		if f.strictlyDominates(af.blk, af.idx, target) {
			return af.anchor, true
		}
	}
	if tp, ok := f.instrPos[target]; ok {
		for dom := tp.blk; dom != nil; dom = dom.Idom() {
			if anchor, ok := f.leLenEdge[dom][k]; ok {
				return anchor, true
			}
		}
	}
	return nil, false
}

func (f *boundsFacts) proveIndexInBounds(target ssa.Instruction, base, idx ssa.Value) (ssa.Instruction, bool) {
	// Must prove idx >= 0 in full index width.
	nonNegAnchor, ok := f.proveNonNeg(target, idx)
	if !ok {
		return nil, false
	}

	// Direct proof: idx < len(base).
	if anchor, ok := f.lookupLtLen(target, idx, base); ok {
		return anchor, true
	}

	// Constant/range proof: idx < C and len(base) >= C (works for both slices
	// with dominating len guards / prior checked accesses AND fixed arrays!).
	if lim, ltAnchor, ok := f.knownLtConst(target, idx); ok {
		if minL, minAnchor, ok := f.knownMinLen(target, base); ok && lim <= minL {
			if minAnchor != nil {
				return minAnchor, true
			}
			if ltAnchor != nil {
				return ltAnchor, true
			}
			return nonNegAnchor, true
		}
	}

	// Widening conversion: idx = Convert(src) where src is non-negative and
	// proven < len(base) without truncation.
	if conv, ok := idx.(*ssa.Convert); ok {
		srcBits, _, okS := intTypeInfo(conv.X.Type(), f.wordBits)
		dstBits, _, okD := intTypeInfo(conv.Type(), f.wordBits)
		if okS && okD && srcBits <= dstBits {
			if _, ok := f.proveNonNeg(target, conv.X); ok {
				if anchor, ok := f.lookupLtLen(target, conv.X, base); ok {
					return anchor, true
				}
			}
		}
	}

	return nil, false
}

func (f *boundsFacts) proveInZeroToLenInclusive(target ssa.Instruction, base, idx ssa.Value) (ssa.Instruction, bool) {
	nonNegAnchor, ok := f.proveNonNeg(target, idx)
	if !ok {
		return nil, false
	}
	if anchor, ok := f.lookupLeLen(target, idx, base); ok {
		return anchor, true
	}
	if c, ok := constNonNegInt64(idx); ok && fitsNativeSigned(c, f.wordBits) {
		if c == 0 {
			return nil, true
		}
		if minL, minAnchor, ok := f.knownMinLen(target, base); ok && c <= minL {
			return minAnchor, true
		}
	}
	if lim, ltAnchor, ok := f.knownLtConst(target, idx); ok && lim > 0 {
		// idx <= lim - 1 <= minLen(base)
		if minL, minAnchor, ok := f.knownMinLen(target, base); ok && lim-1 <= minL {
			if minAnchor != nil {
				return minAnchor, true
			}
			if ltAnchor != nil {
				return ltAnchor, true
			}
			return nonNegAnchor, true
		}
	}
	return nil, false
}

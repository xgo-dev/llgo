package build

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// fixSSAOrder applies an SSA fixup for stdlib compatibility.
//
// go/ssa follows the spec's operand evaluation rules (only calls/receives/logical
// ops are ordered). Some stdlib code relies on the Go compiler's de-facto choice
// of delaying non-call operands, e.g. patterns like:
//
//	var o T
//	return o, o.mutate()
//
// where mutate has a pointer receiver and sets fields in o. go/ssa may materialize
// the first return value as a load before the call, which makes o appear unchanged
// to the return value in our backend.
//
// This pass moves loads of local allocs that feed a Return result and have no
// intervening executable use before that Return to after any intervening calls
// that use the same alloc pointer, matching the behavior of the Go compiler for
// the stdlib cases we rely on (e.g. crypto/x509.ParseOID).
func fixSSAOrder(pkg *ssa.Package) {
	if pkg == nil {
		return
	}
	visited := make(map[*ssa.Function]struct{})
	visitFn := func(fn *ssa.Function) {
		if fn == nil {
			return
		}
		if _, ok := visited[fn]; ok {
			return
		}
		visited[fn] = struct{}{}
		fixSSAOrderFunc(fn)
	}

	for _, mem := range pkg.Members {
		switch m := mem.(type) {
		case *ssa.Function:
			visitFn(m)
		case *ssa.Type:
			if tn, ok := m.Object().(*types.TypeName); ok {
				fixSSAOrderMethods(pkg, tn.Type(), visitFn)
				fixSSAOrderMethods(pkg, types.NewPointer(tn.Type()), visitFn)
			}
		}
	}
}

func fixSSAOrderMethods(pkg *ssa.Package, typ types.Type, visitFn func(*ssa.Function)) {
	if pkg == nil || pkg.Prog == nil || typ == nil {
		return
	}
	mset := pkg.Prog.MethodSets.MethodSet(typ)
	for i, n := 0, mset.Len(); i < n; i++ {
		if fn := pkg.Prog.MethodValue(mset.At(i)); fn != nil {
			visitFn(fn)
		}
	}
}

func fixSSAOrderFunc(fn *ssa.Function) {
	if fn == nil || len(fn.Blocks) == 0 {
		return
	}
	for _, b := range fn.Blocks {
		fixSSAOrderBlock(b)
	}
	for _, anon := range fn.AnonFuncs {
		fixSSAOrderFunc(anon)
	}
}

func movedValuesForIndices(instrs []ssa.Instruction, move map[int]struct{}) map[ssa.Value]struct{} {
	moved := make(map[ssa.Value]struct{}, len(move))
	for i := range move {
		if i < 0 || i >= len(instrs) {
			continue
		}
		if v, ok := instrs[i].(ssa.Value); ok && v != nil {
			moved[v] = struct{}{}
		}
	}
	return moved
}

// includeDebugRefsForMovedValues adds metadata-only uses of moved values to
// move. The moved value set is supplied by the caller so the same set can be
// reused by the subsequent SSA safety check without rescanning move.
func includeDebugRefsForMovedValues(instrs []ssa.Instruction, move map[int]struct{}, moved map[ssa.Value]struct{}, from, through int) {
	if from < 0 {
		from = 0
	}
	if through > len(instrs) {
		through = len(instrs)
	}
	for i := from; i < through; i++ {
		if _, moving := move[i]; moving {
			continue
		}
		ref, ok := instrs[i].(*ssa.DebugRef)
		if !ok {
			continue
		}
		for v := range moved {
			if instrUsesValue(ref, v) {
				move[i] = struct{}{}
				break
			}
		}
	}
}

func fixSSAOrderBlock(b *ssa.BasicBlock) {
	if b == nil || len(b.Instrs) == 0 {
		return
	}
	// Find the (only) Return; by construction it's a terminating instruction.
	retIdx := -1
	var ret *ssa.Return
	for i := len(b.Instrs) - 1; i >= 0; i-- {
		if r, ok := b.Instrs[i].(*ssa.Return); ok {
			retIdx = i
			ret = r
			break
		}
	}
	if retIdx < 0 || ret == nil {
		return
	}

	// For each return result that is a load from a local alloc, try to move the
	// load after any intervening calls that use the alloc pointer.
	for _, rv := range ret.Results {
		u, ok := rv.(*ssa.UnOp)
		if !ok || u.Op != token.MUL {
			continue
		}
		alloc, ok := u.X.(*ssa.Alloc)
		if !ok {
			continue
		}

		loadIdx := indexOfInstr(b.Instrs, u)
		if loadIdx < 0 || loadIdx >= retIdx {
			continue
		}

		// Find the last call between load and return that uses the alloc pointer.
		lastCallIdx := -1
		for i := loadIdx + 1; i < retIdx; i++ {
			ci, ok := b.Instrs[i].(ssa.CallInstruction)
			if !ok {
				continue
			}
			if callUsesValue(ci, alloc) {
				lastCallIdx = i
			}
		}
		if lastCallIdx < 0 {
			continue
		}

		// Bail if the alloc is written between the load and return.
		// Moving the load could otherwise observe a different value.
		writtenBeforeReturn := false
		for i := loadIdx + 1; i < retIdx; i++ {
			if storeWritesAlloc(b.Instrs[i], alloc) {
				writtenBeforeReturn = true
				break
			}
		}
		if writtenBeforeReturn {
			continue
		}

		// DebugRefs are metadata-only and move with the value they describe. Any
		// executable use before Return still makes reordering unsafe.
		movingIndices := map[int]struct{}{loadIdx: {}}
		moved := movedValuesForIndices(b.Instrs, movingIndices)
		includeDebugRefsForMovedValues(b.Instrs, movingIndices, moved, loadIdx+1, retIdx)
		moving := make(map[ssa.Instruction]struct{}, len(movingIndices))
		for i := range movingIndices {
			moving[b.Instrs[i]] = struct{}{}
		}
		usedBeforeReturn := false
		for i := loadIdx + 1; i < retIdx; i++ {
			if _, moving := movingIndices[i]; moving {
				continue
			}
			if instrUsesValue(b.Instrs[i], u) {
				usedBeforeReturn = true
				break
			}
		}
		if usedBeforeReturn {
			continue
		}

		b.Instrs = moveInstrsAfter(b.Instrs, moving, b.Instrs[lastCallIdx])
		retIdx = indexOfInstr(b.Instrs, ret)
	}
}

func indexOfInstr(instrs []ssa.Instruction, target ssa.Instruction) int {
	for i, ins := range instrs {
		if ins == target {
			return i
		}
	}
	return -1
}

func instrUsesValue(ins ssa.Instruction, v ssa.Value) bool {
	if ins == nil || v == nil {
		return false
	}
	for _, op := range ins.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
}

func callUsesValue(ci ssa.CallInstruction, v ssa.Value) bool {
	if ci == nil || v == nil {
		return false
	}
	c := ci.Common()
	if c == nil {
		return false
	}
	for _, op := range c.Operands(nil) {
		if op != nil && *op == v {
			return true
		}
	}
	return false
}

func storeWritesAlloc(ins ssa.Instruction, alloc *ssa.Alloc) bool {
	store, ok := ins.(*ssa.Store)
	if !ok || store == nil || alloc == nil {
		return false
	}
	return valueDependsOn(store.Addr, alloc, map[ssa.Value]struct{}{})
}

func valueDependsOn(v, target ssa.Value, seen map[ssa.Value]struct{}) bool {
	if v == nil || target == nil {
		return false
	}
	if v == target {
		return true
	}
	if _, ok := seen[v]; ok {
		return false
	}
	seen[v] = struct{}{}
	ins, ok := v.(ssa.Instruction)
	if !ok || ins == nil {
		return false
	}
	for _, op := range ins.Operands(nil) {
		if op != nil && valueDependsOn(*op, target, seen) {
			return true
		}
	}
	return false
}

// moveInstrsAfter moves selected instructions as a stable group immediately
// after anchor. The anchor must not be in moving; callers use an instruction
// that remains in the block. It returns instrs unchanged when moving is empty,
// or anchor is nil or absent.
func moveInstrsAfter(instrs []ssa.Instruction, moving map[ssa.Instruction]struct{}, anchor ssa.Instruction) []ssa.Instruction {
	if len(moving) == 0 || anchor == nil {
		return instrs
	}
	if _, ok := moving[anchor]; ok {
		panic("moveInstrsAfter: anchor is in moving set")
	}
	moved := make([]ssa.Instruction, 0, len(moving))
	remaining := make([]ssa.Instruction, 0, len(instrs))
	for _, instr := range instrs {
		if _, ok := moving[instr]; ok {
			moved = append(moved, instr)
			continue
		}
		remaining = append(remaining, instr)
	}
	for i, instr := range remaining {
		if instr == anchor {
			ret := make([]ssa.Instruction, 0, len(instrs))
			ret = append(ret, remaining[:i+1]...)
			ret = append(ret, moved...)
			ret = append(ret, remaining[i+1:]...)
			return ret
		}
	}
	return instrs
}

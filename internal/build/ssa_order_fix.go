package build

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// fixSSAOrder applies small SSA fixups for stdlib compatibility.
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
// This pass moves loads of local allocs used only for the final Return results
// to after any intervening calls that use the same alloc pointer, matching the
// behavior of the Go compiler for the stdlib cases we rely on (e.g. crypto/x509.ParseOID).
func fixSSAOrder(pkg *ssa.Package, files []*ast.File) {
	if pkg == nil {
		return
	}
	selectRecvAssigns := collectSingleCaseSelectRecvAssigns(files)
	visited := make(map[*ssa.Function]struct{})
	visitFn := func(fn *ssa.Function) {
		if fn == nil {
			return
		}
		if _, ok := visited[fn]; ok {
			return
		}
		visited[fn] = struct{}{}
		fixSSAOrderFunc(fn, selectRecvAssigns)
	}

	for _, mem := range pkg.Members {
		switch m := mem.(type) {
		case *ssa.Function:
			visitFn(m)
		case *ssa.Type:
			if tn, ok := m.Object().(*types.TypeName); ok {
				fixSSAOrderMethods(pkg, tn.Type(), visitFn)
			}
		}
	}
}

func fixSSAOrderMethods(pkg *ssa.Package, typ types.Type, visitFn func(*ssa.Function)) {
	if pkg == nil || pkg.Prog == nil || typ == nil {
		return
	}
	named, ok := typ.(*types.Named)
	if !ok {
		return
	}
	for i, n := 0, named.NumMethods(); i < n; i++ {
		if fn := pkg.Prog.FuncValue(named.Method(i)); fn != nil {
			visitFn(fn)
		}
	}
}

func fixSSAOrderFunc(fn *ssa.Function, selectRecvAssigns map[token.Pos]struct{}) {
	if fn == nil || len(fn.Blocks) == 0 {
		return
	}
	for _, b := range fn.Blocks {
		fixSSAOrderBlock(b)
		fixSingleCaseSelectRecvAssignBlock(b, selectRecvAssigns)
	}
	for _, anon := range fn.AnonFuncs {
		fixSSAOrderFunc(anon, selectRecvAssigns)
	}
}

func collectSingleCaseSelectRecvAssigns(files []*ast.File) map[token.Pos]struct{} {
	ret := make(map[token.Pos]struct{})
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			sel, ok := node.(*ast.SelectStmt)
			if !ok || sel.Body == nil || len(sel.Body.List) != 1 {
				return true
			}
			clause, ok := sel.Body.List[0].(*ast.CommClause)
			if !ok {
				return true
			}
			assign, ok := clause.Comm.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				return true
			}
			recv, ok := ast.Unparen(assign.Rhs[0]).(*ast.UnaryExpr)
			if !ok || recv.Op != token.ARROW {
				return true
			}
			ret[recv.OpPos] = struct{}{}
			return true
		})
	}
	return ret
}

func fixSingleCaseSelectRecvAssignBlock(b *ssa.BasicBlock, recvAssigns map[token.Pos]struct{}) {
	if b == nil || len(b.Instrs) == 0 || len(recvAssigns) == 0 {
		return
	}
	for {
		changed := false
		instrIndex := make(map[ssa.Instruction]int, len(b.Instrs))
		for i, ins := range b.Instrs {
			instrIndex[ins] = i
		}
		// Restart after a move because instruction indexes change. Each move
		// removes at least one root dependency from before its receive, so the
		// loop converges once no safely movable dependencies remain.
		for assignIdx, ins := range b.Instrs {
			var roots []ssa.Value
			var val ssa.Value
			switch instr := ins.(type) {
			case *ssa.Store:
				roots = []ssa.Value{instr.Addr}
				val = instr.Val
			case *ssa.MapUpdate:
				roots = []ssa.Value{instr.Map, instr.Key}
				val = instr.Value
			default:
				continue
			}
			recv, ok := selectRecvRoot(val, recvAssigns)
			if !ok {
				continue
			}
			recvInstr, ok := recv.(ssa.Instruction)
			if !ok {
				continue
			}
			recvIdx, found := instrIndex[recvInstr]
			if !found || recvIdx >= assignIdx {
				continue
			}
			if moveAssignDepsAfterRecv(b, roots, recv, recvIdx, assignIdx) {
				changed = true
				break
			}
		}
		if !changed {
			return
		}
	}
}

func selectRecvRoot(v ssa.Value, recvAssigns map[token.Pos]struct{}) (ssa.Value, bool) {
	switch v := v.(type) {
	case *ssa.UnOp:
		if v.Op == token.ARROW {
			_, ok := recvAssigns[v.Pos()]
			return v, ok
		}
	case *ssa.Extract:
		if u, ok := v.Tuple.(*ssa.UnOp); ok && u.Op == token.ARROW {
			_, found := recvAssigns[u.Pos()]
			return u, found
		}
	}
	return nil, false
}

func moveAssignDepsAfterRecv(b *ssa.BasicBlock, roots []ssa.Value, recv ssa.Value, recvIdx, assignIdx int) bool {
	move := make(map[int]struct{})
	seen := make(map[ssa.Value]struct{})
	for i := 0; i < recvIdx; i++ {
		v, ok := b.Instrs[i].(ssa.Value)
		if !ok || v == nil {
			continue
		}
		if valueDependsOnAny(roots, v, seen) && !valueDependsOnReusing(recv, v, seen) {
			move[i] = struct{}{}
		}
	}
	if len(move) == 0 {
		return false
	}
	if moveWouldBreakSSA(b.Instrs, move, recvIdx) {
		return false
	}
	deps := make([]ssa.Instruction, 0, len(move))
	next := make([]ssa.Instruction, 0, len(b.Instrs))
	for i, ins := range b.Instrs {
		if _, ok := move[i]; ok {
			deps = append(deps, ins)
			continue
		}
		next = append(next, ins)
		if i == recvIdx {
			next = append(next, deps...)
		}
	}
	b.Instrs = next
	return true
}

func moveWouldBreakSSA(instrs []ssa.Instruction, move map[int]struct{}, recvIdx int) bool {
	moved := make(map[ssa.Value]struct{}, len(move))
	for i := range move {
		if v, ok := instrs[i].(ssa.Value); ok && v != nil {
			moved[v] = struct{}{}
		}
	}
	for i := 0; i <= recvIdx && i < len(instrs); i++ {
		if _, moving := move[i]; moving {
			continue
		}
		for v := range moved {
			if instrUsesValue(instrs[i], v) {
				return true
			}
		}
	}
	return false
}

func valueDependsOnAny(values []ssa.Value, target ssa.Value, seen map[ssa.Value]struct{}) bool {
	for _, v := range values {
		if valueDependsOnReusing(v, target, seen) {
			return true
		}
	}
	return false
}

func valueDependsOnReusing(v, target ssa.Value, seen map[ssa.Value]struct{}) bool {
	clear(seen)
	return valueDependsOn(v, target, seen)
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
		var storeSeen map[ssa.Value]struct{}
		for i := loadIdx + 1; i < retIdx; i++ {
			if _, ok := b.Instrs[i].(*ssa.Store); !ok {
				continue
			}
			if storeSeen == nil {
				storeSeen = make(map[ssa.Value]struct{})
			}
			if storeWritesAlloc(b.Instrs[i], alloc, storeSeen) {
				writtenBeforeReturn = true
				break
			}
		}
		if writtenBeforeReturn {
			continue
		}

		// If the loaded value is used by any instruction between its current
		// position and the return (excluding return itself), moving it may place
		// its definition after one of those uses and break SSA form.
		usedBeforeReturn := false
		for i := loadIdx + 1; i < retIdx; i++ {
			if instrUsesValue(b.Instrs[i], u) {
				usedBeforeReturn = true
				break
			}
		}
		if usedBeforeReturn {
			continue
		}

		// Move the load right after the last call (but before Return).
		b.Instrs = moveInstr(b.Instrs, loadIdx, lastCallIdx+1)
		// Adjust retIdx for subsequent moves in this block.
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

func storeWritesAlloc(ins ssa.Instruction, alloc *ssa.Alloc, seen map[ssa.Value]struct{}) bool {
	store, ok := ins.(*ssa.Store)
	if !ok || store == nil || alloc == nil {
		return false
	}
	clear(seen)
	return valueDependsOn(store.Addr, alloc, seen)
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

// moveInstr moves instrs[from] to position to (like inserting before to),
// preserving relative order of other elements.
func moveInstr(instrs []ssa.Instruction, from, to int) []ssa.Instruction {
	if from < 0 || from >= len(instrs) {
		return instrs
	}
	if to < 0 {
		to = 0
	}
	if to > len(instrs) {
		to = len(instrs)
	}
	if from == to || from+1 == to {
		return instrs
	}

	ins := instrs[from]
	// Remove.
	copy(instrs[from:], instrs[from+1:])
	instrs = instrs[:len(instrs)-1]

	// Recompute insertion index after removal.
	if to > from {
		to--
	}
	if to < 0 {
		to = 0
	}
	if to > len(instrs) {
		to = len(instrs)
	}

	// Insert.
	instrs = append(instrs, nil)
	copy(instrs[to+1:], instrs[to:])
	instrs[to] = ins
	return instrs
}

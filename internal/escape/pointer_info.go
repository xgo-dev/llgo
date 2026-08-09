package escape

import (
	"math"
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

const (
	rangeUnassigned = int64(math.MinInt32)
	rangeUnknown    = int64(math.MaxInt32)
)

type memoryRange struct {
	offset int64
	size   int64
}

func unknownMemoryRange() memoryRange {
	return memoryRange{offset: rangeUnknown, size: rangeUnknown}
}

func (r memoryRange) isUnassigned() bool {
	return r.offset == rangeUnassigned && r.size == rangeUnassigned
}

func (r memoryRange) isUnknown() bool {
	return r.offset == rangeUnknown || r.size == rangeUnknown
}

func (r memoryRange) mayOverlap(other memoryRange) bool {
	if r.isUnknown() || other.isUnknown() {
		return true
	}
	return other.offset+other.size > r.offset && other.offset < r.offset+r.size
}

func (r *memoryRange) merge(other memoryRange) {
	if other.isUnassigned() {
		return
	}
	if r.isUnassigned() {
		*r = other
		return
	}
	if r.offset == rangeUnknown || other.offset == rangeUnknown {
		r.offset = rangeUnknown
	}
	if r.size == rangeUnknown || other.size == rangeUnknown {
		r.size = rangeUnknown
	}
	if r.offset == rangeUnknown && r.size == rangeUnknown {
		return
	}
	if r.offset == rangeUnknown {
		if other.size > r.size {
			r.size = other.size
		}
		return
	}
	if r.size == rangeUnknown {
		if other.offset < r.offset {
			r.offset = other.offset
		}
		return
	}
	r.offset = min(r.offset, other.offset)
	r.size = max(r.offset+r.size, other.offset+other.size) - r.offset
}

type accessKind uint8

const (
	accessMust accessKind = 1 << iota
	accessMay
	accessRead
	accessWrite
)

type memoryAccess struct {
	local  llvm.Value
	remote llvm.Value
	ranges []memoryRange
	kind   accessKind
}

func (a memoryAccess) isRead() bool  { return a.kind&accessRead != 0 }
func (a memoryAccess) isWrite() bool { return a.kind&accessWrite != 0 }
func (a memoryAccess) isMust() bool  { return a.kind&accessMust != 0 }

type offsetInfo struct {
	offsets []int64
}

func unknownOffsetInfo() offsetInfo {
	return offsetInfo{offsets: []int64{rangeUnknown}}
}

func (o offsetInfo) isUnknown() bool {
	return len(o.offsets) == 1 && o.offsets[0] == rangeUnknown
}

func (o offsetInfo) equal(other offsetInfo) bool {
	if len(o.offsets) != len(other.offsets) {
		return false
	}
	for index := range o.offsets {
		if o.offsets[index] != other.offsets[index] {
			return false
		}
	}
	return true
}

func (o *offsetInfo) merge(other offsetInfo) bool {
	if o.isUnknown() {
		return false
	}
	if other.isUnknown() {
		*o = unknownOffsetInfo()
		return true
	}
	seen := make(map[int64]struct{}, len(o.offsets)+len(other.offsets))
	for _, offset := range o.offsets {
		seen[offset] = struct{}{}
	}
	changed := false
	for _, offset := range other.offsets {
		if _, ok := seen[offset]; ok {
			continue
		}
		seen[offset] = struct{}{}
		o.offsets = append(o.offsets, offset)
		changed = true
	}
	if changed {
		sort.Slice(o.offsets, func(i, j int) bool { return o.offsets[i] < o.offsets[j] })
	}
	return changed
}

type pointerInfo struct {
	valid    bool
	accesses []memoryAccess
}

type pointerInfoState uint8

const (
	pointerInfoUnseen pointerInfoState = iota
	pointerInfoBuilding
	pointerInfoReady
)

type copyAnalysis struct {
	mod       llvm.Module
	td        llvm.TargetData
	infos     map[llvm.Value]*pointerInfo
	states    map[llvm.Value]pointerInfoState
	callSites map[llvm.Value][]llvm.Value
	allCalls  map[llvm.Value]bool
}

func newCopyAnalysis(mod llvm.Module) *copyAnalysis {
	a := &copyAnalysis{
		mod:       mod,
		td:        llvm.NewTargetData(mod.DataLayout()),
		infos:     make(map[llvm.Value]*pointerInfo),
		states:    make(map[llvm.Value]pointerInfoState),
		callSites: make(map[llvm.Value][]llvm.Value),
		allCalls:  make(map[llvm.Value]bool),
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if call := instr.IsACallInst(); !call.IsNil() {
					if callee := call.CalledValue().IsAFunction(); !callee.IsNil() {
						a.callSites[callee] = append(a.callSites[callee], call)
					}
				}
			}
		}
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		linkage := fn.Linkage()
		complete := linkage == llvm.InternalLinkage || linkage == llvm.PrivateLinkage
		for use := fn.FirstUse(); complete && !use.IsNil(); use = use.NextUse() {
			call := use.User().IsACallInst()
			complete = !call.IsNil() && call.CalledValue() == fn
		}
		a.allCalls[fn] = complete
	}
	return a
}

func (a *copyAnalysis) dispose() {
	a.td.Dispose()
}

func (a *copyAnalysis) pointerInfo(root llvm.Value) (*pointerInfo, bool) {
	switch a.states[root] {
	case pointerInfoReady:
		info := a.infos[root]
		return info, info.valid
	case pointerInfoBuilding:
		return nil, false
	}

	info := &pointerInfo{valid: true}
	a.infos[root] = info
	a.states[root] = pointerInfoBuilding
	info.valid = a.buildPointerInfo(root, info)
	a.states[root] = pointerInfoReady
	return info, info.valid
}

func (a *copyAnalysis) buildPointerInfo(root llvm.Value, info *pointerInfo) bool {
	offsets := map[llvm.Value]offsetInfo{root: {offsets: []int64{0}}}
	worklist := []llvm.Value{root}
	inWorklist := map[llvm.Value]bool{root: true}

	enqueue := func(value llvm.Value, incoming offsetInfo) {
		current, ok := offsets[value]
		if ok && !current.merge(incoming) {
			return
		}
		if !ok {
			current = offsetInfo{}
			current.merge(incoming)
			offsets[value] = current
		} else {
			offsets[value] = current
		}
		if !inWorklist[value] {
			worklist = append(worklist, value)
			inWorklist[value] = true
		}
	}

	for len(worklist) != 0 {
		value := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		inWorklist[value] = false
		valueOffsets := offsets[value]

		for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
			user := use.User()
			for operand := 0; operand < user.OperandsCount(); operand++ {
				if user.Operand(operand) != value {
					continue
				}

				if user.InstructionOpcode() == llvm.Store && operand == 0 {
					copies, ok := a.getPotentialCopiesOfStoredValue(user)
					if !ok {
						return false
					}
					for _, copy := range copies {
						enqueue(copy, valueOffsets)
					}
					continue
				}

				switch user.InstructionOpcode() {
				case llvm.GetElementPtr:
					if operand != 0 || user.Type().TypeKind() == llvm.VectorTypeKind {
						return false
					}
					enqueue(user, a.gepOffsets(user, valueOffsets))
				case llvm.BitCast, opcodeAddrSpaceCast:
					enqueue(user, valueOffsets)
				case llvm.Select:
					if operand != 1 && operand != 2 {
						return false
					}
					enqueue(user, valueOffsets)
				case llvm.PHI:
					current, exists := offsets[user]
					if exists && blockInCycle(user.InstructionParent()) && !current.equal(valueOffsets) {
						enqueue(user, unknownOffsetInfo())
					} else {
						enqueue(user, valueOffsets)
					}
				case llvm.Ret:
					fn := user.InstructionParent().Parent()
					if !a.allCalls[fn] {
						return false
					}
					for _, call := range a.callSites[fn] {
						enqueue(call, valueOffsets)
					}
				case llvm.Load:
					kind := accessRead | a.accessCertainty(root, value)
					a.addAccess(info, user, user, valueOffsets, int64(a.td.TypeStoreSize(user.Type())), kind)
				case llvm.Store:
					if operand != 1 {
						return false
					}
					kind := accessWrite | a.accessCertainty(root, value)
					a.addAccess(info, user, user, valueOffsets, int64(a.td.TypeStoreSize(user.Operand(0).Type())), kind)
				case opcodeAtomicRMW, opcodeAtomicCmpXchg:
					if operand != 0 {
						return false
					}
					kind := accessRead | accessWrite | a.accessCertainty(root, value)
					a.addAccess(info, user, user, valueOffsets, int64(a.td.TypeStoreSize(user.Operand(1).Type())), kind)
				// TODO: Handle llvm.Invoke if LLGo starts emitting it.
				case llvm.Call:
					if !a.addCallAccesses(info, root, value, valueOffsets, user, operand) {
						return false
					}
				default:
					return false
				}
			}
		}
	}
	return true
}

func (a *copyAnalysis) accessCertainty(root, pointer llvm.Value) accessKind {
	objects := a.underlyingObjects(pointer)
	if len(objects) == 1 && objects[0] == root {
		return accessMust
	}
	return accessMay
}

func (a *copyAnalysis) addAccess(info *pointerInfo, local, remote llvm.Value, offsets offsetInfo, size int64, kind accessKind) {
	ranges := make([]memoryRange, 0, len(offsets.offsets))
	if offsets.isUnknown() || size == rangeUnknown {
		ranges = append(ranges, unknownMemoryRange())
	} else {
		for _, offset := range offsets.offsets {
			ranges = append(ranges, memoryRange{offset: offset, size: size})
		}
	}
	if len(ranges) != 1 {
		kind &^= accessMust
		kind |= accessMay
	}
	for index := range info.accesses {
		access := &info.accesses[index]
		if access.local != local || access.remote != remote {
			continue
		}
		for _, candidate := range ranges {
			found := false
			for rangeIndex := range access.ranges {
				if access.ranges[rangeIndex].offset == candidate.offset {
					access.ranges[rangeIndex].merge(candidate)
					found = true
					break
				}
			}
			if !found {
				access.ranges = append(access.ranges, candidate)
			}
		}
		if len(access.ranges) != 1 {
			access.kind &^= accessMust
			access.kind |= accessMay
		}
		access.kind |= kind
		if access.kind&accessMay != 0 {
			access.kind &^= accessMust
		}
		return
	}
	info.accesses = append(info.accesses, memoryAccess{local: local, remote: remote, ranges: ranges, kind: kind})
}

func (a *copyAnalysis) gepOffsets(gep llvm.Value, base offsetInfo) offsetInfo {
	if base.isUnknown() {
		return unknownOffsetInfo()
	}
	current := append([]int64(nil), base.offsets...)
	typ := gep.GEPSourceElementType()
	for operand := 1; operand < gep.OperandsCount(); operand++ {
		indices, ok := a.potentialConstants(gep.Operand(operand), make(map[llvm.Value]bool))
		if !ok || len(indices) == 0 {
			return unknownOffsetInfo()
		}
		var additions []int64
		if operand == 1 {
			scale := int64(a.td.TypeAllocSize(typ))
			for _, index := range indices {
				additions = append(additions, index*scale)
			}
		} else {
			switch typ.TypeKind() {
			case llvm.StructTypeKind:
				if len(indices) != 1 || indices[0] < 0 || indices[0] >= int64(typ.StructElementTypesCount()) {
					return unknownOffsetInfo()
				}
				field := int(indices[0])
				additions = append(additions, int64(a.td.ElementOffset(typ, field)))
				typ = typ.StructElementTypes()[field]
				current = addOffsetProduct(current, additions)
				continue
			case llvm.ArrayTypeKind, llvm.VectorTypeKind:
				elem := typ.ElementType()
				scale := int64(a.td.TypeAllocSize(elem))
				for _, index := range indices {
					additions = append(additions, index*scale)
				}
				typ = elem
			default:
				return unknownOffsetInfo()
			}
		}
		current = addOffsetProduct(current, additions)
	}
	return offsetInfo{offsets: uniqueSortedOffsets(current)}
}

func addOffsetProduct(base, additions []int64) []int64 {
	result := make([]int64, 0, len(base)*len(additions))
	for _, value := range base {
		for _, addition := range additions {
			result = append(result, value+addition)
		}
	}
	return result
}

func uniqueSortedOffsets(offsets []int64) []int64 {
	sort.Slice(offsets, func(i, j int) bool { return offsets[i] < offsets[j] })
	result := offsets[:0]
	for _, offset := range offsets {
		if len(result) == 0 || result[len(result)-1] != offset {
			result = append(result, offset)
		}
	}
	return result
}

func (a *copyAnalysis) potentialConstants(value llvm.Value, visiting map[llvm.Value]bool) ([]int64, bool) {
	if constant := value.IsAConstantInt(); !constant.IsNil() {
		return []int64{constant.SExtValue()}, true
	}
	if visiting[value] {
		return nil, false
	}
	visiting[value] = true
	defer delete(visiting, value)

	var values []int64
	switch value.InstructionOpcode() {
	case llvm.Select:
		for operand := 1; operand <= 2; operand++ {
			set, ok := a.potentialConstants(value.Operand(operand), visiting)
			if !ok {
				return nil, false
			}
			values = append(values, set...)
		}
	case llvm.PHI:
		for index := 0; index < value.IncomingCount(); index++ {
			set, ok := a.potentialConstants(value.IncomingValue(index), visiting)
			if !ok {
				return nil, false
			}
			values = append(values, set...)
		}
	default:
		return nil, false
	}
	return uniqueSortedOffsets(values), true
}

func (a *copyAnalysis) underlyingObjects(value llvm.Value) []llvm.Value {
	objects := make(map[llvm.Value]struct{})
	var visit func(llvm.Value, map[llvm.Value]bool)
	visit = func(current llvm.Value, path map[llvm.Value]bool) {
		if path[current] {
			return
		}
		nextPath := make(map[llvm.Value]bool, len(path)+1)
		for seen := range path {
			nextPath[seen] = true
		}
		nextPath[current] = true

		switch current.InstructionOpcode() {
		case llvm.GetElementPtr, llvm.BitCast, opcodeAddrSpaceCast:
			visit(current.Operand(0), nextPath)
		case llvm.Select:
			visit(current.Operand(1), nextPath)
			visit(current.Operand(2), nextPath)
		case llvm.PHI:
			for index := 0; index < current.IncomingCount(); index++ {
				visit(current.IncomingValue(index), nextPath)
			}
		default:
			objects[current] = struct{}{}
		}
	}
	visit(value, make(map[llvm.Value]bool))
	result := make([]llvm.Value, 0, len(objects))
	for object := range objects {
		result = append(result, object)
	}
	return result
}

func (a *copyAnalysis) addCallAccesses(info *pointerInfo, root, pointer llvm.Value, offsets offsetInfo, call llvm.Value, operand int) bool {
	calleeOperand := callCalleeOperand(call)
	if operand == calleeOperand {
		return true
	}
	if strings.HasPrefix(call.CalledValue().Name(), "llvm.lifetime.") {
		return true
	}
	if !call.IsAMemCpyInst().IsNil() || !call.IsAMemMoveInst().IsNil() || !call.IsAMemSetInst().IsNil() {
		if operand > 1 {
			return true
		}
		size := int64(rangeUnknown)
		if call.OperandsCount() > 2 {
			if length := call.Operand(2).IsAConstantInt(); !length.IsNil() {
				size = int64(length.SExtValue())
			}
		}
		kind := accessMust | accessWrite
		if operand == 1 {
			kind = accessMust | accessRead
		}
		a.addAccess(info, call, call, offsets, size, kind)
		return true
	}

	callee := call.CalledValue().IsAFunction()
	if callee.IsNil() || operand >= callee.ParamsCount() {
		return false
	}
	if !callee.IsDeclaration() {
		calleeInfo, ok := a.pointerInfo(callee.Param(operand))
		if !ok {
			return false
		}
		for _, access := range calleeInfo.accesses {
			translated := access
			translated.local = call
			translated.ranges = translateRanges(access.ranges, offsets)
			if len(translated.ranges) != 1 {
				translated.kind &^= accessMust
				translated.kind |= accessMay
			}
			info.accesses = append(info.accesses, translated)
		}
		return true
	}

	if !callArgumentHasAttribute(call, callee, operand, "nocapture") {
		return false
	}
	if callArgumentHasAttribute(call, callee, operand, "readnone") {
		return true
	}
	kind := accessMay | accessRead | accessWrite
	if callArgumentHasAttribute(call, callee, operand, "readonly") {
		kind = accessMay | accessRead
	}
	a.addAccess(info, call, call, unknownOffsetInfo(), rangeUnknown, kind)
	return true
}

func translateRanges(ranges []memoryRange, offsets offsetInfo) []memoryRange {
	if offsets.isUnknown() {
		return []memoryRange{unknownMemoryRange()}
	}
	var result []memoryRange
	for _, base := range offsets.offsets {
		for _, current := range ranges {
			if current.isUnknown() {
				return []memoryRange{unknownMemoryRange()}
			}
			result = append(result, memoryRange{offset: base + current.offset, size: current.size})
		}
	}
	return result
}

func callArgumentHasAttribute(call, callee llvm.Value, argument int, name string) bool {
	kind := llvm.AttributeKindID(name)
	index := argument + 1
	if attr := call.GetCallSiteEnumAttribute(index, kind); !attr.IsNil() {
		return true
	}
	return !callee.GetEnumAttributeAtIndex(index, kind).IsNil()
}

func (a *copyAnalysis) getPotentialCopiesOfStoredValue(store llvm.Value) ([]llvm.Value, bool) {
	objects := a.underlyingObjects(store.Operand(1))
	potentialCopies := make(map[llvm.Value]struct{})

	for _, object := range objects {
		if !object.IsAUndefValue().IsNil() {
			continue
		}
		if !object.IsAConstantPointerNull().IsNil() {
			if store.Operand(1) == object && !functionNullPointerIsValid(store.InstructionParent().Parent()) {
				continue
			}
			return nil, false
		}
		if !a.supportedUnderlyingObject(object) {
			return nil, false
		}
		info, ok := a.pointerInfo(object)
		if !ok {
			return nil, false
		}
		copies, ok := a.interferingLoadCopies(store, object, info)
		if !ok {
			return nil, false
		}
		for _, copy := range copies {
			potentialCopies[copy] = struct{}{}
		}
	}

	result := make([]llvm.Value, 0, len(potentialCopies))
	for copy := range potentialCopies {
		result = append(result, copy)
	}
	return result, true
}

func functionNullPointerIsValid(fn llvm.Value) bool {
	return !fn.GetStringAttributeAtIndex(-1, "null-pointer-is-valid").IsNil()
}

func (a *copyAnalysis) supportedUnderlyingObject(object llvm.Value) bool {
	if !object.IsAAllocaInst().IsNil() {
		return true
	}
	if global := object.IsAGlobalVariable(); !global.IsNil() {
		linkage := global.Linkage()
		return linkage == llvm.InternalLinkage || linkage == llvm.PrivateLinkage || global.IsGlobalConstant() && !global.Initializer().IsNil()
	}
	if call := object.IsACallInst(); !call.IsNil() {
		callee := call.CalledValue().IsAFunction()
		if callee.IsNil() {
			return false
		}
		kind := llvm.AttributeKindID("noalias")
		return !call.GetCallSiteEnumAttribute(0, kind).IsNil() || !callee.GetEnumAttributeAtIndex(0, kind).IsNil()
	}
	return false
}

func (a *copyAnalysis) interferingLoadCopies(store, object llvm.Value, info *pointerInfo) ([]llvm.Value, bool) {
	query := memoryRange{offset: rangeUnassigned, size: rangeUnassigned}
	for _, access := range info.accesses {
		if access.local != store && access.remote != store {
			continue
		}
		for _, current := range access.ranges {
			query.merge(current)
		}
	}
	if query.isUnassigned() {
		return nil, false
	}

	blockers := make(map[llvm.Value]bool)
	for _, access := range info.accesses {
		if !access.isWrite() || !access.isMust() || access.remote == store {
			continue
		}
		for _, current := range access.ranges {
			if current == query && !query.isUnknown() {
				blockers[access.remote] = true
			}
		}
	}

	copies := make(map[llvm.Value]struct{})
	threadLocal := !object.IsAAllocaInst().IsNil() || object.IsThreadLocal()
	for _, access := range info.accesses {
		if !access.isRead() {
			continue
		}
		for _, current := range access.ranges {
			if !query.mayOverlap(current) {
				continue
			}
			if threadLocal && !a.potentiallyReachable(store, access.remote, blockers) {
				continue
			}
			if query.isUnknown() || current.isUnknown() || current != query {
				return nil, false
			}
			if access.remote.IsALoadInst().IsNil() {
				return nil, false
			}
			copies[access.remote] = struct{}{}
		}
	}

	result := make([]llvm.Value, 0, len(copies))
	for copy := range copies {
		result = append(result, copy)
	}
	return result, true
}

func (a *copyAnalysis) potentiallyReachable(from, to llvm.Value, blockers map[llvm.Value]bool) bool {
	fromFn := from.InstructionParent().Parent()
	toFn := to.InstructionParent().Parent()
	if fromFn == toFn {
		if intraFunctionReachable(from, to, blockers) {
			return true
		}
		if !canReachFunctionReturn(from, blockers) {
			return false
		}
		norecurse := llvm.AttributeKindID("norecurse")
		return fromFn.GetEnumFunctionAttribute(norecurse).IsNil()
	}

	worklist := []llvm.Value{from}
	visited := make(map[llvm.Value]bool)
	for len(worklist) != 0 {
		current := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		if current == to {
			return true
		}
		if visited[current] || current != from && blockers[current] {
			continue
		}
		visited[current] = true
		worklist = append(worklist, a.instructionSuccessors(current)...)
	}
	return false
}

func intraFunctionReachable(from, to llvm.Value, blockers map[llvm.Value]bool) bool {
	worklist := []llvm.Value{from}
	visited := make(map[llvm.Value]bool)
	for len(worklist) != 0 {
		current := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		if current == to {
			return true
		}
		if visited[current] || current != from && blockers[current] {
			continue
		}
		visited[current] = true
		if next := llvm.NextInstruction(current); !next.IsNil() {
			worklist = append(worklist, next)
			continue
		}
		terminator := current.InstructionParent().LastInstruction()
		for index := 0; index < terminator.SuccessorsCount(); index++ {
			if first := terminator.Successor(index).FirstInstruction(); !first.IsNil() {
				worklist = append(worklist, first)
			}
		}
	}
	return false
}

func canReachFunctionReturn(from llvm.Value, blockers map[llvm.Value]bool) bool {
	fn := from.InstructionParent().Parent()
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		terminator := block.LastInstruction()
		if terminator.InstructionOpcode() == llvm.Ret && intraFunctionReachable(from, terminator, blockers) {
			return true
		}
	}
	return false
}

func (a *copyAnalysis) instructionSuccessors(instr llvm.Value) []llvm.Value {
	var result []llvm.Value
	if next := llvm.NextInstruction(instr); !next.IsNil() {
		result = append(result, next)
	} else {
		terminator := instr.InstructionParent().LastInstruction()
		for index := 0; index < terminator.SuccessorsCount(); index++ {
			if first := terminator.Successor(index).FirstInstruction(); !first.IsNil() {
				result = append(result, first)
			}
		}
	}

	if call := instr.IsACallInst(); !call.IsNil() {
		if callee := call.CalledValue().IsAFunction(); !callee.IsNil() && !callee.IsDeclaration() {
			if first := callee.EntryBasicBlock().FirstInstruction(); !first.IsNil() {
				result = append(result, first)
			}
		}
	}
	if instr.InstructionOpcode() == llvm.Ret {
		fn := instr.InstructionParent().Parent()
		for _, call := range a.callSites[fn] {
			if next := llvm.NextInstruction(call); !next.IsNil() {
				result = append(result, next)
			}
		}
	}
	return result
}

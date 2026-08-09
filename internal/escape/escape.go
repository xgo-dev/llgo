// Package escape implements LLGo's LLVM-IR escape analysis and heap-to-stack
// transform.
package escape

import (
	"math"
	"sort"
	"strings"

	llabi "github.com/goplus/llgo/internal/abi"
	"github.com/xgo-dev/llvm"
)

const (
	runtimePrefix = "github.com/goplus/llgo/runtime/internal/runtime."
	runtimeAllocZ = runtimePrefix + "AllocZ"
	runtimeAllocU = runtimePrefix + "AllocU"

	// github.com/xgo-dev/llvm does not expose these LLVM 19 C opcode names.
	opcodeAtomicCmpXchg = llvm.Opcode(56)
	opcodeAtomicRMW     = llvm.Opcode(57)
	opcodeAddrSpaceCast = llvm.Opcode(60)
)

type paramKey struct {
	fn    llvm.Value
	param int
}

type locationEdgeKind uint8

const (
	locationFlow locationEdgeKind = iota
	locationDeref
	locationHeap
	locationMutator
	locationCallee
)

type location struct {
	value  llvm.Value
	result int
}

type locationEdge struct {
	to   location
	kind locationEdgeKind
}

// locationGraph records diagnostic pointer flows but is never queried by the
// optimization analyses.
type locationGraph struct {
	edges map[location]map[locationEdge]struct{}
}

func newLocationGraph() locationGraph {
	return locationGraph{edges: make(map[location]map[locationEdge]struct{})}
}

func valueLocation(value llvm.Value) location {
	return location{value: value, result: -1}
}

func resultLocation(fn llvm.Value, result int) location {
	return location{value: fn, result: result}
}

func (g *locationGraph) add(from location, edge locationEdge) {
	edges := g.edges[from]
	if edges == nil {
		edges = make(map[locationEdge]struct{})
		g.edges[from] = edges
	}
	edges[edge] = struct{}{}
}

func (g *locationGraph) addFlow(from, to llvm.Value) {
	g.add(valueLocation(from), locationEdge{to: valueLocation(to), kind: locationFlow})
}

func (g *locationGraph) addDeref(from, to llvm.Value) {
	g.add(valueLocation(from), locationEdge{to: valueLocation(to), kind: locationDeref})
}

func (g *locationGraph) addHeap(from llvm.Value) {
	g.add(valueLocation(from), locationEdge{kind: locationHeap})
}

func (g *locationGraph) addMutator(from llvm.Value) {
	g.add(valueLocation(from), locationEdge{kind: locationMutator})
}

func (g *locationGraph) addCallee(from llvm.Value) {
	g.add(valueLocation(from), locationEdge{kind: locationCallee})
}

func (g *locationGraph) addResult(from, fn llvm.Value, result int) {
	g.add(valueLocation(from), locationEdge{to: resultLocation(fn, result), kind: locationFlow})
}

func (g *locationGraph) addCallResult(fn llvm.Value, result int, call llvm.Value) {
	g.add(resultLocation(fn, result), locationEdge{to: valueLocation(call), kind: locationFlow})
}

const (
	leakHeap = iota
	leakMutator
	leakCallee
	leakResult0
	numEscResults = 5
)

type leaks [leakResult0 + numEscResults]uint8

func (l leaks) get(index int) int {
	return int(l[index]) - 1
}

func (l *leaks) add(index, level int) {
	if old := l.get(index); old >= 0 && old <= level {
		return
	}
	value := level + 1
	if value > math.MaxUint8 {
		value = math.MaxUint8
	}
	l[index] = uint8(value)
}

func (l *leaks) optimize() {
	heap := l.get(leakHeap)
	if heap < 0 {
		return
	}
	for index := leakMutator; index < len(l); index++ {
		if l.get(index) >= heap {
			l[index] = 0
		}
	}
}

func (g *locationGraph) summarize(root paramKey) leaks {
	var summary leaks
	start := valueLocation(root.fn.Param(root.param))
	best := map[location]int{start: 0}
	worklist := []location{start}

	for len(worklist) != 0 {
		current := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		level := best[current]
		for edge := range g.edges[current] {
			switch edge.kind {
			case locationFlow, locationDeref:
				nextLevel := level
				if edge.kind == locationDeref {
					nextLevel++
				}
				if edge.to.result >= 0 && edge.to.value == root.fn {
					if edge.to.result < numEscResults {
						summary.add(leakResult0+edge.to.result, nextLevel)
					}
					continue
				}
				if old, ok := best[edge.to]; !ok || nextLevel < old {
					best[edge.to] = nextLevel
					worklist = append(worklist, edge.to)
				}
			case locationHeap:
				summary.add(leakHeap, level)
			case locationMutator:
				summary.add(leakMutator, level)
			case locationCallee:
				summary.add(leakCallee, level)
			}
		}
	}
	summary.optimize()
	return summary
}

type analyzer struct {
	noCapture     map[paramKey]bool
	noCaptureKeys []paramKey
	locations     *locationGraph
	copies        *copyAnalysis
}

func newAnalyzer(mod llvm.Module, diagnostics bool) *analyzer {
	a := &analyzer{
		noCapture: make(map[paramKey]bool),
		copies:    newCopyAnalysis(mod),
	}
	if diagnostics {
		locations := newLocationGraph()
		a.locations = &locations
	}
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() || isRuntimeFunction(fn) {
			continue
		}
		for index, param := range fn.Params() {
			if !isPointer(param.Type()) {
				continue
			}
			key := paramKey{fn: fn, param: index}
			a.noCapture[key] = true
			a.noCaptureKeys = append(a.noCaptureKeys, key)
		}
	}
	return a
}

func isRuntimeFunction(fn llvm.Value) bool {
	return strings.HasPrefix(fn.Name(), runtimePrefix)
}

func isPointer(typ llvm.Type) bool {
	return typ.TypeKind() == llvm.PointerTypeKind
}

func (a *analyzer) solveNoCapture() {
	for {
		changed := false
		for _, key := range a.noCaptureKeys {
			if a.noCapture[key] && !a.parameterNoCapture(key) {
				a.noCapture[key] = false
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

type useState struct {
	user    llvm.Value
	operand int
}

type walkResult struct {
	escaped   bool
	alignment int
}

func (a *analyzer) addUses(worklist *[]useState, value llvm.Value) {
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		for operand := 0; operand < user.OperandsCount(); operand++ {
			if user.Operand(operand) == value {
				state := useState{user: user, operand: operand}
				*worklist = append(*worklist, state)
			}
		}
	}
}

type useAction uint8

const (
	useSafe useAction = iota
	useFollow
	useCapture
)

func (a *analyzer) checkForAllUses(root llvm.Value, classify func(useState) useAction) bool {
	worklist := make([]useState, 0, 8)
	a.addUses(&worklist, root)
	visited := make(map[useState]struct{})
	valid := true

	for len(worklist) != 0 {
		state := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		if _, ok := visited[state]; ok {
			continue
		}
		visited[state] = struct{}{}

		if state.user.InstructionOpcode() == llvm.Store && state.operand == 0 {
			copies, ok := a.copies.getPotentialCopiesOfStoredValue(state.user)
			if ok {
				for _, copy := range copies {
					a.addUses(&worklist, copy)
				}
				continue
			}
		}

		switch classify(state) {
		case useFollow:
			a.addUses(&worklist, state.user)
		case useCapture:
			valid = false
		}
	}
	return valid
}

func (a *analyzer) parameterNoCapture(key paramKey) bool {
	return a.checkForAllUses(key.fn.Param(key.param), a.classifyNoCaptureUse)
}

func callCalleeOperand(call llvm.Value) int {
	called := call.CalledValue()
	index := -1
	for operand := 0; operand < call.OperandsCount(); operand++ {
		if call.Operand(operand) == called {
			index = operand
		}
	}
	return index
}

func definedCallParameter(call llvm.Value, argument int) (paramKey, bool) {
	fn := call.CalledValue().IsAFunction()
	if fn.IsNil() || fn.IntrinsicID() != 0 || fn.IsDeclaration() || isRuntimeFunction(fn) || argument >= fn.ParamsCount() {
		return paramKey{}, false
	}
	return paramKey{fn: fn, param: argument}, true
}

func (a *analyzer) classifyNoCaptureUse(state useState) useAction {
	user := state.user
	switch user.InstructionOpcode() {
	case llvm.Call, llvm.Invoke:
		if a.callArgumentNoCapture(state) {
			return useSafe
		}
	case llvm.Load:
		return useSafe
	case llvm.Store:
		if state.operand == 1 {
			return useSafe
		}
	case opcodeAtomicRMW:
		if state.operand == 0 {
			return useSafe
		}
	case opcodeAtomicCmpXchg:
		if state.operand == 0 {
			return useSafe
		}
	case llvm.VAArg:
		return useSafe
	case llvm.GetElementPtr:
		if state.operand == 0 && user.Type().TypeKind() != llvm.VectorTypeKind {
			return useFollow
		}
	case llvm.BitCast, llvm.PHI, opcodeAddrSpaceCast:
		return useFollow
	case llvm.Select:
		if state.operand == 1 || state.operand == 2 {
			return useFollow
		}
	}
	return useCapture
}

func (a *analyzer) requiredAlignment(root paramKey) int {
	alignment := 0
	worklist := []paramKey{root}
	visited := make(map[paramKey]struct{})

	for len(worklist) != 0 {
		key := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		if _, ok := visited[key]; ok {
			continue
		}
		visited[key] = struct{}{}

		a.checkForAllUses(key.fn.Param(key.param), func(state useState) useAction {
			user := state.user
			switch user.InstructionOpcode() {
			case llvm.Load:
				if user.Alignment() > alignment {
					alignment = user.Alignment()
				}
			case llvm.Store:
				if state.operand == 1 && user.Alignment() > alignment {
					alignment = user.Alignment()
				}
			case llvm.Call, llvm.Invoke:
				if callee, ok := definedCallParameter(user, state.operand); ok && a.noCapture[callee] {
					worklist = append(worklist, callee)
				}
			case llvm.GetElementPtr:
				if state.operand == 0 {
					return useFollow
				}
			case llvm.BitCast, llvm.PHI, opcodeAddrSpaceCast:
				return useFollow
			case llvm.Select:
				if state.operand == 1 || state.operand == 2 {
					return useFollow
				}
			case opcodeAtomicCmpXchg, opcodeAtomicRMW:
				if state.operand == 0 && user.Alignment() > alignment {
					alignment = user.Alignment()
				}
			}
			return useSafe
		})
	}
	return alignment
}

func (a *analyzer) allocationUses(root llvm.Value) walkResult {
	result := walkResult{}
	result.escaped = !a.checkForAllUses(root, func(state useState) useAction {
		user := state.user
		switch user.InstructionOpcode() {
		case llvm.Load:
			if user.Alignment() > result.alignment {
				result.alignment = user.Alignment()
			}
			return useSafe
		case llvm.Store:
			if state.operand == 1 {
				if user.Alignment() > result.alignment {
					result.alignment = user.Alignment()
				}
				return useSafe
			}
		case llvm.Call, llvm.Invoke:
			if a.callArgumentNoCapture(state) {
				if key, ok := definedCallParameter(user, state.operand); ok {
					if alignment := a.requiredAlignment(key); alignment > result.alignment {
						result.alignment = alignment
					}
				}
				return useSafe
			}
		case llvm.GetElementPtr:
			if state.operand == 0 {
				return useFollow
			}
		case llvm.BitCast, llvm.PHI:
			return useFollow
		case llvm.Select:
			if state.operand == 1 || state.operand == 2 {
				return useFollow
			}
		}
		return useCapture
	})
	return result
}

func (a *analyzer) callArgumentNoCapture(state useState) bool {
	call := state.user
	calleeOperand := callCalleeOperand(call)
	if state.operand == calleeOperand {
		return true
	}

	callee := call.CalledValue()
	if !callee.IsAInlineAsm().IsNil() {
		return false
	}
	fn := callee.IsAFunction()
	if fn.IsNil() {
		return false
	}
	if fn.IntrinsicID() != 0 {
		return intrinsicArgumentNoCapture(call, fn, state.operand)
	}
	key, ok := definedCallParameter(call, state.operand)
	if !ok {
		return false
	}

	return a.noCapture[key]
}

func (a *analyzer) recordLocations() {
	for _, key := range a.noCaptureKeys {
		a.recordParameterLocations(key)
	}
}

func (a *analyzer) recordParameterLocations(root paramKey) {
	worklist := make([]useState, 0, 8)
	a.addUses(&worklist, root.fn.Param(root.param))
	visited := make(map[useState]struct{})

	for len(worklist) != 0 {
		state := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		if _, ok := visited[state]; ok {
			continue
		}
		visited[state] = struct{}{}
		from := state.user.Operand(state.operand)

		if state.user.InstructionOpcode() == llvm.Store && state.operand == 0 {
			copies, ok := a.copies.getPotentialCopiesOfStoredValue(state.user)
			if ok {
				for _, copy := range copies {
					a.locations.addFlow(from, copy)
					a.addUses(&worklist, copy)
				}
				continue
			}
		}

		user := state.user
		switch user.InstructionOpcode() {
		case llvm.Load:
			if state.operand == 0 && isPointer(user.Type()) {
				a.locations.addDeref(from, user)
				a.addUses(&worklist, user)
			}
		case llvm.Store:
			if state.operand == 0 {
				a.locations.addHeap(from)
			} else if state.operand == 1 {
				a.locations.addMutator(from)
			}
		case llvm.Call, llvm.Invoke:
			calleeOperand := callCalleeOperand(user)
			if state.operand == calleeOperand {
				a.locations.addCallee(from)
				continue
			}
			if callee, ok := definedCallParameter(user, state.operand); ok {
				a.locations.addFlow(from, callee.fn.Param(callee.param))
				if isPointer(user.Type()) {
					a.locations.addCallResult(callee.fn, 0, user)
					a.addUses(&worklist, user)
				}
				continue
			}
			fn := user.CalledValue().IsAFunction()
			if fn.IsNil() || fn.IntrinsicID() == 0 || !intrinsicArgumentNoCapture(user, fn, state.operand) {
				a.locations.addHeap(from)
			}
		case llvm.Ret:
			a.locations.addResult(from, root.fn, 0)
		case llvm.GetElementPtr:
			if state.operand == 0 && user.Type().TypeKind() != llvm.VectorTypeKind {
				a.locations.addFlow(from, user)
				a.addUses(&worklist, user)
			} else {
				a.locations.addHeap(from)
			}
		case llvm.BitCast, llvm.PHI, opcodeAddrSpaceCast:
			a.locations.addFlow(from, user)
			a.addUses(&worklist, user)
		case llvm.Select:
			if state.operand == 1 || state.operand == 2 {
				a.locations.addFlow(from, user)
				a.addUses(&worklist, user)
			} else {
				a.locations.addHeap(from)
			}
		case opcodeAtomicRMW, opcodeAtomicCmpXchg:
			if state.operand == 0 {
				a.locations.addMutator(from)
			} else {
				a.locations.addHeap(from)
			}
		case llvm.VAArg:
		default:
			a.locations.addHeap(from)
		}
	}
}

// Result contains diagnostic facts computed independently from heap-to-stack
// decisions.
type Result struct {
	Parameters []ParameterSummary
}

// ParameterSummary contains the Go-compatible leak summary for one LLVM formal
// parameter.
type ParameterSummary struct {
	Function     string
	Parameter    int
	HeapLevel    int
	MutatorLevel int
	CalleeLevel  int
	Results      []ResultLeak
}

// ResultLeak describes a parameter flow to one direct LLVM result.
type ResultLeak struct {
	Result int
	Level  int
}

func (a *analyzer) result() Result {
	if a.locations == nil {
		return Result{}
	}
	result := Result{Parameters: make([]ParameterSummary, 0, len(a.noCaptureKeys))}
	for _, key := range a.noCaptureKeys {
		leaks := a.locations.summarize(key)
		summary := ParameterSummary{
			Function:     key.fn.Name(),
			Parameter:    key.param,
			HeapLevel:    leaks.get(leakHeap),
			MutatorLevel: leaks.get(leakMutator),
			CalleeLevel:  leaks.get(leakCallee),
		}
		for resultIndex := 0; resultIndex < numEscResults; resultIndex++ {
			if level := leaks.get(leakResult0 + resultIndex); level >= 0 {
				summary.Results = append(summary.Results, ResultLeak{Result: resultIndex, Level: level})
			}
		}
		result.Parameters = append(result.Parameters, summary)
	}
	sort.Slice(result.Parameters, func(i, j int) bool {
		left, right := result.Parameters[i], result.Parameters[j]
		if left.Function != right.Function {
			return left.Function < right.Function
		}
		return left.Parameter < right.Parameter
	})
	return result
}

func intrinsicArgumentNoCapture(call, fn llvm.Value, argument int) bool {
	name := fn.Name()
	if argument < 0 {
		return false
	}
	if strings.HasPrefix(name, "llvm.dbg.") ||
		strings.HasPrefix(name, "llvm.lifetime.start.") ||
		strings.HasPrefix(name, "llvm.lifetime.end.") ||
		strings.HasPrefix(name, "llvm.objectsize.") {
		return true
	}
	kind := llvm.AttributeKindID("nocapture")
	attributeIndex := argument + 1
	if attr := call.GetCallSiteEnumAttribute(attributeIndex, kind); !attr.IsNil() {
		return true
	}
	return !fn.GetEnumAttributeAtIndex(attributeIndex, kind).IsNil()
}

type allocationPlan struct {
	call      llvm.Value
	size      llvm.Value
	zero      bool
	alignment int
}

// TransformModule analyzes eligible LLGo allocations and rewrites proven-local
// AllocZ and AllocU calls. Diagnostic summaries are computed only when
// diagnostics is true.
func TransformModule(mod llvm.Module, diagnostics bool) Result {
	a := newAnalyzer(mod, diagnostics)
	defer a.copies.dispose()
	a.solveNoCapture()
	if a.locations != nil {
		a.recordLocations()
	}
	result := a.result()

	var plans []allocationPlan
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for instr := block.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				plan, ok := a.planAllocation(instr)
				if ok {
					plans = append(plans, plan)
				}
			}
		}
	}

	for _, plan := range plans {
		rewriteAllocation(mod.Context(), plan)
	}
	return result
}

func (a *analyzer) planAllocation(instr llvm.Value) (allocationPlan, bool) {
	call := instr.IsACallInst()
	if call.IsNil() || call.OperandsCount() < 2 || !isPointer(call.Type()) {
		return allocationPlan{}, false
	}
	callee := call.CalledValue().IsAFunction()
	if callee.IsNil() {
		return allocationPlan{}, false
	}
	name := callee.Name()
	if name != runtimeAllocZ && name != runtimeAllocU {
		return allocationPlan{}, false
	}
	size := call.Operand(0).IsAConstantInt()
	if size.IsNil() {
		return allocationPlan{}, false
	}
	sizeBytes := size.ZExtValue()
	if sizeBytes == 0 || sizeBytes > llabi.MaxImplicitStackVarSize || blockInCycle(call.InstructionParent()) {
		return allocationPlan{}, false
	}

	result := a.allocationUses(call)
	if result.escaped {
		return allocationPlan{}, false
	}
	alignment := callResultAlignment(call, callee)
	if result.alignment > alignment {
		alignment = result.alignment
	}
	if alignment == 0 {
		alignment = 1
	}
	if alignment&(alignment-1) != 0 || alignment > int(llabi.MaxImplicitStackVarSize) {
		return allocationPlan{}, false
	}
	return allocationPlan{call: call, size: size, zero: name == runtimeAllocZ, alignment: alignment}, true
}

func callResultAlignment(call, callee llvm.Value) int {
	kind := llvm.AttributeKindID("align")
	align := 0
	if attr := call.GetCallSiteEnumAttribute(0, kind); !attr.IsNil() {
		align = int(attr.GetEnumValue())
	}
	if attr := callee.GetEnumAttributeAtIndex(0, kind); !attr.IsNil() && int(attr.GetEnumValue()) > align {
		align = int(attr.GetEnumValue())
	}
	return align
}

func blockInCycle(start llvm.BasicBlock) bool {
	visited := make(map[llvm.BasicBlock]bool)
	var visit func(llvm.BasicBlock) bool
	visit = func(block llvm.BasicBlock) bool {
		if block == start {
			return true
		}
		if visited[block] {
			return false
		}
		visited[block] = true
		terminator := block.LastInstruction()
		for index := 0; index < terminator.SuccessorsCount(); index++ {
			if visit(terminator.Successor(index)) {
				return true
			}
		}
		return false
	}

	terminator := start.LastInstruction()
	for index := 0; index < terminator.SuccessorsCount(); index++ {
		if visit(terminator.Successor(index)) {
			return true
		}
	}
	return false
}

func rewriteAllocation(ctx llvm.Context, plan allocationPlan) bool {
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointBefore(plan.call)
	stack := llvm.CreateArrayAlloca(builder, ctx.Int8Type(), plan.size)
	if stack.Type().PointerAddressSpace() != plan.call.Type().PointerAddressSpace() {
		stack.EraseFromParentAsInstruction()
		return false
	}
	stack.SetName(plan.call.Name() + ".stack")
	stack.SetAlignment(plan.alignment)
	if plan.zero {
		builder.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
			stack,
			llvm.ConstInt(ctx.Int8Type(), 0, false),
			plan.size,
			llvm.ConstInt(ctx.Int1Type(), 0, false),
		}, "")
	}
	plan.call.ReplaceAllUsesWith(stack)
	plan.call.EraseFromParentAsInstruction()
	return true
}

// Package reflectnames recovers bounded MethodByName name sets from optimized
// LLVM IR. Unknown memory, aliases, and IR shapes retain generic reflection.
package reflectnames

import (
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

const maxChoices = 32
const maxDepth = 24

// Resolve is the closed-module variant of ResolveWithGlobals. Mutable globals
// with non-local linkage are not assumed to have no writers in other modules.
func Resolve(mod llvm.Module, ptr, length llvm.Value) ([]string, bool) {
	return ResolveWithGlobals(mod, ptr, length, nil)
}

// ResolveWithGlobals returns a conservative finite set of string values.
// canReadExternal must certify absence of unseen users and interposition for
// a non-local global; local writes and pointer escapes are still checked here.
// In particular, a global initializer is not itself a proof of immutability.
func ResolveWithGlobals(mod llvm.Module, ptr, length llvm.Value, canReadExternal func(llvm.Value) bool) ([]string, bool) {
	r := newResolver(mod, canReadExternal)
	defer r.layout.Dispose()
	return r.result(r.pair(ptr, length, 0))
}

// ResolveStringValueWithGlobals accepts the Go string aggregate representation.
func ResolveStringValueWithGlobals(mod llvm.Module, value llvm.Value, canReadExternal func(llvm.Value) bool) ([]string, bool) {
	r := newResolver(mod, canReadExternal)
	defer r.layout.Dispose()
	return r.result(r.aggregate(value, 0))
}

// ResolveCall uses the frontend/C-ABI-preserved string argument marker. It
// accepts both an aggregate Go string and its split pointer/length form.
func ResolveCall(mod llvm.Module, call llvm.Value, canReadExternal func(llvm.Value) bool) ([]string, bool) {
	if opcode(call) != llvm.Call && opcode(call) != llvm.Invoke {
		return nil, false
	}
	index := -1
	for i := 0; i < call.ArgOperandsCount(); i++ {
		if !call.GetCallSiteStringAttribute(i+1, "llgo.reflect.methodbyname.name").IsNil() {
			if index != -1 {
				return nil, false
			}
			index = i
		}
	}
	if index < 0 {
		return nil, false
	}
	arg := call.Operand(index)
	if arg.Type().TypeKind() == llvm.StructTypeKind {
		return ResolveStringValueWithGlobals(mod, arg, canReadExternal)
	}
	if index+1 == call.ArgOperandsCount() {
		return nil, false
	}
	return ResolveWithGlobals(mod, arg, call.Operand(index+1), canReadExternal)
}

type resolver struct {
	layout   llvm.TargetData
	external func(llvm.Value) bool
	names    map[string]bool
	readonly map[llvm.Value]bool
	steps    int
}

func newResolver(mod llvm.Module, external func(llvm.Value) bool) *resolver {
	return &resolver{layout: llvm.NewTargetData(mod.DataLayout()), external: external,
		names: make(map[string]bool), readonly: make(map[llvm.Value]bool)}
}

func (r *resolver) result(ok bool) ([]string, bool) {
	if !ok || len(r.names) == 0 {
		return nil, false
	}
	names := make([]string, 0, len(r.names))
	for name := range r.names {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, true
}

func (r *resolver) step(depth int) bool {
	r.steps++
	return depth <= maxDepth && r.steps <= 4096
}

func (r *resolver) pair(ptr, length llvm.Value, depth int) bool {
	if !r.step(depth) || ptr.IsNil() || length.IsNil() || ptr.Type().TypeKind() != llvm.PointerTypeKind || ptr.Type().PointerAddressSpace() != 0 || length.Type().TypeKind() != llvm.IntegerTypeKind {
		return false
	}
	if n, ok := integer(length); ok {
		locations, ok := r.pointers(ptr, depth+1)
		if !ok {
			return false
		}
		for _, location := range locations {
			if !r.canRead(location.global) {
				return false
			}
			init := location.global.Initializer()
			if !init.IsConstantString() {
				return false
			}
			data := init.ConstGetAsString()
			if location.offset > uint64(len(data)) || n > uint64(len(data))-location.offset {
				return false
			}
			r.names[data[location.offset:location.offset+n]] = true
			if len(r.names) > maxChoices {
				return false
			}
		}
		return len(locations) != 0
	}
	if opcode(ptr) == llvm.Select && opcode(length) == llvm.Select && ptr.Operand(0) == length.Operand(0) {
		return r.pair(ptr.Operand(1), length.Operand(1), depth+1) && r.pair(ptr.Operand(2), length.Operand(2), depth+1)
	}
	if opcode(ptr) == llvm.PHI && opcode(length) == llvm.PHI && ptr.InstructionParent() == length.InstructionParent() && ptr.IncomingCount() == length.IncomingCount() {
		for i := 0; i < ptr.IncomingCount(); i++ {
			j := 0
			for ; j < length.IncomingCount() && ptr.IncomingBlock(i) != length.IncomingBlock(j); j++ {
			}
			if j == length.IncomingCount() || !r.pair(ptr.IncomingValue(i), length.IncomingValue(j), depth+1) {
				return false
			}
		}
		return ptr.IncomingCount() != 0
	}
	if opcode(ptr) == llvm.ExtractValue && opcode(length) == llvm.ExtractValue && ptr.Operand(0) == length.Operand(0) && singleIndex(ptr, 0) && singleIndex(length, 1) {
		return r.aggregate(ptr.Operand(0), depth+1)
	}
	if simpleLoad(ptr) && simpleLoad(length) {
		base, offset, ok := r.constantOffset(length.Operand(0), depth+1)
		pbase, poffset, pok := r.constantOffset(ptr.Operand(0), depth+1)
		if !ok || !pok || base != pbase || offset != poffset+uint64(r.layout.PointerSize()) {
			return false
		}
		locations, ok := r.pointers(ptr.Operand(0), depth+1)
		if !ok {
			return false
		}
		for _, loc := range locations {
			p, ok := r.read(loc, ptr.Type())
			if !ok {
				return false
			}
			loc.offset += uint64(r.layout.PointerSize())
			n, ok := r.read(loc, length.Type())
			if !ok || !r.pair(p, n, depth+1) {
				return false
			}
		}
		return len(locations) != 0
	}
	return false
}

func (r *resolver) aggregate(value llvm.Value, depth int) bool {
	if !r.step(depth) || value.IsNil() || value.Type().TypeKind() != llvm.StructTypeKind {
		return false
	}
	fields := value.Type().StructElementTypes()
	if len(fields) != 2 || fields[0].TypeKind() != llvm.PointerTypeKind || fields[1].TypeKind() != llvm.IntegerTypeKind {
		return false
	}
	if !value.IsAConstantStruct().IsNil() {
		return r.pair(value.Operand(0), value.Operand(1), depth+1)
	}
	if opcode(value) == llvm.InsertValue {
		var p, n llvm.Value
		cur := value
		for i := 0; i < maxDepth && opcode(cur) == llvm.InsertValue; i++ {
			if singleIndex(cur, 0) && p.IsNil() {
				p = cur.Operand(1)
			}
			if singleIndex(cur, 1) && n.IsNil() {
				n = cur.Operand(1)
			}
			cur = cur.Operand(0)
		}
		if !cur.IsAConstantStruct().IsNil() {
			if p.IsNil() {
				p = cur.Operand(0)
			}
			if n.IsNil() {
				n = cur.Operand(1)
			}
		}
		return !p.IsNil() && !n.IsNil() && r.pair(p, n, depth+1)
	}
	if simpleLoad(value) {
		locations, ok := r.pointers(value.Operand(0), depth+1)
		if !ok {
			return false
		}
		for _, loc := range locations {
			v, ok := r.read(loc, value.Type())
			if !ok || !r.aggregate(v, depth+1) {
				return false
			}
		}
		return len(locations) != 0
	}
	if opcode(value) == llvm.Select {
		return r.aggregate(value.Operand(1), depth+1) && r.aggregate(value.Operand(2), depth+1)
	}
	if opcode(value) == llvm.PHI {
		for i := 0; i < value.IncomingCount(); i++ {
			if !r.aggregate(value.IncomingValue(i), depth+1) {
				return false
			}
		}
		return value.IncomingCount() != 0
	}
	return false
}

type location struct {
	global llvm.Value
	offset uint64
}

func (r *resolver) pointers(v llvm.Value, depth int) ([]location, bool) {
	if !r.step(depth) {
		return nil, false
	}
	if !v.IsAGlobalVariable().IsNil() {
		return []location{{global: v}}, true
	}
	switch opcode(v) {
	case llvm.BitCast:
		return r.pointers(v.Operand(0), depth+1)
	case llvm.Load:
		if !simpleLoad(v) {
			return nil, false
		}
		from, ok := r.pointers(v.Operand(0), depth+1)
		if !ok {
			return nil, false
		}
		var result []location
		for _, loc := range from {
			p, ok := r.read(loc, v.Type())
			if !ok {
				return nil, false
			}
			more, ok := r.pointers(p, depth+1)
			if !ok {
				return nil, false
			}
			result = append(result, more...)
			if len(result) > maxChoices {
				return nil, false
			}
		}
		return result, len(result) > 0
	case llvm.GetElementPtr:
		bases, ok := r.pointers(v.Operand(0), depth+1)
		if !ok {
			return nil, false
		}
		if delta, ok := r.gepOffset(v); ok {
			for i := range bases {
				if bases[i].offset > ^uint64(0)-delta {
					return nil, false
				}
				bases[i].offset += delta
			}
			return bases, true
		}
		// Only a complete top-level array is enumerated. LLVM inbounds plus
		// the subsequent paired loads restrict defined executions to that
		// allocation; an arbitrary subarray or dynamic byte offset is rejected.
		if v.OperandsCount() != 2 || !inBoundsGEP(v) {
			return nil, false
		}
		var result []location
		for _, base := range bases {
			typ := base.global.GlobalValueType()
			if base.offset != 0 || typ.TypeKind() != llvm.ArrayTypeKind || typ.ElementType() != v.GEPSourceElementType() || typ.ArrayLength() > maxChoices {
				return nil, false
			}
			stride := r.layout.TypeAllocSize(typ.ElementType())
			if stride == 0 {
				return nil, false
			}
			for i := 0; i < typ.ArrayLength(); i++ {
				result = append(result, location{base.global, uint64(i) * stride})
			}
			if len(result) > maxChoices {
				return nil, false
			}
		}
		return result, len(result) > 0
	}
	return nil, false
}

func (r *resolver) gepOffset(v llvm.Value) (uint64, bool) {
	typ := v.GEPSourceElementType()
	var offset uint64
	for i := 1; i < v.OperandsCount(); i++ {
		index, ok := integer(v.Operand(i))
		if !ok {
			return 0, false
		}
		var delta uint64
		if i == 1 {
			stride := r.layout.TypeAllocSize(typ)
			if stride != 0 && index > ^uint64(0)/stride {
				return 0, false
			}
			delta = index * stride
		} else {
			switch typ.TypeKind() {
			case llvm.StructTypeKind:
				fields := typ.StructElementTypes()
				if index >= uint64(len(fields)) {
					return 0, false
				}
				delta = r.layout.ElementOffset(typ, int(index))
				typ = fields[index]
			case llvm.ArrayTypeKind:
				typ = typ.ElementType()
				stride := r.layout.TypeAllocSize(typ)
				if stride != 0 && index > ^uint64(0)/stride {
					return 0, false
				}
				delta = index * stride
			default:
				return 0, false
			}
		}
		if offset > ^uint64(0)-delta {
			return 0, false
		}
		offset += delta
	}
	return offset, true
}

func (r *resolver) constantOffset(v llvm.Value, depth int) (llvm.Value, uint64, bool) {
	if !r.step(depth) {
		return llvm.Value{}, 0, false
	}
	if opcode(v) == llvm.GetElementPtr {
		if delta, ok := r.gepOffset(v); ok {
			base, offset, ok := r.constantOffset(v.Operand(0), depth+1)
			if !ok || offset > ^uint64(0)-delta {
				return llvm.Value{}, 0, false
			}
			return base, offset + delta, true
		}
	}
	if opcode(v) == llvm.BitCast {
		return r.constantOffset(v.Operand(0), depth+1)
	}
	return v, 0, true
}

func (r *resolver) read(loc location, typ llvm.Type) (llvm.Value, bool) {
	if !r.canRead(loc.global) {
		return llvm.Value{}, false
	}
	return r.at(loc.global.Initializer(), loc.offset, typ, 0)
}

func (r *resolver) at(init llvm.Value, offset uint64, want llvm.Type, depth int) (llvm.Value, bool) {
	if !r.step(depth) || init.IsNil() {
		return llvm.Value{}, false
	}
	typ := init.Type()
	if offset == 0 && typ == want {
		return init, true
	}
	switch typ.TypeKind() {
	case llvm.StructTypeKind:
		for i, field := range typ.StructElementTypes() {
			start := r.layout.ElementOffset(typ, i)
			size := r.layout.TypeAllocSize(field)
			if offset >= start && offset-start < size && i < init.OperandsCount() {
				return r.at(init.Operand(i), offset-start, want, depth+1)
			}
		}
	case llvm.ArrayTypeKind:
		stride := r.layout.TypeAllocSize(typ.ElementType())
		if stride == 0 {
			return llvm.Value{}, false
		}
		i := offset / stride
		if i < uint64(typ.ArrayLength()) && i < uint64(init.OperandsCount()) {
			return r.at(init.Operand(int(i)), offset%stride, want, depth+1)
		}
	}
	return llvm.Value{}, false
}

func (r *resolver) closedGlobal(g llvm.Value) bool {
	return g.Linkage() == llvm.InternalLinkage || g.Linkage() == llvm.PrivateLinkage || r.external != nil && r.external(g)
}

func (r *resolver) canRead(g llvm.Value) bool {
	if g.IsNil() || g.IsAGlobalVariable().IsNil() || g.Initializer().IsNil() {
		return false
	}
	if cached, ok := r.readonly[g]; ok {
		return cached
	}
	r.readonly[g] = false
	if !r.closedGlobal(g) {
		return false
	}
	// An externally_initialized definition explicitly permits initialization
	// outside this IR. The bindings do not expose its flag; inspect only the
	// global's modifiers before the global/constant keyword.
	declaration, ok := definitionRHS(g.String())
	if !ok {
		return false
	}
	for _, token := range strings.Fields(declaration) {
		if token == "global" || token == "constant" {
			break
		}
		if token == "externally_initialized" {
			return false
		}
	}
	if g.IsGlobalConstant() {
		r.readonly[g] = true
		return true
	}
	ok = r.readUses(g, 0, make(map[readUse]bool), 0)
	r.readonly[g] = ok
	return ok
}

type readUse struct {
	value llvm.Value
	loads int
}

// loads counts initializer indirections to the original global. A header that
// contains its address requires following one pointer load; an address holder
// pointing to that header requires following two. Aggregate constants alone do
// not add a memory indirection.
func (r *resolver) readUses(value llvm.Value, loads int, seen map[readUse]bool, depth int) bool {
	key := readUse{value, loads}
	if seen[key] {
		return true
	}
	seen[key] = true
	if !r.step(depth) {
		return false
	}
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		u := use.User()
		if simpleLoad(u) && u.Operand(0) == value {
			if loads > 0 {
				if u.Type().TypeKind() == llvm.PointerTypeKind {
					if !r.readUses(u, loads-1, seen, depth+1) {
						return false
					}
				} else if u.Type().TypeKind() != llvm.IntegerTypeKind || !r.readIntegerUses(u, make(map[llvm.Value]bool), depth+1) {
					return false
				}
			}
			continue
		}
		switch opcode(u) {
		case llvm.GetElementPtr, llvm.BitCast:
			if u.Operand(0) != value || !r.readUses(u, loads, seen, depth+1) {
				return false
			}
			continue
		}
		if !u.IsAConstantStruct().IsNil() || !u.IsAConstantArray().IsNil() {
			if !r.readUses(u, loads, seen, depth+1) {
				return false
			}
			continue
		}
		if !u.IsAGlobalVariable().IsNil() && r.closedGlobal(u) && u.Initializer() == value {
			if !r.readUses(u, loads+1, seen, depth+1) {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// An alias-bearing header may also load integer length/capacity fields. Permit
// control-flow-only integer computation, but reject conversion to a pointer or
// escape: a type-punned integer load can contain the pointer's address bits.
func (r *resolver) readIntegerUses(value llvm.Value, seen map[llvm.Value]bool, depth int) bool {
	if seen[value] {
		return true
	}
	seen[value] = true
	if !r.step(depth) {
		return false
	}
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		u := use.User()
		switch opcode(u) {
		case llvm.Br, llvm.Switch:
			continue
		case llvm.Add, llvm.Sub, llvm.Mul, llvm.UDiv, llvm.SDiv, llvm.URem, llvm.SRem,
			llvm.Shl, llvm.LShr, llvm.AShr, llvm.And, llvm.Or, llvm.Xor,
			llvm.Trunc, llvm.ZExt, llvm.SExt, llvm.ICmp, llvm.PHI, llvm.Select:
			if u.Type().TypeKind() == llvm.IntegerTypeKind && r.readIntegerUses(u, seen, depth+1) {
				continue
			}
		}
		return false
	}
	return true
}

func opcode(v llvm.Value) llvm.Opcode {
	if v.IsNil() {
		return 0
	}
	if !v.IsAInstruction().IsNil() {
		return v.InstructionOpcode()
	}
	if !v.IsAConstantExpr().IsNil() {
		return v.Opcode()
	}
	return 0
}

func integer(v llvm.Value) (uint64, bool) {
	if v.IsNil() || v.IsAConstantInt().IsNil() || v.Type().IntTypeWidth() > 64 || v.SExtValue() < 0 {
		return 0, false
	}
	return v.ZExtValue(), true
}

func simpleLoad(v llvm.Value) bool {
	return opcode(v) == llvm.Load && !v.IsVolatile() && v.Ordering() == llvm.AtomicOrderingNotAtomic
}
func singleIndex(v llvm.Value, index uint32) bool {
	indices := v.Indices()
	return len(indices) == 1 && indices[0] == index
}

func inBoundsGEP(v llvm.Value) bool {
	// The current Go bindings do not expose LLVMIsInBounds. Inspect only the
	// opcode's printed modifier, never an arbitrary occurrence in a symbol.
	text := strings.TrimSpace(v.String())
	if !v.IsAInstruction().IsNil() {
		var ok bool
		if text, ok = definitionRHS(text); !ok {
			return false
		}
	} else {
		text = strings.TrimPrefix(text, "ptr ")
	}
	return strings.HasPrefix(text, "getelementptr inbounds ")
}

// LLVM permits quoted names containing spaces, equals signs and opcode text.
// Only an assignment separator outside a quoted name starts the definition.
func definitionRHS(text string) (string, bool) {
	quoted := false
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\\':
			if quoted {
				i++
			}
		case '"':
			quoted = !quoted
		case ' ':
			if !quoted && strings.HasPrefix(text[i:], " = ") {
				return strings.TrimSpace(text[i+3:]), true
			}
		}
	}
	return "", false
}

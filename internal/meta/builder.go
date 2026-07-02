package meta

// bitmap is a bit array backed by []uint32; each uint32 holds 32 bits.
// Indexed by LocalSymbol: bit i lives in [i/32] at position i%32.
// The slice length is always kept in sync with the symbol table by grow(),
// so set() and has() never need to check bounds.
type bitmap []uint32

// grow ensures the bitmap can hold bit index i.
// Called whenever a new symbol is registered.
func (bm *bitmap) grow(i LocalSymbol) {
	need := uint(i)/32 + 1
	for uint(len(*bm)) < need {
		*bm = append(*bm, 0)
	}
}

// set sets bit i. grow(i) must have been called beforehand.
func (bm bitmap) set(i LocalSymbol) {
	bm[uint(i)/32] |= 1 << (uint(i) % 32)
}

// has reports whether bit i is set.
func (bm bitmap) has(i LocalSymbol) bool {
	return bm[uint(i)/32]&(1<<(uint(i)%32)) != 0
}

// Builder accumulates per-package metadata facts and serializes them into
// the binary wire format understood by PackageMeta.
//
// Typical usage:
//
//	b := NewBuilder()
//	fn := b.DefSym("main.main")
//	callee := b.RefSym("fmt.Println")
//	b.AddEdge(fn, callee, EdgeOrdinary, 0)
//	pm, err := b.Build()
type Builder struct {
	// string interning
	strData []byte            // raw byte stream, all strings concatenated
	strMap  map[string]uint32 // string → offset in strData

	// symbol table
	symNames []symEntry             // indexed by LocalSymbol
	symMap   map[string]LocalSymbol // name → LocalSymbol

	// per-symbol edge lists (source LocalSymbol → edges)
	edges [][]bEdge

	// per-symbol TypeChildren lists
	typeChildren    [][]LocalSymbol
	typeChildrenSet map[[2]LocalSymbol]struct{} // dedup (parent, child) pairs

	// per-symbol MethodInfo (only concrete types)
	methodInfo [][]bMethodSlot

	// per-symbol InterfaceInfo (only interface types)
	ifaceInfo [][]bMethodSig

	// reflect bitmap: bit i set means symbol i triggers conservative reflection
	reflectBits bitmap
}

type symEntry struct {
	nameOff uint32
	nameLen uint32
}

type bEdge struct {
	target uint32 // LocalSymbol or stringTable offset (UseNamedMethod)
	extra  uint32
	kind   uint8
}

type bMethodSlot struct {
	name  NameRef // method short name
	mtype uint32  // LocalSymbol
	ifn   uint32  // LocalSymbol
	tfn   uint32  // LocalSymbol
}

type bMethodSig struct {
	name  NameRef // method short name
	mtype uint32  // LocalSymbol
}

// NewBuilder creates an empty Builder.
func NewBuilder() *Builder {
	return &Builder{
		strMap:          make(map[string]uint32),
		symMap:          make(map[string]LocalSymbol),
		typeChildrenSet: make(map[[2]LocalSymbol]struct{}),
	}
}

// internStr adds s to the string byte stream (idempotent) and returns its offset.
func (b *Builder) internStr(s string) uint32 {
	if off, ok := b.strMap[s]; ok {
		return off
	}
	off := uint32(len(b.strData))
	b.strData = append(b.strData, s...)
	b.strMap[s] = off
	return off
}

// internName registers a name string and returns a NameRef.
func (b *Builder) internName(s string) NameRef {
	return NameRef{Off: b.internStr(s), Len: uint32(len(s))}
}

// Sym registers a symbol by name and returns its LocalSymbol.
// Calling Sym with the same name twice returns the same LocalSymbol.
// Whether the symbol is defined in this package or referenced from another
// makes no difference to the metadata format.
func (b *Builder) Sym(name string) LocalSymbol {
	return b.sym(name)
}

func (b *Builder) sym(name string) LocalSymbol {
	if id, ok := b.symMap[name]; ok {
		return id
	}
	id := LocalSymbol(len(b.symNames))
	off := b.internStr(name)
	b.symNames = append(b.symNames, symEntry{nameOff: off, nameLen: uint32(len(name))})
	b.symMap[name] = id
	// grow all per-symbol structures in sync with the symbol table
	b.edges = append(b.edges, nil)
	b.typeChildren = append(b.typeChildren, nil)
	b.methodInfo = append(b.methodInfo, nil)
	b.ifaceInfo = append(b.ifaceInfo, nil)
	b.reflectBits.grow(id) // ensure bit slot exists before any MarkReflect call
	return id
}

// AddEdge records a directed edge from src to dst with the given kind and extra.
//
//   - EdgeOrdinary:       dst is a LocalSymbol; extra = 0
//   - EdgeUseIface:       dst is a LocalSymbol (type); extra = 0
//   - EdgeUseIfaceMethod: dst is a LocalSymbol (interface); extra = method index
//   - EdgeUseNamedMethod: dst is a string (method name); extra = 0
func (b *Builder) AddEdge(src, dst LocalSymbol, kind uint8, extra uint32) {
	b.edges[src] = append(b.edges[src], bEdge{
		target: uint32(dst),
		extra:  extra,
		kind:   kind,
	})
}

// AddNamedMethodEdge records an EdgeUseNamedMethod edge where the target is a
// method name string rather than a LocalSymbol. The name's byte offset is stored
// in target and its length in extra, together forming a NameRef.
func (b *Builder) AddNamedMethodEdge(src LocalSymbol, methodName string) {
	ref := b.internName(methodName)
	b.edges[src] = append(b.edges[src], bEdge{
		target: ref.Off,
		extra:  ref.Len,
		kind:   EdgeUseNamedMethod,
	})
}

// AddTypeChild records that parent type structurally contains child type.
// Idempotent: duplicate (parent, child) pairs are silently ignored.
func (b *Builder) AddTypeChild(parent, child LocalSymbol) {
	key := [2]LocalSymbol{parent, child}
	if _, ok := b.typeChildrenSet[key]; ok {
		return
	}
	b.typeChildrenSet[key] = struct{}{}
	b.typeChildren[parent] = append(b.typeChildren[parent], child)
}

// AddMethodSlot records one ABI method slot for a concrete type.
// Slots must be appended in abi.Method table order.
func (b *Builder) AddMethodSlot(typ LocalSymbol, methodName string, mtype, ifn, tfn LocalSymbol) {
	b.methodInfo[typ] = append(b.methodInfo[typ], bMethodSlot{
		name:  b.internName(methodName),
		mtype: uint32(mtype),
		ifn:   uint32(ifn),
		tfn:   uint32(tfn),
	})
}

// AddIfaceMethod records one method in an interface's method set.
// Idempotent: if the same (name, mtype) pair is already registered for iface,
// this call is a no-op — the Builder deduplicates internally.
func (b *Builder) AddIfaceMethod(iface LocalSymbol, methodName string, mtype LocalSymbol) {
	ref := b.internName(methodName)
	mt := uint32(mtype)
	for _, s := range b.ifaceInfo[iface] {
		if s.name == ref && s.mtype == mt {
			return
		}
	}
	b.ifaceInfo[iface] = append(b.ifaceInfo[iface], bMethodSig{
		name:  ref,
		mtype: mt,
	})
}

// MarkReflect marks sym as triggering conservative reflection handling.
func (b *Builder) MarkReflect(sym LocalSymbol) {
	b.reflectBits.set(sym)
}

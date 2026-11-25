package meta

import "encoding/binary"

// Builder accumulates per-package metadata facts and serializes them into
// the binary wire format understood by PackageMeta.
//
// Typical usage:
//
//	b := NewBuilder()
//	fn := b.Sym("main.main")
//	callee := b.Sym("fmt.Println")
//	b.AddOrdinaryEdge(fn, callee)
//	pm, err := b.Build()
type Builder struct {
	// string interning
	strData []byte            // raw byte stream, all strings concatenated
	strMap  map[string]uint32 // string → offset in strData

	// symbol table
	symNames []symEntry        // indexed by Symbol
	symMap   map[string]Symbol // name → Symbol

	// per-symbol ordinary edge lists (source Symbol → target package-local Symbols)
	ordinaryEdges [][]Symbol

	// per-symbol function demand lists (source Symbol → demand facts)
	funcDemands [][]bFuncDemand

	// per-symbol TypeChildren lists
	typeChildren    [][]Symbol
	typeChildrenSet map[[2]Symbol]struct{} // dedup (parent, child) pairs

	// per-symbol MethodInfo (only concrete types)
	methodInfo [][]bMethodSlot

	// per-symbol InterfaceInfo (only interface types)
	ifaceInfo [][]bMethodSig
}

type symEntry struct {
	nameOff uint32
	nameLen uint32
}

type bFuncDemand struct {
	kind   DemandKind
	target uint32 // Symbol or stringTable offset (DemandNamedMethod)
	extra  uint32
}

type bMethodSlot struct {
	name  nameRef // method short name
	mtype uint32  // Symbol
	ifn   uint32  // Symbol
	tfn   uint32  // Symbol
}

type bMethodSig struct {
	name  nameRef // method short name
	mtype uint32  // Symbol
}

// NewBuilder creates an empty Builder.
func NewBuilder() *Builder {
	return &Builder{
		strMap:          make(map[string]uint32),
		symMap:          make(map[string]Symbol),
		typeChildrenSet: make(map[[2]Symbol]struct{}),
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

// internName registers a name string and returns a nameRef.
func (b *Builder) internName(s string) nameRef {
	return nameRef{Off: b.internStr(s), Len: uint32(len(s))}
}

// Sym registers a symbol by name and returns its Symbol.
// Calling Sym with the same name twice returns the same Symbol.
// Whether the symbol is defined in this package or referenced from another
// makes no difference to the metadata format.
func (b *Builder) Sym(name string) Symbol {
	return b.sym(name)
}

func (b *Builder) sym(name string) Symbol {
	if id, ok := b.symMap[name]; ok {
		return id
	}
	id := Symbol(len(b.symNames))
	off := b.internStr(name)
	b.symNames = append(b.symNames, symEntry{nameOff: off, nameLen: uint32(len(name))})
	b.symMap[name] = id
	// grow all per-symbol structures in sync with the symbol table
	b.ordinaryEdges = append(b.ordinaryEdges, nil)
	b.funcDemands = append(b.funcDemands, nil)
	b.typeChildren = append(b.typeChildren, nil)
	b.methodInfo = append(b.methodInfo, nil)
	b.ifaceInfo = append(b.ifaceInfo, nil)
	return id
}

// AddOrdinaryEdge records a plain symbol-to-symbol reference from src to dst
// (call, type use, global var reference, etc.).
func (b *Builder) AddOrdinaryEdge(src, dst Symbol) {
	b.ordinaryEdges[src] = append(b.ordinaryEdges[src], dst)
}

// AddIfaceUse records that src converts a value of type typ to an interface.
func (b *Builder) AddIfaceUse(src, typ Symbol) {
	b.funcDemands[src] = append(b.funcDemands[src], bFuncDemand{
		kind:   DemandUseIface,
		target: uint32(typ),
	})
}

// AddIfaceMethodUse records that src calls the methodIndex-th method (in
// declaration order) of interface iface.
func (b *Builder) AddIfaceMethodUse(src, iface Symbol, methodIndex uint32) {
	b.funcDemands[src] = append(b.funcDemands[src], bFuncDemand{
		kind:   DemandIfaceMethod,
		target: uint32(iface),
		extra:  methodIndex,
	})
}

// AddNamedMethodUse records that src does a constant MethodByName(methodName)
// call. The method name is stored as a string-table reference.
func (b *Builder) AddNamedMethodUse(src Symbol, methodName string) {
	ref := b.internName(methodName)
	b.funcDemands[src] = append(b.funcDemands[src], bFuncDemand{
		kind:   DemandNamedMethod,
		target: ref.Off,
		extra:  ref.Len,
	})
}

// AddTypeChild records that parent type structurally contains child type.
// Idempotent: duplicate (parent, child) pairs are silently ignored.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	key := [2]Symbol{parent, child}
	if _, ok := b.typeChildrenSet[key]; ok {
		return
	}
	b.typeChildrenSet[key] = struct{}{}
	b.typeChildren[parent] = append(b.typeChildren[parent], child)
}

// AddMethodSlot records one ABI method slot for a concrete type.
// Slots must be appended in abi.Method table order.
func (b *Builder) AddMethodSlot(typ Symbol, methodName string, mtype, ifn, tfn Symbol) {
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
func (b *Builder) AddIfaceMethod(iface Symbol, methodName string, mtype Symbol) {
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
func (b *Builder) MarkReflect(sym Symbol) {
	b.funcDemands[sym] = append(b.funcDemands[sym], bFuncDemand{
		kind: DemandReflectMethod,
	})
}

// Build serializes all accumulated facts into a PackageMeta.
//
// The process is:
//  1. Calculate the byte size of every section.
//  2. Derive each section's start offset.
//  3. Allocate one []byte for the whole file.
//  4. Write header + every section directly into the buffer — no intermediate
//     allocations, no copies.
func (b *Builder) Build() (*PackageMeta, error) {
	nsyms := uint32(len(b.symNames))

	// ── 1. calculate section sizes ────────────────────────────────────────────

	// stringTable is padded to a 4-byte boundary so every following section
	// starts 4-byte aligned, enabling zero-copy unsafe access (e.g. TypeChildren).
	strSize := align4(uint32(len(b.strData)))

	symSize := 4 + nsyms*12 // nsyms u32 + N×SymbolRecord(12)

	totalOrdinary := uint32(0)
	for _, es := range b.ordinaryEdges {
		totalOrdinary += uint32(len(es))
	}
	ordinarySize := 4 + (nsyms+1)*4 + totalOrdinary*4 // nsyms + offsets[N+1] + N×Symbol(4)

	totalDemands := uint32(0)
	for _, ds := range b.funcDemands {
		totalDemands += uint32(len(ds))
	}
	demandSize := 4 + (nsyms+1)*4 + totalDemands*12 // nsyms + offsets[N+1] + N×FuncDemand(12)

	totalChildren := uint32(0)
	for _, cs := range b.typeChildren {
		totalChildren += uint32(len(cs))
	}
	childSize := 4 + (nsyms+1)*4 + totalChildren*4

	totalSlots := uint32(0)
	for _, ms := range b.methodInfo {
		totalSlots += uint32(len(ms))
	}
	methodSize := 4 + (nsyms+1)*4 + totalSlots*20 // N×localMethodSlot(20: nameRef(8)+mtype+ifn+tfn)

	totalSigs := uint32(0)
	for _, ss := range b.ifaceInfo {
		totalSigs += uint32(len(ss))
	}
	ifaceSize := 4 + (nsyms+1)*4 + totalSigs*12 // N×localMethodSig(12: nameRef(8)+mtype)

	// ── 2. calculate section offsets ─────────────────────────────────────────

	var offsets [numSections]uint32
	cur := uint32(headerSize)
	offsets[secStringTable] = cur
	cur += strSize
	offsets[secSymbols] = cur
	cur += symSize
	offsets[secOrdinaryEdges] = cur
	cur += ordinarySize
	offsets[secFuncDemand] = cur
	cur += demandSize
	offsets[secTypeChildren] = cur
	cur += childSize
	offsets[secMethodInfo] = cur
	cur += methodSize
	offsets[secIfaceInfo] = cur
	cur += ifaceSize

	// ── 3. allocate one buffer ────────────────────────────────────────────────

	raw := make([]byte, cur)

	// ── 4. write header ───────────────────────────────────────────────────────

	copy(raw[0:4], magic)
	binary.LittleEndian.PutUint32(raw[4:8], version)
	for i, off := range offsets {
		binary.LittleEndian.PutUint32(raw[8+i*4:], off)
	}

	// ── 5. write each section directly into raw ───────────────────────────────

	writeStringTable(raw[offsets[secStringTable]:], b)
	writeSymbols(raw[offsets[secSymbols]:], b, nsyms)
	writeOrdinaryEdges(raw[offsets[secOrdinaryEdges]:], b, nsyms)
	writeFuncDemand(raw[offsets[secFuncDemand]:], b, nsyms)
	writeTypeChildren(raw[offsets[secTypeChildren]:], b, nsyms)
	writeMethodInfo(raw[offsets[secMethodInfo]:], b, nsyms)
	writeIfaceInfo(raw[offsets[secIfaceInfo]:], b, nsyms)

	return packageMetaView(raw, offsets, nsyms), nil
}

// ── section writers ───────────────────────────────────────────────────────────
// Each writer receives a slice starting exactly at its section's offset.
// It writes directly into that slice — no allocation, no copy.

func writeStringTable(dst []byte, b *Builder) {
	// dst may be longer than strData (padding); padding bytes stay zero.
	copy(dst, b.strData)
}

// align4 rounds n up to the next multiple of 4.
func align4(n uint32) uint32 {
	return (n + 3) &^ 3
}

// writeSymbols writes:
//
//	nsyms u32
//	[nsyms] { nameOff u32, nameLen u32, _ [4]byte }  (12 bytes each)
func writeSymbols(dst []byte, b *Builder, nsyms uint32) {
	binary.LittleEndian.PutUint32(dst, nsyms)
	const rec = 12
	for i, e := range b.symNames {
		base := 4 + i*rec
		binary.LittleEndian.PutUint32(dst[base:], e.nameOff)
		binary.LittleEndian.PutUint32(dst[base+4:], e.nameLen)
		// dst[base+8 : base+12] reserved, already zero
	}
}

// writeCSRHeader writes:
//
//	nsyms u32
//	offsets [nsyms+1] u32
//
// and returns the slice starting at the data area (after the offsets array).
// cur accumulates the running data index as each symbol's entries are counted.
func writeCSROffsets(dst []byte, nsyms uint32, counts []int) []byte {
	binary.LittleEndian.PutUint32(dst, nsyms)
	offsetBase := dst[4:]
	cur := uint32(0)
	for i, c := range counts {
		binary.LittleEndian.PutUint32(offsetBase[i*4:], cur)
		cur += uint32(c)
	}
	// sentinel
	binary.LittleEndian.PutUint32(offsetBase[len(counts)*4:], cur)
	// return slice starting at data area
	return dst[4+(nsyms+1)*4:]
}

// writeOrdinaryEdges writes the OrdinaryEdges section.
//
//	nsyms   u32
//	offsets [nsyms+1] u32
//	data    [] u32  (Symbol)
func writeOrdinaryEdges(dst []byte, b *Builder, nsyms uint32) {
	counts := make([]int, nsyms)
	for i := range b.ordinaryEdges {
		counts[i] = len(b.ordinaryEdges[i])
	}
	data := writeCSROffsets(dst, nsyms, counts)
	pos := 0
	for _, es := range b.ordinaryEdges {
		for _, target := range es {
			binary.LittleEndian.PutUint32(data[pos:], uint32(target))
			pos += 4
		}
	}
}

// writeFuncDemand writes the FuncDemand section.
//
//	nsyms   u32
//	offsets [nsyms+1] u32
//	data    [] { kind u32, target u32, extra u32 }  (12 bytes each)
func writeFuncDemand(dst []byte, b *Builder, nsyms uint32) {
	counts := make([]int, nsyms)
	for i := range b.funcDemands {
		counts[i] = len(b.funcDemands[i])
	}
	data := writeCSROffsets(dst, nsyms, counts)
	const rec = 12
	pos := 0
	for _, ds := range b.funcDemands {
		for _, d := range ds {
			binary.LittleEndian.PutUint32(data[pos:], uint32(d.kind))
			binary.LittleEndian.PutUint32(data[pos+4:], d.target)
			binary.LittleEndian.PutUint32(data[pos+8:], d.extra)
			pos += rec
		}
	}
}

// writeTypeChildren writes the TypeChildren section.
//
//	nsyms   u32
//	offsets [nsyms+1] u32
//	data    [] u32  (Symbol)
func writeTypeChildren(dst []byte, b *Builder, nsyms uint32) {
	counts := make([]int, nsyms)
	for i := range b.typeChildren {
		counts[i] = len(b.typeChildren[i])
	}
	data := writeCSROffsets(dst, nsyms, counts)
	pos := 0
	for _, cs := range b.typeChildren {
		for _, child := range cs {
			binary.LittleEndian.PutUint32(data[pos:], uint32(child))
			pos += 4
		}
	}
}

// writeMethodInfo writes the MethodInfo section.
//
//	nsyms   u32
//	offsets [nsyms+1] u32
//	data    [] { nameOff u32, nameLen u32, mtype u32, ifn u32, tfn u32 }  (20 bytes each)
func writeMethodInfo(dst []byte, b *Builder, nsyms uint32) {
	counts := make([]int, nsyms)
	for i := range b.methodInfo {
		counts[i] = len(b.methodInfo[i])
	}
	data := writeCSROffsets(dst, nsyms, counts)
	const rec = 20
	pos := 0
	for _, slots := range b.methodInfo {
		for _, slot := range slots {
			binary.LittleEndian.PutUint32(data[pos:], slot.name.Off)
			binary.LittleEndian.PutUint32(data[pos+4:], slot.name.Len)
			binary.LittleEndian.PutUint32(data[pos+8:], slot.mtype)
			binary.LittleEndian.PutUint32(data[pos+12:], slot.ifn)
			binary.LittleEndian.PutUint32(data[pos+16:], slot.tfn)
			pos += rec
		}
	}
}

// writeIfaceInfo writes the InterfaceInfo section.
//
//	nsyms   u32
//	offsets [nsyms+1] u32
//	data    [] { nameOff u32, nameLen u32, mtype u32 }  (12 bytes each)
func writeIfaceInfo(dst []byte, b *Builder, nsyms uint32) {
	counts := make([]int, nsyms)
	for i := range b.ifaceInfo {
		counts[i] = len(b.ifaceInfo[i])
	}
	data := writeCSROffsets(dst, nsyms, counts)
	const rec = 12
	pos := 0
	for _, sigs := range b.ifaceInfo {
		for _, sig := range sigs {
			binary.LittleEndian.PutUint32(data[pos:], sig.name.Off)
			binary.LittleEndian.PutUint32(data[pos+4:], sig.name.Len)
			binary.LittleEndian.PutUint32(data[pos+8:], sig.mtype)
			pos += rec
		}
	}
}

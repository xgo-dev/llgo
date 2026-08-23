package meta

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

// nameRef identifies a byte range in the package-local string table.
type nameRef struct {
	Off uint32
	Len uint32
}

// version is the compatibility boundary for the on-disk layout.
const (
	magic   = "LLPM"
	version = 1
)

// Section IDs are also indexes into the header's section-offset array.
// Reordering or changing their wire representation is a format change.
const (
	secStringTable = iota
	secSymbols
	secOrdinaryEdges
	secFuncDemand
	secTypeChildren
	secMethodInfo
	secIfaceInfo
	numSections
)

// headerSize = magic(4) + version(4) + sectionOffsets(numSections*4)
const headerSize = 4 + 4 + numSections*4

// PackageMeta is a read-only, zero-copy view of one package's metadata.
// Its backing bytes are either owned Go memory from Builder.Build or a
// read-only mapping created by Open. Package-local lookup helpers return
// strings and slices that alias those bytes.
//
// The version-1 wire format is below. All integers are little-endian uint32
// values. Header offsets are absolute byte offsets from the beginning of the
// file. Symbol values are indexes in this package's Symbols section.
//
//	Header (36 bytes)
//	  [0:4]   magic: "LLPM"
//	  [4:8]   version: 1
//	  [8:36]  section offsets, in this exact order:
//	            StringTable, Symbols, OrdinaryEdges, FuncDemand,
//	            TypeChildren, MethodInfo, InterfaceInfo
//
//	StringTable
//	  starts at byte 36
//	  concatenated string bytes, followed by zero padding to a 4-byte boundary
//
//	Symbols
//	  nsyms
//	  records[nsyms]: {nameOff, nameLen, reserved}                    // 12 bytes
//
//	OrdinaryEdges: CSR<Symbol>
//	  nsyms
//	  offsets[nsyms+1]
//	  data[]: Symbol                                                  // 4 bytes
//
//	FuncDemand: CSR<localFuncDemand>
//	  nsyms
//	  offsets[nsyms+1]
//	  data[]: {kind, target, extra}                                  // 12 bytes
//
//	TypeChildren: CSR<Symbol>
//	  nsyms
//	  offsets[nsyms+1]
//	  data[]: Symbol                                                  // 4 bytes
//
//	MethodInfo: CSR<localMethodSlot>
//	  nsyms
//	  offsets[nsyms+1]
//	  data[]: {nameOff, nameLen, mtype, ifn, tfn}                    // 20 bytes
//
//	InterfaceInfo: CSR<localMethodSig>
//	  nsyms
//	  offsets[nsyms+1]
//	  data[]: {nameOff, nameLen, mtype}                              // 12 bytes
//
// Every CSR offsets entry is a record index into that section's data array,
// not a byte offset. Each CSR nsyms must equal Symbols.nsyms. Every nameOff is
// relative to the start of StringTable; nameLen excludes alignment padding.
// The Symbols reserved field is written as zero and ignored when reading.
// Section sizes are derived from adjacent header offsets, and InterfaceInfo
// extends to the end of the file. Every section starts on a 4-byte boundary.
type PackageMeta struct {
	raw  []byte
	mmap bool // whether Close must unmap raw

	nsyms uint32

	// cached section start offsets (parsed once from header)
	strOff      uint32
	symOff      uint32
	ordinaryOff uint32
	demandOff   uint32
	childOff    uint32
	methodOff   uint32
	ifaceOff    uint32
}

// localFuncDemand is the package-local wire representation of a function
// demand. Its layout (Kind@0, Target@4, Extra@8, size 12) must match the file
// format so funcDemands can return a zero-copy view of the backing bytes.
type localFuncDemand struct {
	Kind DemandKind
	// Target is a Symbol for DemandUseIface and DemandIfaceMethod, a string-table
	// offset for DemandNamedMethod, and zero for DemandReflectMethod.
	Target uint32
	// Extra is an interface-method index for DemandIfaceMethod, a string length
	// for DemandNamedMethod, and zero for the other kinds.
	Extra uint32
}

// Compile-time assertion: localFuncDemand must be exactly 12 bytes. If either const
// goes negative the build fails, pinning the wire/struct layout match.
const (
	_ = uint(unsafe.Sizeof(localFuncDemand{}) - 12)
	_ = uint(12 - unsafe.Sizeof(localFuncDemand{}))
)

// localMethodSlot is the package-local wire representation of an ABI method
// slot. Its layout (nameRef@0..8, MType@8, IFn@12, TFn@16, size 20) must match
// the file format for zero-copy reads.
type localMethodSlot struct {
	Name  nameRef // bare if exported; package-qualified if unexported
	MType Symbol
	IFn   Symbol
	TFn   Symbol
}

// localMethodSig is the package-local wire representation of an interface
// method signature. Its layout (nameRef@0..8, MType@8, size 12) must match the
// file format for zero-copy reads.
type localMethodSig struct {
	Name  nameRef // bare if exported; package-qualified if unexported
	MType Symbol
}

// Compile-time assertions pinning the wire/struct layout for zero-copy reads.
// If a struct's size drifts, one of these uint consts goes negative and the
// build fails.
const (
	_ = uint(unsafe.Sizeof(localMethodSlot{}) - 20)
	_ = uint(20 - unsafe.Sizeof(localMethodSlot{}))
	_ = uint(unsafe.Sizeof(localMethodSig{}) - 12)
	_ = uint(12 - unsafe.Sizeof(localMethodSig{}))
)

// Open maps path read-only and returns a PackageMeta backed by that mapping.
// The caller must call Close. Values returned by package-local lookup helpers
// must not be used after Close.
func Open(path string) (*PackageMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(fi.Size())

	raw, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("meta: mmap %s: %w", path, err)
	}

	pm, err := newPackageMeta(raw)
	if err != nil {
		_ = syscall.Munmap(raw)
		return nil, err
	}
	pm.mmap = true
	return pm, nil
}

// WriteTo writes the complete metadata file in its binary wire format.
func (pm *PackageMeta) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(pm.raw)
	if err == nil && n != len(pm.raw) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}

// Close releases the mapping owned by a PackageMeta returned from Open.
// It is a no-op for PackageMeta values returned from Builder.Build.
func (pm *PackageMeta) Close() error {
	if pm.mmap && pm.raw != nil {
		err := syscall.Munmap(pm.raw)
		pm.raw = nil
		return err
	}
	return nil
}

// DemandFunctionNames returns package-local symbols that own function-scoped
// interface, method, or reflection demands. The returned strings alias the
// PackageMeta backing bytes and must not be used after Close.
func (pm *PackageMeta) DemandFunctionNames() []string {
	if pm == nil {
		return nil
	}
	var names []string
	for sym := Symbol(0); sym < Symbol(pm.nsyms); sym++ {
		if pm.hasFuncDemand(sym) {
			names = append(names, pm.symbolName(sym))
		}
	}
	return names
}

// symbolName returns the name of package-local sym as a string that aliases the
// backing bytes. For a PackageMeta returned from Open, the string must not be
// used after Close.
func (pm *PackageMeta) symbolName(sym Symbol) string {
	const recSize = 12
	base := pm.symOff + 4 + uint32(sym)*recSize
	nameOff := binary.LittleEndian.Uint32(pm.raw[base+0:])
	nameLen := binary.LittleEndian.Uint32(pm.raw[base+4:])
	return unsafe.String(&pm.raw[pm.strOff+nameOff], int(nameLen))
}

// nameString returns the string referenced by a valid package-local ref. The
// string aliases the backing bytes and, for a PackageMeta returned from Open,
// must not be used after Close.
func (pm *PackageMeta) nameString(ref nameRef) string {
	return unsafe.String(&pm.raw[pm.strOff+ref.Off], int(ref.Len))
}

// nordinaryEdge returns the number of ordinary reachability edges from sym.
func (pm *PackageMeta) nordinaryEdge(sym Symbol) uint32 {
	s, e := pm.csrRange(pm.ordinaryOff, sym)
	return e - s
}

// ordinaryEdges returns the package-local ordinary-edge targets from sym. The
// returned slice aliases the backing bytes.
func (pm *PackageMeta) ordinaryEdges(sym Symbol) []Symbol {
	return csrSlice[Symbol](pm, pm.ordinaryOff, sym, 4)
}

// nfuncDemand returns the number of function demands owned by sym.
func (pm *PackageMeta) nfuncDemand(sym Symbol) uint32 {
	s, e := pm.csrRange(pm.demandOff, sym)
	return e - s
}

// funcDemands returns the package-local function-demand records owned by sym.
// The returned slice aliases the backing bytes.
func (pm *PackageMeta) funcDemands(sym Symbol) []localFuncDemand {
	return csrSlice[localFuncDemand](pm, pm.demandOff, sym, 12)
}

// ntypeChild returns the number of type children recorded for sym.
func (pm *PackageMeta) ntypeChild(sym Symbol) uint32 {
	s, e := pm.csrRange(pm.childOff, sym)
	return e - s
}

// typeChildren returns the package-local child type Symbols recorded for sym.
// The returned slice aliases the backing bytes.
func (pm *PackageMeta) typeChildren(sym Symbol) []Symbol {
	return csrSlice[Symbol](pm, pm.childOff, sym, 4)
}

// nmethodSlot returns the number of ABI method slots recorded for sym.
func (pm *PackageMeta) nmethodSlot(sym Symbol) uint32 {
	s, e := pm.csrRange(pm.methodOff, sym)
	return e - s
}

// methodSlots returns the package-local ABI method slots recorded for sym. The
// returned slice aliases the backing bytes.
func (pm *PackageMeta) methodSlots(sym Symbol) []localMethodSlot {
	return csrSlice[localMethodSlot](pm, pm.methodOff, sym, 20)
}

// nifaceMethod returns the number of interface method signatures recorded for
// sym.
func (pm *PackageMeta) nifaceMethod(sym Symbol) uint32 {
	s, e := pm.csrRange(pm.ifaceOff, sym)
	return e - s
}

// ifaceMethods returns the package-local interface method signatures recorded
// for sym. The returned slice aliases the backing bytes.
func (pm *PackageMeta) ifaceMethods(sym Symbol) []localMethodSig {
	return csrSlice[localMethodSig](pm, pm.ifaceOff, sym, 12)
}

// hasOrdinaryEdges reports whether sym owns any ordinary reachability edges.
func (pm *PackageMeta) hasOrdinaryEdges(sym Symbol) bool {
	return pm.nordinaryEdge(sym) > 0
}

// hasFuncDemand reports whether sym owns any function demand.
func (pm *PackageMeta) hasFuncDemand(sym Symbol) bool {
	return pm.nfuncDemand(sym) > 0
}

// ── internal helpers ──────────────────────────────────────────────────────────

// newPackageMeta checks the magic and version, then decodes the section offsets
// from the fixed header.
func newPackageMeta(raw []byte) (*PackageMeta, error) {
	if string(raw[0:4]) != magic {
		return nil, fmt.Errorf("meta: bad magic %q", raw[0:4])
	}
	ver := binary.LittleEndian.Uint32(raw[4:8])
	if ver != version {
		return nil, fmt.Errorf("meta: unsupported version %d", ver)
	}

	pm := &PackageMeta{raw: raw}
	pm.strOff = binary.LittleEndian.Uint32(raw[8+secStringTable*4:])
	pm.symOff = binary.LittleEndian.Uint32(raw[8+secSymbols*4:])
	pm.ordinaryOff = binary.LittleEndian.Uint32(raw[8+secOrdinaryEdges*4:])
	pm.demandOff = binary.LittleEndian.Uint32(raw[8+secFuncDemand*4:])
	pm.childOff = binary.LittleEndian.Uint32(raw[8+secTypeChildren*4:])
	pm.methodOff = binary.LittleEndian.Uint32(raw[8+secMethodInfo*4:])
	pm.ifaceOff = binary.LittleEndian.Uint32(raw[8+secIfaceInfo*4:])

	// read nsyms from Symbols section header
	pm.nsyms = binary.LittleEndian.Uint32(raw[pm.symOff:])
	return pm, nil
}

// csrSlice returns the records for package-local sym from a CSR section, or nil
// if it has no records. The returned slice aliases pm.raw. recSize must match
// unsafe.Sizeof(T).
func csrSlice[T any](pm *PackageMeta, sectionOff uint32, sym Symbol, recSize uintptr) []T {
	start, end := pm.csrRange(sectionOff, sym)
	if start == end {
		return nil
	}
	dataBase := sectionOff + 4 + (pm.nsyms+1)*4
	p := (*T)(unsafe.Pointer(&pm.raw[dataBase+uint32(uintptr(start)*recSize)]))
	return unsafe.Slice(p, end-start)
}

// csrRange returns the half-open data-record range for package-local sym.
// Callers must pass an in-range sym and a section offset decoded from pm.
func (pm *PackageMeta) csrRange(sectionOff uint32, sym Symbol) (start, end uint32) {
	offsetsBase := sectionOff + 4 // skip nsyms u32
	start = binary.LittleEndian.Uint32(pm.raw[offsetsBase+uint32(sym)*4:])
	end = binary.LittleEndian.Uint32(pm.raw[offsetsBase+(uint32(sym)+1)*4:])
	return
}

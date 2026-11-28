package meta

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
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
	if fi.Size() < headerSize || uint64(fi.Size()) > uint64(^uint32(0)) || fi.Size() > int64(^uint(0)>>1) {
		return nil, fmt.Errorf("meta: mmap %s: invalid file size %d", path, fi.Size())
	}
	size := int(fi.Size())

	raw, err := mapFile(f, size)
	if err != nil {
		return nil, fmt.Errorf("meta: mmap %s: %w", path, err)
	}

	pm, err := newPackageMeta(raw)
	if err != nil {
		_ = unmapFile(raw)
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
		err := unmapFile(pm.raw)
		pm.raw = nil
		pm.mmap = false
		return err
	}
	return nil
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
	if err := validateMetaSize(uint64(len(raw))); err != nil {
		return nil, err
	}
	if string(raw[0:4]) != magic {
		return nil, fmt.Errorf("meta: bad magic %q", raw[0:4])
	}
	ver := binary.LittleEndian.Uint32(raw[4:8])
	if ver != version {
		return nil, fmt.Errorf("meta: unsupported version %d", ver)
	}

	var offsets [numSections]uint32
	prev := uint32(headerSize)
	for sec := range offsets {
		off := binary.LittleEndian.Uint32(raw[8+sec*4:])
		if off < prev || uint64(off) > uint64(len(raw)) || off%4 != 0 {
			return nil, fmt.Errorf("meta: invalid section %d offset %d", sec, off)
		}
		offsets[sec] = off
		prev = off
	}
	if offsets[secOrdinaryEdges]-offsets[secSymbols] < 4 {
		return nil, fmt.Errorf("meta: truncated symbols section")
	}
	nsyms := binary.LittleEndian.Uint32(raw[offsets[secSymbols]:])
	pm := packageMetaView(raw, offsets, nsyms)
	if err := pm.validate(); err != nil {
		return nil, err
	}
	return pm, nil
}

func validateMetaSize(size uint64) error {
	if size < headerSize {
		return fmt.Errorf("meta: file too small: %d bytes", size)
	}
	if size > uint64(^uint32(0)) {
		return fmt.Errorf("meta: file too large: %d bytes", size)
	}
	return nil
}

// packageMetaView constructs a view over a layout whose header has already
// been decoded. Builder uses it for bytes it just wrote; Open validates the
// returned view before exposing mapped, file-controlled bytes.
func packageMetaView(raw []byte, offsets [numSections]uint32, nsyms uint32) *PackageMeta {
	return &PackageMeta{
		raw:         raw,
		nsyms:       nsyms,
		strOff:      offsets[secStringTable],
		symOff:      offsets[secSymbols],
		ordinaryOff: offsets[secOrdinaryEdges],
		demandOff:   offsets[secFuncDemand],
		childOff:    offsets[secTypeChildren],
		methodOff:   offsets[secMethodInfo],
		ifaceOff:    offsets[secIfaceInfo],
	}
}

type csrLayout struct {
	dataOff  uint32
	nrecords uint32
}

func (pm *PackageMeta) validate() error {
	symSize := uint64(pm.ordinaryOff - pm.symOff)
	wantSymSize := uint64(4) + uint64(pm.nsyms)*12
	if symSize != wantSymSize {
		return fmt.Errorf("meta: invalid symbols section size %d for %d symbols", symSize, pm.nsyms)
	}

	sections := [...]struct {
		name       string
		start      uint32
		end        uint32
		recordSize uint32
	}{
		{name: "OrdinaryEdges", start: pm.ordinaryOff, end: pm.demandOff, recordSize: 4},
		{name: "FuncDemand", start: pm.demandOff, end: pm.childOff, recordSize: 12},
		{name: "TypeChildren", start: pm.childOff, end: pm.methodOff, recordSize: 4},
		{name: "MethodInfo", start: pm.methodOff, end: pm.ifaceOff, recordSize: 20},
		{name: "InterfaceInfo", start: pm.ifaceOff, end: uint32(len(pm.raw)), recordSize: 12},
	}
	var layouts [len(sections)]csrLayout
	for i, section := range sections {
		layout, err := validateCSRSection(pm.raw, section.name, section.start, section.end, pm.nsyms, section.recordSize)
		if err != nil {
			return err
		}
		layouts[i] = layout
	}

	strSize := pm.symOff - pm.strOff
	if err := pm.validateNameRecords("Symbols", pm.symOff+4, pm.nsyms, 12, strSize); err != nil {
		return err
	}
	if err := pm.validateSymbolRecords(sections[0].name, layouts[0], 4, symbolField{}); err != nil {
		return err
	}
	if err := pm.validateFuncDemands(layouts[1], strSize); err != nil {
		return err
	}
	if err := pm.validateSymbolRecords(sections[2].name, layouts[2], 4, symbolField{}); err != nil {
		return err
	}
	if err := pm.validateNameRecords(sections[3].name, layouts[3].dataOff, layouts[3].nrecords, 20, strSize); err != nil {
		return err
	}
	if err := pm.validateSymbolRecords(sections[3].name, layouts[3], 20,
		symbolField{name: "MType", offset: 8},
		symbolField{name: "IFn", offset: 12},
		symbolField{name: "TFn", offset: 16},
	); err != nil {
		return err
	}
	if err := pm.validateNameRecords(sections[4].name, layouts[4].dataOff, layouts[4].nrecords, 12, strSize); err != nil {
		return err
	}
	return pm.validateSymbolRecords(sections[4].name, layouts[4], 12, symbolField{name: "MType", offset: 8})
}

func validNameRef(ref nameRef, strSize uint32) bool {
	return uint64(ref.Off)+uint64(ref.Len) <= uint64(strSize)
}

func validateCSRSection(raw []byte, name string, start, end, nsyms, recordSize uint32) (csrLayout, error) {
	sectionSize := uint64(end - start)
	headerSize := uint64(4) + (uint64(nsyms)+1)*4
	if sectionSize < headerSize {
		return csrLayout{}, fmt.Errorf("meta: truncated %s CSR header", name)
	}
	if got := binary.LittleEndian.Uint32(raw[start:]); got != nsyms {
		return csrLayout{}, fmt.Errorf("meta: %s has %d symbols, want %d", name, got, nsyms)
	}
	dataSize := sectionSize - headerSize
	if dataSize%uint64(recordSize) != 0 {
		return csrLayout{}, fmt.Errorf("meta: invalid %s data size %d", name, dataSize)
	}
	nrecords := dataSize / uint64(recordSize)
	offsetsBase := start + 4
	prev := binary.LittleEndian.Uint32(raw[offsetsBase:])
	if prev != 0 {
		return csrLayout{}, fmt.Errorf("meta: %s CSR first offset is %d, want 0", name, prev)
	}
	for sym := uint32(0); sym < nsyms; sym++ {
		cur := binary.LittleEndian.Uint32(raw[offsetsBase+(sym+1)*4:])
		if cur < prev || uint64(cur) > nrecords {
			return csrLayout{}, fmt.Errorf("meta: invalid %s CSR offset %d at symbol %d", name, cur, sym)
		}
		prev = cur
	}
	if uint64(prev) != nrecords {
		return csrLayout{}, fmt.Errorf("meta: %s CSR covers %d records, section contains %d", name, prev, nrecords)
	}
	return csrLayout{dataOff: uint32(uint64(start) + headerSize), nrecords: uint32(nrecords)}, nil
}

func (pm *PackageMeta) validateFuncDemands(layout csrLayout, strSize uint32) error {
	for i := uint32(0); i < layout.nrecords; i++ {
		base := layout.dataOff + i*12
		kind := DemandKind(binary.LittleEndian.Uint32(pm.raw[base:]))
		switch kind {
		case DemandUseIface, DemandIfaceMethod:
			target := binary.LittleEndian.Uint32(pm.raw[base+4:])
			if target >= pm.nsyms {
				return fmt.Errorf("meta: FuncDemand record %d has invalid target symbol %d (symbol count %d)", i, target, pm.nsyms)
			}
		case DemandNamedMethod:
			ref := nameRef{
				Off: binary.LittleEndian.Uint32(pm.raw[base+4:]),
				Len: binary.LittleEndian.Uint32(pm.raw[base+8:]),
			}
			if !validNameRef(ref, strSize) {
				return fmt.Errorf("meta: FuncDemand record %d has invalid name range [%d,%d)", i, ref.Off, uint64(ref.Off)+uint64(ref.Len))
			}
		}
	}
	return nil
}

type symbolField struct {
	name   string
	offset uint32
}

func (pm *PackageMeta) validateSymbolRecords(section string, layout csrLayout, recordSize uint32, fields ...symbolField) error {
	for i := uint32(0); i < layout.nrecords; i++ {
		base := layout.dataOff + i*recordSize
		for _, field := range fields {
			sym := binary.LittleEndian.Uint32(pm.raw[base+field.offset:])
			if sym < pm.nsyms {
				continue
			}
			if field.name == "" {
				return fmt.Errorf("meta: %s record %d has invalid symbol %d (symbol count %d)", section, i, sym, pm.nsyms)
			}
			return fmt.Errorf("meta: %s record %d has invalid %s symbol %d (symbol count %d)", section, i, field.name, sym, pm.nsyms)
		}
	}
	return nil
}

func (pm *PackageMeta) validateNameRecords(section string, dataOff, nrecords, recordSize, strSize uint32) error {
	for i := uint32(0); i < nrecords; i++ {
		base := dataOff + i*recordSize
		ref := nameRef{
			Off: binary.LittleEndian.Uint32(pm.raw[base:]),
			Len: binary.LittleEndian.Uint32(pm.raw[base+4:]),
		}
		if !validNameRef(ref, strSize) {
			return fmt.Errorf("meta: %s record %d has invalid name range [%d,%d)", section, i, ref.Off, uint64(ref.Off)+uint64(ref.Len))
		}
	}
	return nil
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

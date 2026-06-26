package meta

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// PackageMeta is a zero-copy view over a .meta file byte slice.
// The underlying bytes may come from an mmap'd file or from Builder.Build().
// All query methods read directly from the byte slice with no allocation.
type PackageMeta struct {
	raw  []byte
	mmap bool // true → must Munmap on Close

	nsyms uint32

	// cached section start offsets (parsed once from header)
	strOff    uint32
	symOff    uint32
	edgeOff   uint32
	childOff  uint32
	methodOff uint32
	ifaceOff  uint32
	reflOff   uint32
}

// Edge is a decoded edge record returned by query methods. Its in-memory layout
// (Target@0, Extra@4, Kind@8, size 12) must match the on-disk wire layout exactly
// so Edges can reinterpret the mmap bytes as []Edge with no copy.
type Edge struct {
	Target uint32 // LocalSymbol or stringTable offset (UseNamedMethod)
	Extra  uint32
	Kind   uint8
}

// Compile-time assertion: Edge must be exactly 12 bytes. If either const goes
// negative the build fails, pinning the wire/struct layout match.
const (
	_ = uint(unsafe.Sizeof(Edge{}) - 12)
	_ = uint(12 - unsafe.Sizeof(Edge{}))
)

// MethodSlot is a decoded method slot record. Its layout (NameRef@0..8,
// MType@8, IFn@12, TFn@16, size 20) must match the on-disk wire layout for
// zero-copy reads.
type MethodSlot struct {
	Name  NameRef // method short name
	MType LocalSymbol
	IFn   LocalSymbol
	TFn   LocalSymbol
}

// MethodSig is a decoded interface method signature. Layout: NameRef@0..8,
// MType@8, size 12 — must match the on-disk wire layout for zero-copy reads.
type MethodSig struct {
	Name  NameRef // method short name
	MType LocalSymbol
}

// Compile-time assertions pinning the wire/struct layout for zero-copy reads.
// If a struct's size drifts, one of these uint consts goes negative and the
// build fails.
const (
	_ = uint(unsafe.Sizeof(MethodSlot{}) - 20)
	_ = uint(20 - unsafe.Sizeof(MethodSlot{}))
	_ = uint(unsafe.Sizeof(MethodSig{}) - 12)
	_ = uint(12 - unsafe.Sizeof(MethodSig{}))
)

// ReadMeta opens path, mmaps it, and returns a PackageMeta view.
// Call Close when done to release the mapping.
func ReadMeta(path string) (*PackageMeta, error) {
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
	if size < headerSize {
		return nil, fmt.Errorf("meta: file too small: %s", path)
	}

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

// NewPackageMetaFromBytes wraps an in-memory byte slice as a PackageMeta.
// The slice must remain valid for the lifetime of the returned PackageMeta.
func NewPackageMetaFromBytes(raw []byte) (*PackageMeta, error) {
	return newPackageMeta(raw)
}

// Bytes returns the underlying raw byte slice (for writing to disk).
func (pm *PackageMeta) Bytes() []byte { return pm.raw }

// Close releases the mmap mapping if one was used.
func (pm *PackageMeta) Close() error {
	if pm.mmap && pm.raw != nil {
		err := syscall.Munmap(pm.raw)
		pm.raw = nil
		return err
	}
	return nil
}

// NSyms returns the number of symbols in this package.
func (pm *PackageMeta) NSyms() uint32 { return pm.nsyms }

// SymbolName returns the name of sym by reading directly from the string table.
func (pm *PackageMeta) SymbolName(sym LocalSymbol) string {
	if uint32(sym) >= pm.nsyms {
		return ""
	}
	const recSize = 12
	base := pm.symOff + 4 + uint32(sym)*recSize
	nameOff := binary.LittleEndian.Uint32(pm.raw[base+0:])
	nameLen := binary.LittleEndian.Uint32(pm.raw[base+4:])
	return string(pm.raw[pm.strOff+nameOff : pm.strOff+nameOff+nameLen])
}

// NameString returns the string referenced by a NameRef.
func (pm *PackageMeta) NameString(ref NameRef) string {
	return string(pm.raw[pm.strOff+ref.Off : pm.strOff+ref.Off+ref.Len])
}

// Edges returns all edges from sym as a zero-copy view into the mmap region:
// the on-disk 12-byte edge records are reinterpreted directly as []Edge. The
// result is only valid for the lifetime of pm and must not be mutated. Assumes
// a little-endian host and 4-byte aligned data (guaranteed by stringTable padding).
func (pm *PackageMeta) Edges(sym LocalSymbol) []Edge {
	if uint32(sym) >= pm.nsyms {
		return nil
	}
	const recSize = 12
	start, end := pm.csrRange(pm.edgeOff, sym)
	if start == end {
		return nil
	}
	dataBase := pm.edgeOff + 4 + (pm.nsyms+1)*4
	p := (*Edge)(unsafe.Pointer(&pm.raw[dataBase+start*recSize]))
	return unsafe.Slice(p, end-start)
}

// TypeChildren returns the child type LocalSymbols for sym.
//
// The returned slice is a zero-copy view directly into the mmap region: the
// on-disk uint32 array is reinterpreted as []LocalSymbol with no allocation.
// It is only valid for the lifetime of pm (i.e. until Close) and must not be
// mutated. This assumes a little-endian host (matching the wire format) and a
// 4-byte aligned data section (guaranteed by stringTable padding).
func (pm *PackageMeta) TypeChildren(sym LocalSymbol) []LocalSymbol {
	if uint32(sym) >= pm.nsyms {
		return nil
	}
	start, end := pm.csrRange(pm.childOff, sym)
	if start == end {
		return nil
	}
	dataBase := pm.childOff + 4 + (pm.nsyms+1)*4
	p := (*LocalSymbol)(unsafe.Pointer(&pm.raw[dataBase+start*4]))
	return unsafe.Slice(p, end-start)
}

// MethodSlots returns the ABI method slots for concrete type sym as a zero-copy
// view into the mmap region. Valid only for the lifetime of pm; do not mutate.
func (pm *PackageMeta) MethodSlots(sym LocalSymbol) []MethodSlot {
	if uint32(sym) >= pm.nsyms {
		return nil
	}
	const recSize = 20
	start, end := pm.csrRange(pm.methodOff, sym)
	if start == end {
		return nil
	}
	dataBase := pm.methodOff + 4 + (pm.nsyms+1)*4
	p := (*MethodSlot)(unsafe.Pointer(&pm.raw[dataBase+start*recSize]))
	return unsafe.Slice(p, end-start)
}

// IfaceMethods returns the method signatures for interface sym as a zero-copy
// view into the mmap region. Valid only for the lifetime of pm; do not mutate.
func (pm *PackageMeta) IfaceMethods(sym LocalSymbol) []MethodSig {
	if uint32(sym) >= pm.nsyms {
		return nil
	}
	const recSize = 12
	start, end := pm.csrRange(pm.ifaceOff, sym)
	if start == end {
		return nil
	}
	dataBase := pm.ifaceOff + 4 + (pm.nsyms+1)*4
	p := (*MethodSig)(unsafe.Pointer(&pm.raw[dataBase+start*recSize]))
	return unsafe.Slice(p, end-start)
}

// HasReflect reports whether sym triggers conservative reflection handling.
func (pm *PackageMeta) HasReflect(sym LocalSymbol) bool {
	if uint32(sym) >= pm.nsyms {
		return false
	}
	bitmapBase := pm.reflOff + 4
	return pm.raw[bitmapBase+uint32(sym)/8]&(1<<(sym%8)) != 0
}

// IsCompositeType reports whether sym has TypeChildren (i.e. is a composite type).
func (pm *PackageMeta) IsCompositeType(sym LocalSymbol) bool {
	s, e := pm.csrRange(pm.childOff, sym)
	return s != e
}

// HasEdges reports whether sym has any outgoing edges.
func (pm *PackageMeta) HasEdges(sym LocalSymbol) bool {
	s, e := pm.csrRange(pm.edgeOff, sym)
	return s != e
}

// IsConcreteType reports whether sym has MethodSlots (i.e. is a concrete type with methods).
func (pm *PackageMeta) IsConcreteType(sym LocalSymbol) bool {
	s, e := pm.csrRange(pm.methodOff, sym)
	return s != e
}

// IsInterface reports whether sym has interface methods (i.e. is an interface type).
func (pm *PackageMeta) IsInterface(sym LocalSymbol) bool {
	s, e := pm.csrRange(pm.ifaceOff, sym)
	return s != e
}

// ── internal helpers ──────────────────────────────────────────────────────────

// newPackageMeta parses the header of raw and returns a PackageMeta.
func newPackageMeta(raw []byte) (*PackageMeta, error) {
	if len(raw) < headerSize {
		return nil, fmt.Errorf("meta: raw too small (%d bytes)", len(raw))
	}
	if string(raw[0:4]) != Magic {
		return nil, fmt.Errorf("meta: bad magic %q", raw[0:4])
	}
	ver := binary.LittleEndian.Uint32(raw[4:8])
	if ver != Version {
		return nil, fmt.Errorf("meta: unsupported version %d", ver)
	}

	pm := &PackageMeta{raw: raw}
	pm.strOff = binary.LittleEndian.Uint32(raw[8+SecStringTable*4:])
	pm.symOff = binary.LittleEndian.Uint32(raw[8+SecSymbols*4:])
	pm.edgeOff = binary.LittleEndian.Uint32(raw[8+SecEdges*4:])
	pm.childOff = binary.LittleEndian.Uint32(raw[8+SecTypeChildren*4:])
	pm.methodOff = binary.LittleEndian.Uint32(raw[8+SecMethodInfo*4:])
	pm.ifaceOff = binary.LittleEndian.Uint32(raw[8+SecIfaceInfo*4:])
	pm.reflOff = binary.LittleEndian.Uint32(raw[8+SecReflect*4:])

	// read nsyms from Symbols section header
	pm.nsyms = binary.LittleEndian.Uint32(raw[pm.symOff:])
	return pm, nil
}

// csrRange returns [start, end) data indices for sym in the CSR section
// starting at sectionOff.
func (pm *PackageMeta) csrRange(sectionOff uint32, sym LocalSymbol) (start, end uint32) {
	offsetsBase := sectionOff + 4 // skip nsyms u32
	start = binary.LittleEndian.Uint32(pm.raw[offsetsBase+uint32(sym)*4:])
	end = binary.LittleEndian.Uint32(pm.raw[offsetsBase+(uint32(sym)+1)*4:])
	return
}

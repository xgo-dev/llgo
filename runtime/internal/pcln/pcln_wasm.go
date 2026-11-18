//go:build llgo && llgo_pclntab_external && wasm

package pcln

import (
	"unsafe"

	latomic "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	externalPCLNMagic      = "LLGOPCL1"
	externalPCLNVersion    = uint32(4)
	externalPCLNABIVersion = uint32(1)
	externalPCLNHeaderSize = uintptr(256)
	externalPCLNMaxSize    = int64(64 << 20)
)

const (
	externalHeaderVersion     = 8
	externalHeaderSize        = 12
	externalHeaderPointerSize = 16
	externalHeaderEndian      = 17
	externalHeaderGOOS        = 18
	externalHeaderGOARCH      = 19
	externalHeaderABIVersion  = 20
	externalHeaderFileSize    = 24
	externalHeaderPayloadHash = 32
	externalHeaderTextStart   = 48
	externalHeaderTextEnd     = 56
	externalHeaderIdentity    = 64
	externalHeaderDescriptors = 96
)

const (
	externalDescRecords = iota
	externalDescPCLines
	externalDescStrings
	externalDescStringOffsets
	externalDescHash
	externalDescSymbolIndex
	externalDescEntrySites
	externalDescPCSites
	externalDescCount
)

const (
	externalRecordSize       = uintptr(28)
	externalPCLineSize       = uintptr(24)
	externalStringOffsetSize = uintptr(4)
	externalHashSize         = uintptr(2)
	externalSymbolIndexSize  = uintptr(16)
	externalSiteSize         = uintptr(16)
)

const (
	// Keep zero-copy decoding strides synchronized with the runtime structs.
	_ = uintptr(0) - (externalRecordSize - unsafe.Sizeof(runtimeFuncInfoRecord{}))
	_ = uintptr(0) - (externalPCLineSize - unsafe.Sizeof(runtimePCLineRecord{}))
)

const (
	externalPCLNUnattempted uint32 = iota
	externalPCLNLoading
	externalPCLNLoaded
	externalPCLNUnavailable
)

//go:linkname runtimePCLNIdentity C.__llgo_pclntab_identity
var runtimePCLNIdentity [32]byte

var externalPCLNState uint32
var externalPCLNBacking []byte

type externalPCLNSection struct {
	offset uintptr
	count  uintptr
}

type externalPCLNView struct {
	textStart uintptr
	textEnd   uintptr
	sections  [externalDescCount]externalPCLNSection
}

func Ready() bool {
	return latomic.Load(&externalPCLNState) == externalPCLNLoaded
}

// Ensure loads metadata only on allocation/I/O-safe symbolization paths.
// Success and failure are terminal states. The JS provider requires poll to
// yield the current goroutine while its asynchronous read completes. When
// poll is nil, a caller observing another load in progress returns false
// instead of waiting. Fatal paths consult Ready without calling Ensure.
func Ensure(poll func()) bool {
	if externalPCLNQueryOnly {
		return Ready()
	}
	return Preload(poll)
}

// Preload starts the load on the runtime's startup path.
func Preload(poll func()) bool {
	for {
		switch latomic.Load(&externalPCLNState) {
		case externalPCLNLoaded:
			return true
		case externalPCLNUnavailable:
			return false
		case externalPCLNUnattempted:
			if _, ok := latomic.CompareAndExchange(&externalPCLNState, externalPCLNUnattempted, externalPCLNLoading); !ok {
				continue
			}
			if loadExternalPCLN(poll) {
				latomic.Store(&externalPCLNState, externalPCLNLoaded)
				return true
			}
			latomic.Store(&externalPCLNState, externalPCLNUnavailable)
			return false
		default:
			// Only safe callers wait. A signal path observes !Ready and
			// immediately degrades without entering this loop.
			if poll != nil {
				poll()
			}
			if poll == nil {
				return false
			}
		}
	}
}

func loadExternalPCLN(poll func()) bool {
	raw := readExternalPCLN(poll)
	if len(raw) == 0 {
		return false
	}
	installed := false
	defer func() {
		if !installed {
			c.Free(unsafe.Pointer(&raw[0]))
		}
	}()
	view, ok := validateExternalPCLN(raw, 0)
	if !ok || !installExternalPCLN(raw, view, 0) {
		return false
	}
	// The process keeps this C-owned buffer after successful validation; no
	// second, sidecar-sized Go heap allocation is needed during loading.
	externalPCLNBacking = raw
	installed = true
	return true
}

func externalUint32(raw []byte, off int) uint32 {
	return uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
}

func externalUint64(raw []byte, off int) uint64 {
	return uint64(externalUint32(raw, off)) | uint64(externalUint32(raw, off+4))<<32
}

func externalSectionSize(index int) uintptr {
	switch index {
	case externalDescRecords:
		return externalRecordSize
	case externalDescPCLines:
		return externalPCLineSize
	case externalDescStrings:
		return 1
	case externalDescStringOffsets:
		return externalStringOffsetSize
	case externalDescHash:
		return externalHashSize
	case externalDescSymbolIndex:
		return externalSymbolIndexSize
	case externalDescEntrySites, externalDescPCSites:
		return externalSiteSize
	}
	return 0
}

func validateExternalPCLN(raw []byte, loadBase uintptr) (externalPCLNView, bool) {
	var view externalPCLNView
	if len(raw) < int(externalPCLNHeaderSize) || string(raw[:len(externalPCLNMagic)]) != externalPCLNMagic {
		return view, false
	}
	if externalUint32(raw, externalHeaderVersion) != externalPCLNVersion ||
		externalUint32(raw, externalHeaderSize) != uint32(externalPCLNHeaderSize) ||
		externalUint32(raw, externalHeaderABIVersion) != externalPCLNABIVersion ||
		raw[externalHeaderPointerSize] != byte(nativePointerSize()) ||
		raw[externalHeaderEndian] != 1 || raw[externalHeaderGOOS] != externalPCLNGOOS ||
		raw[externalHeaderGOARCH] != externalPCLNGOARCH ||
		externalUint64(raw, externalHeaderFileSize) != uint64(len(raw)) {
		return view, false
	}
	for i := range runtimePCLNIdentity {
		if raw[externalHeaderIdentity+i] != runtimePCLNIdentity[i] {
			return view, false
		}
	}
	if externalUint64(raw, externalHeaderPayloadHash) != externalFNV64(raw[externalPCLNHeaderSize:]) {
		return view, false
	}
	textStart := externalUint64(raw, externalHeaderTextStart)
	textEnd := externalUint64(raw, externalHeaderTextEnd)
	if textEnd <= textStart || textEnd > uint64(^uintptr(0))-uint64(loadBase) {
		return view, false
	}
	view.textStart, view.textEnd = loadBase+uintptr(textStart), loadBase+uintptr(textEnd)
	for i := 0; i < externalDescCount; i++ {
		base := externalHeaderDescriptors + i*16
		off64, count64 := externalUint64(raw, base), externalUint64(raw, base+8)
		if off64 > uint64(^uintptr(0)) || count64 > uint64(^uintptr(0)) {
			return externalPCLNView{}, false
		}
		off, count, size := uintptr(off64), uintptr(count64), externalSectionSize(i)
		if off < externalPCLNHeaderSize || off > uintptr(len(raw)) || size == 0 || count > (uintptr(len(raw))-off)/size {
			return externalPCLNView{}, false
		}
		if size > 1 && off&(externalSectionAlignment(i)-1) != 0 {
			return externalPCLNView{}, false
		}
		view.sections[i] = externalPCLNSection{offset: off, count: count}
	}
	for i := 0; i < externalDescCount; i++ {
		left := view.sections[i]
		if left.count == 0 {
			continue
		}
		leftEnd := left.offset + left.count*externalSectionSize(i)
		for j := i + 1; j < externalDescCount; j++ {
			right := view.sections[j]
			if right.count == 0 {
				continue
			}
			rightEnd := right.offset + right.count*externalSectionSize(j)
			if left.offset < rightEnd && right.offset < leftEnd {
				return externalPCLNView{}, false
			}
		}
	}
	return view, true
}

func externalSectionAlignment(index int) uintptr {
	switch index {
	case externalDescRecords, externalDescStringOffsets:
		return 4
	case externalDescPCLines, externalDescSymbolIndex, externalDescEntrySites, externalDescPCSites:
		return 8
	case externalDescHash:
		return 2
	}
	return 1
}

func externalFNV64(raw []byte) uint64 {
	const (
		offset = uint64(14695981039346656037)
		prime  = uint64(1099511628211)
	)
	h := offset
	// Keep the sidecar-sized loop independent of the slice header. LLGo
	// currently keeps bounds-check temporaries alive until function return.
	n := len(raw)
	if n == 0 {
		return h
	}
	base := unsafe.Pointer(&raw[0])
	for i := 0; i < n; i++ {
		b := *(*byte)(unsafe.Add(base, uintptr(i)))
		h ^= uint64(b)
		h *= prime
	}
	return h
}

func externalSectionPtr(raw []byte, sec externalPCLNSection) unsafe.Pointer {
	if sec.count == 0 {
		return nil
	}
	return unsafe.Pointer(&raw[sec.offset])
}

func installExternalPCLN(raw []byte, view externalPCLNView, loadBase uintptr) bool {
	records := view.sections[externalDescRecords]
	pclines := view.sections[externalDescPCLines]
	stringsSec := view.sections[externalDescStrings]
	offsets := view.sections[externalDescStringOffsets]
	hash := view.sections[externalDescHash]
	symbols := view.sections[externalDescSymbolIndex]
	entries := view.sections[externalDescEntrySites]
	pcsites := view.sections[externalDescPCSites]
	// String IDs are uint32. The offset section and sidecar size bound their
	// count now, rather than the old uint16 ID limit.
	if records.count == 0 || records.count > 1<<20 || offsets.count == 0 ||
		stringsSec.count == 0 || stringsSec.count > 1<<30 || pclines.count > 1<<22 ||
		symbols.count > records.count || entries.count > records.count*16 || pcsites.count > 1<<24 {
		return false
	}
	// Validate every string ID before any runtime pointer is published.
	stringBase := externalSectionPtr(raw, stringsSec)
	offsetBase := externalSectionPtr(raw, offsets)
	// Offsets may reuse suffixes inside another string, so they need not point
	// immediately after a NUL. A trailing NUL and an in-bounds offset are
	// sufficient to guarantee termination without rescanning every suffix.
	if *(*byte)(unsafe.Add(stringBase, stringsSec.count-1)) != 0 {
		return false
	}
	for i := uintptr(0); i < offsets.count; i++ {
		off := uintptr(*(*uint32)(unsafe.Add(offsetBase, i*externalStringOffsetSize)))
		if off >= stringsSec.count {
			return false
		}
	}
	recordBase := externalSectionPtr(raw, records)
	for i := uintptr(0); i < records.count; i++ {
		rec := (*runtimeFuncInfoRecord)(unsafe.Add(recordBase, i*externalRecordSize))
		if uintptr(rec.symbolPkg) >= offsets.count || uintptr(rec.symbolName) >= offsets.count ||
			uintptr(rec.namePkg) >= offsets.count || uintptr(rec.nameName) >= offsets.count ||
			uintptr(rec.fileRoot) >= offsets.count || uintptr(rec.fileName) >= offsets.count {
			return false
		}
	}
	if hash.count != 0 && (hash.count&(hash.count-1) != 0 || hash.count > 1<<22) {
		return false
	}
	hashBase := externalSectionPtr(raw, hash)
	for i := uintptr(0); i < hash.count; i++ {
		idx := *(*uint16)(unsafe.Add(hashBase, i*externalHashSize))
		if uintptr(idx) > records.count {
			return false
		}
	}
	var prevSymbol uint64
	symbolBase := externalSectionPtr(raw, symbols)
	for i := uintptr(0); i < symbols.count; i++ {
		rec := (*runtimeFuncInfoSymbolIndexRecord)(unsafe.Add(symbolBase, i*externalSymbolIndexSize))
		if rec.symbolID == 0 || rec.funcIndex == 0 || uintptr(rec.funcIndex) > records.count || (i != 0 && rec.symbolID <= prevSymbol) {
			return false
		}
		prevSymbol = rec.symbolID
	}
	var prevPCLine uint64
	pclineBase := externalSectionPtr(raw, pclines)
	for i := uintptr(0); i < pclines.count; i++ {
		rec := (*runtimePCLineRecord)(unsafe.Add(pclineBase, i*externalPCLineSize))
		if rec.id == 0 || rec.funcIndex == 0 || uintptr(rec.funcIndex) > records.count ||
			(i != 0 && rec.id <= prevPCLine) || uintptr(rec.fileRoot) >= offsets.count || uintptr(rec.fileName) >= offsets.count {
			return false
		}
		prevPCLine = rec.id
	}
	if entries.count != 0 || pcsites.count != 0 {
		return false
	}

	runtimeFuncInfoTable = (*runtimeFuncInfoRecord)(recordBase)
	runtimeFuncInfoCount = records.count
	runtimeFuncInfoStrings = (*c.Char)(stringBase)
	runtimeFuncInfoStringOffsets = (*uint32)(offsetBase)
	runtimeFuncInfoStringCount = offsets.count
	runtimeFuncInfoHash = (*uint16)(hashBase)
	if hash.count != 0 {
		runtimeFuncInfoHashMask = hash.count - 1
	} else {
		runtimeFuncInfoHashMask = 0
	}
	runtimeFuncInfoSymbolIndex = (*runtimeFuncInfoSymbolIndexRecord)(symbolBase)
	runtimeFuncInfoSymbolIndexCount = symbols.count
	runtimePCLineTable = (*runtimePCLineRecord)(pclineBase)
	runtimePCLineCount = pclines.count

	return true
}

// Resolve translates an instrumentation ID only on an explicit symbolization
// request. Recording a frame performs neither file I/O nor string allocation.
func Resolve(id uint64) (name, file string, line, start int) {
	if id == 0 || !Ready() {
		return
	}
	lo, hi := uintptr(0), runtimePCLineCount
	for lo < hi {
		mid := (lo + hi) / 2
		rec := (*runtimePCLineRecord)(unsafe.Add(unsafe.Pointer(runtimePCLineTable), mid*externalPCLineSize))
		if rec.id < id {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= runtimePCLineCount {
		return
	}
	rec := (*runtimePCLineRecord)(unsafe.Add(unsafe.Pointer(runtimePCLineTable), lo*externalPCLineSize))
	if rec.id != id {
		return
	}
	fn := (*runtimeFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), uintptr(rec.funcIndex-1)*externalRecordSize))
	name = join(fn.namePkg, fn.nameName, ".")
	file = join(rec.fileRoot, rec.fileName, "")
	line, start = int(rec.line), int(fn.line&((1<<31)-1))
	return
}
func stringAt(id uint32) string {
	off := *(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStringOffsets), uintptr(id)*4))
	return c.GoString((*c.Char)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStrings), uintptr(off))))
}
func join(a, b uint32, sep string) string {
	left, right := stringAt(a), stringAt(b)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + sep + right
}

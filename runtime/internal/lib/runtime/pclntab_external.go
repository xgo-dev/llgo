//go:build llgo_pclntab_external && (linux || darwin) && (amd64 || arm64)

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	clitedebug "github.com/xgo-dev/llgo/runtime/internal/clite/debug"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

const (
	externalPCLNMagic      = "LLGOPCL1"
	externalPCLNVersion    = uint32(4)
	externalPCLNABIVersion = uint32(1)
	externalPCLNHeaderSize = uintptr(256)
	externalPCLNMaxSize    = int64(512 << 20)
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

//go:linkname runtimePCLNIdentity __llgo_pclntab_identity
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

func runtimePCLNReady() bool {
	return latomic.LoadUint32(&externalPCLNState) == externalPCLNLoaded
}

// ensureRuntimePCLN is called only by allocation/I/O-safe public
// symbolization entry points. Fatal-signal paths consult runtimePCLNReady and
// never enter this function. Success and failure are both terminal states.
func ensureRuntimePCLN() bool {
	for {
		switch latomic.LoadUint32(&externalPCLNState) {
		case externalPCLNLoaded:
			return true
		case externalPCLNUnavailable:
			return false
		case externalPCLNUnattempted:
			if !latomic.CompareAndSwapUint32(&externalPCLNState, externalPCLNUnattempted, externalPCLNLoading) {
				continue
			}
			if loadExternalPCLN() {
				latomic.StoreUint32(&externalPCLNState, externalPCLNLoaded)
				return true
			}
			latomic.StoreUint32(&externalPCLNState, externalPCLNUnavailable)
			return false
		default:
			// Only safe callers wait. A signal path observes !Ready and
			// immediately degrades without entering this loop.
			c.Usleep(1)
		}
	}
}

func loadExternalPCLN() bool {
	var info clitedebug.Info
	if clitedebug.Addrinfo(clitedebug.Address(), &info) == 0 || info.Fbase == nil {
		return false
	}
	path := externalPCLNExecutablePath()
	if path == "" && info.Fname != nil {
		path = c.GoString(info.Fname)
	}
	if path == "" {
		return false
	}
	path += ".pclntab"
	fd := cliteos.Open(c.AllocaCStr(path), cliteos.O_RDONLY)
	if fd < 0 {
		return false
	}
	defer cliteos.Close(fd)
	var st cliteos.StatT
	if cliteos.Fstat(fd, &st) != 0 || st.Size < int64(externalPCLNHeaderSize) || st.Size > externalPCLNMaxSize {
		return false
	}
	raw := make([]byte, int(st.Size))
	rawBase := unsafe.Pointer(&raw[0])
	rawLen := len(raw)
	for off := 0; off < rawLen; {
		read := cliteos.Read(fd, unsafe.Add(rawBase, uintptr(off)), uintptr(rawLen-off))
		if read <= 0 {
			return false
		}
		off += read
	}
	view, ok := validateExternalPCLN(raw, uintptr(info.Fbase))
	if !ok {
		return false
	}
	if !installExternalPCLN(raw, view, uintptr(info.Fbase)) {
		return false
	}
	externalPCLNBacking = raw
	// Complete all allocating table construction before publishing Loaded.
	// A signal racing the load sees !runtimePCLNReady and never observes the
	// partially installed globals.
	runtimeFuncPCInitState = runtimeFuncInfoInitBusy
	initRuntimeFuncPCFramesOnce()
	latomic.StoreUint32(&runtimeFuncPCInitState, runtimeFuncInfoInitDone)
	runtimePCLineInitState = runtimeFuncInfoInitBusy
	initRuntimePCLineFramesOnce()
	latomic.StoreUint32(&runtimePCLineInitState, runtimeFuncInfoInitDone)
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
		raw[externalHeaderPointerSize] != byte(unsafe.Sizeof(uintptr(0))) ||
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
	relocateSites := func(sec externalPCLNSection) bool {
		base := externalSectionPtr(raw, sec)
		for i := uintptr(0); i < sec.count; i++ {
			site := (*runtimeFuncInfoEntryRecord)(unsafe.Add(base, i*externalSiteSize))
			off := site.pc
			if site.symbolID == 0 || off < view.textStart-loadBase || off >= view.textEnd-loadBase || off > ^uintptr(0)-loadBase {
				return false
			}
			site.pc = loadBase + off
		}
		return true
	}
	if !relocateSites(entries) || !relocateSites(pcsites) {
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
	runtimeFuncInfoEntryStart = (*runtimeFuncInfoEntryRecord)(externalSectionPtr(raw, entries))
	if entries.count != 0 {
		runtimeFuncInfoEntryEnd = (*runtimeFuncInfoEntryRecord)(unsafe.Add(externalSectionPtr(raw, entries), entries.count*externalSiteSize))
	} else {
		runtimeFuncInfoEntryEnd = nil
	}
	runtimePCSiteStart = (*runtimePCSiteRecord)(externalSectionPtr(raw, pcsites))
	if pcsites.count != 0 {
		runtimePCSiteEnd = (*runtimePCSiteRecord)(unsafe.Add(externalSectionPtr(raw, pcsites), pcsites.count*externalSiteSize))
	} else {
		runtimePCSiteEnd = nil
	}
	return true
}

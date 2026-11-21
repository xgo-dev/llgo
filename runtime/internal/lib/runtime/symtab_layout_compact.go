//go:build llgo && llgo.pclntab.compact

package runtime

import "unsafe"

const (
	runtimeFuncInfoShortIDs = 1 << iota
	runtimeFuncInfoShortOffsets
)

// The synthetic entry object sets these bits for the linked metadata. Runtime
// archives in this profile can consume either width and share the same cache.
//
//go:linkname runtimeFuncInfoLayout C.__llgo_funcinfo_layout
var runtimeFuncInfoLayout uint8

type runtimeCompactFuncInfoRecord struct {
	symbolPkg, symbolName uint16
	namePkg, nameName     uint16
	fileRoot, fileName    uint16
	line                  uint32
}

// Reuse caller-owned scratch in table scans. The wide layout remains a
// direct view and caller-visible line flags stay 32-bit in either layout.
func funcInfoAt(index uintptr, scratch *runtimeFuncInfoRecord) *runtimeFuncInfoRecord {
	if runtimeFuncInfoLayout&runtimeFuncInfoShortIDs != 0 {
		rec := (*runtimeCompactFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), index*unsafe.Sizeof(runtimeCompactFuncInfoRecord{})))
		scratch.symbolPkg = uint32(rec.symbolPkg)
		scratch.symbolName = uint32(rec.symbolName)
		scratch.namePkg = uint32(rec.namePkg)
		scratch.nameName = uint32(rec.nameName)
		scratch.fileRoot = uint32(rec.fileRoot)
		scratch.fileName = uint32(rec.fileName)
		scratch.line = rec.line
		return scratch
	}
	return (*runtimeFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), index*unsafe.Sizeof(runtimeFuncInfoRecord{})))
}

func funcInfoStringOffset(index uintptr) uint32 {
	base := unsafe.Pointer(runtimeFuncInfoStringOffsets)
	if runtimeFuncInfoLayout&runtimeFuncInfoShortOffsets != 0 {
		return uint32(*(*uint16)(unsafe.Add(base, index*unsafe.Sizeof(uint16(0)))))
	}
	return *(*uint32)(unsafe.Add(base, index*unsafe.Sizeof(uint32(0))))
}

func funcInfoRecordSize() uintptr {
	if runtimeFuncInfoLayout&runtimeFuncInfoShortIDs != 0 {
		return unsafe.Sizeof(runtimeCompactFuncInfoRecord{})
	}
	return unsafe.Sizeof(runtimeFuncInfoRecord{})
}

func funcInfoStringOffsetSize() uintptr {
	if runtimeFuncInfoLayout&runtimeFuncInfoShortOffsets != 0 {
		return unsafe.Sizeof(uint16(0))
	}
	return unsafe.Sizeof(uint32(0))
}

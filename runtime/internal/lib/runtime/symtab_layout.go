//go:build !llgo || !llgo.pclntab.compact

package runtime

import "unsafe"

func funcInfoAt(index uintptr, _ *runtimeFuncInfoRecord) *runtimeFuncInfoRecord {
	return (*runtimeFuncInfoRecord)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoTable), index*unsafe.Sizeof(runtimeFuncInfoRecord{})))
}

func funcInfoStringOffset(index uintptr) uint32 {
	return *(*uint32)(unsafe.Add(unsafe.Pointer(runtimeFuncInfoStringOffsets), index*unsafe.Sizeof(uint32(0))))
}

func funcInfoRecordSize() uintptr {
	return unsafe.Sizeof(runtimeFuncInfoRecord{})
}

func funcInfoStringOffsetSize() uintptr {
	return unsafe.Sizeof(uint32(0))
}

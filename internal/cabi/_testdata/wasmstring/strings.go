package main

import "unsafe"

func validString(s *string, data uint64, length int) bool {
	return *(*uint64)(unsafe.Pointer(s)) == data && len(*s) == length
}

//export LocalStringStorage
func LocalStringStorage() bool {
	words := [2]uint64{0xa5a5a5a500001000, 7}
	s := *(*string)(unsafe.Pointer(&words))
	return words[0] == 0xa5a5a5a500001000 && validString(&s, 0x1000, 7)
}

//go:linkname foreignGlobal C.foreignGlobal
var foreignGlobal string

//export ImportedGlobal
func ImportedGlobal() bool {
	s := foreignGlobal
	return validString(&s, 0x6000, 19)
}

type SparseStorage struct {
	Data [4096]uint64
	Text string
}

//export CopySparseStorage
func CopySparseStorage(dst, src *SparseStorage) { *dst = *src }

//export ReceiveSparseStorage
func ReceiveSparseStorage(dst *SparseStorage, ch <-chan SparseStorage) { *dst = <-ch }

func main() {}

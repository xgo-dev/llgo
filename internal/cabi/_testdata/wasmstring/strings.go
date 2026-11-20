package main

import "unsafe"

func validString(s *string, data uint64, length int) bool {
	return *(*uint64)(unsafe.Pointer(s)) == data && len(*s) == length
}

//export LocalStringStorage
func LocalStringStorage() bool {
	words := [2]uint64{0xa5a5a5a500001000, 7}
	s := *(*string)(unsafe.Pointer(&words))
	return words[0] == 0xa5a5a5a500001000 && uintptr(unsafe.Pointer(unsafe.StringData(s))) == 0x1000 && len(s) == 7
}

//go:linkname foreignGlobal C.foreignGlobal
var foreignGlobal string

//export ImportedGlobal
func ImportedGlobal() bool {
	s := foreignGlobal
	return uintptr(unsafe.Pointer(unsafe.StringData(s))) == 0x6000 && len(s) == 19
}

func main() {}

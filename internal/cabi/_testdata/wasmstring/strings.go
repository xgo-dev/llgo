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

//export CheckSparseStorage
func CheckSparseStorage() bool {
	var value SparseStorage
	value.Data[0], value.Data[len(value.Data)-1] = 11, 29
	*(*[2]uint64)(unsafe.Pointer(&value.Text)) = [2]uint64{0xa5a5a5a500001000, 7}
	CopySparseStorage(&value, &value)
	return value.Data[0] == 11 && value.Data[len(value.Data)-1] == 29 && validString(&value.Text, 0x1000, 7)
}

//export ReceiveSparseStorage
func ReceiveSparseStorage(dst *SparseStorage, ch <-chan SparseStorage) { *dst = <-ch }

// This array is below the 4 KiB struct threshold but still needs copy lowering.
type SparseArrayStorage [2]struct {
	Data [250]uint64
	Text string
}

//export CopySparseArrayStorage
func CopySparseArrayStorage(dst, src *SparseArrayStorage) { *dst = *src }

//export ReceiveSparseArrayStorage
func ReceiveSparseArrayStorage(dst *SparseArrayStorage, ch <-chan SparseArrayStorage) { *dst = <-ch }

var sparseArrayValue SparseArrayStorage

//export CheckSparseArrayStorage
func CheckSparseArrayStorage() bool {
	value := &sparseArrayValue
	value[0].Data[0], value[1].Data[249] = 11, 29
	*(*[2]uint64)(unsafe.Pointer(&value[0].Text)) = [2]uint64{0xa5a5a5a500001000, 7}
	*(*[2]uint64)(unsafe.Pointer(&value[1].Text)) = [2]uint64{0xa5a5a5a500002000, 9}
	CopySparseArrayStorage(value, value)
	return value[0].Data[0] == 11 && value[1].Data[249] == 29 &&
		validString(&value[0].Text, 0x1000, 7) && validString(&value[1].Text, 0x2000, 9)
}

func main() {}

package main

import "unsafe"

type Lookalike struct {
	Data *byte
	Len  int64
}

type Envelope struct {
	Prefix uint64
	Text   string
	Bytes  []byte
	Record Lookalike
}

// Observe storage through pointers: a by-value string or Envelope parameter
// would itself be normalized on entry and could hide a broken boundary.
func validString(s *string, data uint64, length int) bool {
	return *(*uint64)(unsafe.Pointer(s)) == data && len(*s) == length
}

func validEnvelope(v *Envelope, prefix uint64) bool {
	return v.Prefix == prefix &&
		*(*uint64)(unsafe.Pointer(&v.Text)) == 0x1000 && len(v.Text) == 7 &&
		*(*uint64)(unsafe.Pointer(&v.Bytes)) == 0x5000 && len(v.Bytes) == 3 && cap(v.Bytes) == 8 &&
		*(*uint64)(unsafe.Pointer(&v.Record)) == 0xdeadbeef00003000 &&
		v.Record.Len == 23
}

//go:linkname foreignString C.foreignString
func foreignString() string

//go:linkname foreignStringAlternate C.foreignStringAlternate
func foreignStringAlternate() string

//go:linkname foreignEnvelope C.foreignEnvelope
func foreignEnvelope() Envelope

//go:linkname foreignEnvelopeAlternate C.foreignEnvelopeAlternate
func foreignEnvelopeAlternate() Envelope

//export ExportedString
func ExportedString(s string) bool { return validString(&s, 0x1000, 7) }

//export ExportedEnvelope
func ExportedEnvelope(v Envelope) bool { return validEnvelope(&v, 11) }

//export ImportedString
func ImportedString() bool {
	s := foreignString()
	return validString(&s, 0x1000, 7)
}

//export ImportedEnvelope
func ImportedEnvelope() bool {
	v := foreignEnvelope()
	return validEnvelope(&v, 11)
}

// Runtime selection forces a Go funcval call whose environment must retain
// the selected native function pointer and whose result must retain its data.
//
//export ImportedStringFuncval
func ImportedStringFuncval(alternate bool) bool {
	f := foreignString
	data, length := uint64(0x1000), 7
	if alternate {
		f = foreignStringAlternate
		data, length = 0x4000, 17
	}
	s := f()
	return validString(&s, data, length)
}

//export ImportedEnvelopeFuncval
func ImportedEnvelopeFuncval(alternate bool) bool {
	f := foreignEnvelope
	prefix := uint64(11)
	if alternate {
		f = foreignEnvelopeAlternate
		prefix = 37
	}
	v := f()
	return validEnvelope(&v, prefix)
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

//go:linkname takeHeaderArray C.takeHeaderArray
func takeHeaderArray([256]string)

//go:linkname takeLargeHeaderArray C.takeLargeHeaderArray
func takeLargeHeaderArray([4097]string)

//go:linkname mutateHeaderArray C.mutateHeaderArray
func mutateHeaderArray(*[256]string)

//export PassHeaderArray
func PassHeaderArray(src *[256]string) { takeHeaderArray(*src) }

//export PassLargeHeaderArray
func PassLargeHeaderArray(src *[4097]string) { takeLargeHeaderArray(*src) }

//export ReturnHeaderArray
func ReturnHeaderArray(src *[4097]string) [4097]string { return *src }

//export PassHeaderArrayAfterMutation
func PassHeaderArrayAfterMutation(src *[256]string) {
	value := *src
	mutateHeaderArray(src)
	takeHeaderArray(value)
}

func main() {}

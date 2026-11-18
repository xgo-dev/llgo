//go:build llgo

package main

import "unsafe"

//llgo:cdecl
type nativeRecord struct {
	Data unsafe.Pointer
	Size uintptr
	Tag  int32
}

//llgo:cdecl
type recordCallback func(nativeRecord) nativeRecord

//llgo:cdecl
type nativeWords [2]uintptr

//go:linkname cWords C.llgo_wasm_profile_words
func cWords(*nativeWords) int32

//go:linkname cRecordSize C.llgo_wasm_profile_record_size
func cRecordSize() uintptr

//go:linkname cRecordOffset C.llgo_wasm_profile_record_offset
func cRecordOffset() uintptr

//go:linkname cRecordRoundTrip C.llgo_wasm_profile_record_round_trip
func cRecordRoundTrip(nativeRecord, recordCallback) nativeRecord

//go:linkname cSignedWord C.llgo_wasm_profile_signed_word
func cSignedWord(nativeSignedWord) nativeSignedWord

func updateNativeRecord(r nativeRecord) nativeRecord {
	if r.Size != 7 || r.Tag != -3 || *(*byte)(r.Data) != 41 {
		panic("C-to-Go aggregate arguments corrupted")
	}
	*(*byte)(r.Data) = 42
	r.Size++
	r.Tag--
	return r
}

func checkNativeRecord() {
	words := nativeWords{7, 9}
	if unsafe.Sizeof(words) != 2*expectedCPointerSize || cWords(&words) != 1 || words[1] != 11 {
		panic("native array layout disagrees with C")
	}
	if cRecordSize() != unsafe.Sizeof(nativeRecord{}) || cRecordOffset() != unsafe.Offsetof(nativeRecord{}.Size) {
		println("native layout", cRecordSize(), unsafe.Sizeof(nativeRecord{}), cRecordOffset(), unsafe.Offsetof(nativeRecord{}.Size))
		panic("native aggregate layout disagrees with C")
	}
	b := byte(41)
	r := cRecordRoundTrip(nativeRecord{Data: unsafe.Pointer(&b), Size: 7, Tag: -3}, updateNativeRecord)
	if r.Data != unsafe.Pointer(&b) || b != 42 || r.Size != 8 || r.Tag != -4 {
		panic("Go-to-C aggregate return corrupted")
	}
	if cSignedWord(-17) != -17 {
		panic("signed C word round trip failed")
	}
}

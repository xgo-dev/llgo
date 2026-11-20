//go:build llgo && wasm

package wasmexportabi

import "unsafe"

const LLGoFiles = "_wrap/caller.c"

//export llgo_test_signed_word
func signedWord(n int) int { return n - 1 }

//export llgo_test_unsigned_word
func unsignedWord(n uint) uint { return n + 1 }

//export llgo_test_go_slice
func goSlice(s []byte) []byte {
	if len(s) != 3 || cap(s) != 5 || !slicePointerSlotOK(&s) || s[0] != 7 {
		return nil
	}
	return s[1:4]
}

//go:noinline
func slicePointerSlotOK(s *[]byte) bool {
	// Read the whole Go storage word as well as the actual pointer. On
	// Memory32, incoming C padding must never become the high half of the word.
	return *(*uintptr)(unsafe.Pointer(s)) == uintptr(unsafe.Pointer(unsafe.SliceData(*s)))
}

//go:linkname callGoExports C.llgo_test_call_go_exports
func callGoExports() int32

//go:linkname cSlice C.llgo_test_c_slice
func cSlice([]byte) []byte

// Mixed exercises a slice nested inside a regular Go struct.
type Mixed struct {
	Text string
	Data []byte
}

//export llgo_test_go_mixed
func goMixed(m Mixed) Mixed {
	if m.Text != "wasm" || !slicePointerSlotOK(&m.Data) {
		return Mixed{}
	}
	return Mixed{Text: m.Text, Data: goSlice(m.Data)}
}

//go:linkname callGoMixed C.llgo_test_call_go_mixed
func callGoMixed() int32

//go:linkname cMixed C.llgo_test_c_mixed
func cMixed(Mixed) Mixed

//go:noinline
func callSliceFunc(fn func([]byte) []byte, s []byte) []byte {
	return fn(s)
}

//llgo:cdecl
type sliceSink func([]byte)

//llgo:cdecl
type sliceSource func() []byte

//go:linkname cCallSink C.llgo_test_call_sink
func cCallSink(sliceSink)

//go:linkname cCallSource C.llgo_test_call_source
func cCallSource(sliceSource) int32

var callbackByte byte
var callbackData = [5]byte{7, 8, 9, 10, 11}

func readFirst(s []byte) {
	if len(s) == 3 && cap(s) == 5 && slicePointerSlotOK(&s) {
		callbackByte = s[0]
	}
}

func readSecond(s []byte) {
	if len(s) == 3 && cap(s) == 5 && slicePointerSlotOK(&s) {
		callbackByte = s[1]
	}
}

func firstSlice() []byte  { return callbackData[:3] }
func secondSlice() []byte { return callbackData[1:4] }

//go:noinline
func callCallbacks(second bool) bool {
	sink, source := readFirst, firstSlice
	if second {
		sink, source = readSecond, secondSlice
	}
	callbackByte = 0
	cCallSink(sliceSink(sink))
	want := byte(7)
	if second {
		want = 8
	}
	return callbackByte == want && cCallSource(sliceSource(source)) == int32(want)
}

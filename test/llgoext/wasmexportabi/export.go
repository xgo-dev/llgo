//go:build llgo && wasm

package wasmexportabi

import (
	"structs"
	"unsafe"

	"github.com/xgo-dev/llgo/test/llgoext/wasmexportabi/callback"
)

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

//go:noinline
func stringPointerSlotOK(s *string) bool {
	return *(*uintptr)(unsafe.Pointer(s)) == uintptr(unsafe.Pointer(unsafe.StringData(*s)))
}

//go:linkname callGoExports C.llgo_test_call_go_exports
func callGoExports() int32

//go:linkname cSlice C.llgo_test_c_slice
func cSlice([]byte) []byte

// Mixed exercises a string and a slice nested inside a regular Go struct.
type Mixed struct {
	Text string
	Data []byte
}

type hostRecord struct {
	_ structs.HostLayout
	Mixed
	Word uintptr
	Rows [1][2]Mixed
}

//go:linkname hostGlobal C.llgo_test_host_global
var hostGlobal hostRecord

//go:linkname plainGlobal C.llgo_test_plain_global
var plainGlobal struct {
	Mixed
	Word uintptr
}

//go:linkname dirtyHostHeaders C.llgo_test_dirty_host_headers
func dirtyHostHeaders() *string

//export llgo_test_go_mixed
func goMixed(m Mixed) Mixed {
	if m.Text != "wasm" || !stringPointerSlotOK(&m.Text) || !slicePointerSlotOK(&m.Data) {
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

type largeText struct{ text [4097]string }

//go:linkname cLargeResult C.llgo_test_large_result
func cLargeResult() largeText

//export llgo_test_go_large_param
func goLargeParam(arg largeText) int32 {
	for sample, want := range [...]string{"head", "body", "tail"} {
		i := sample * (len(arg.text) - 1) / 2
		if arg.text[i] != want || !stringPointerSlotOK(&arg.text[i]) {
			return int32(i + 1)
		}
	}
	return 0
}

//go:linkname callLargeParam C.llgo_test_call_large_param
func callLargeParam() int32

type projectionSlices struct{ Data [64][]byte }
type projectionStrings struct{ Data [64]string }

//export llgo_test_slice_len
func projectionSliceLen(p projectionSlices) int { return len(p.Data[0]) }

//export llgo_test_slice_cap
func projectionSliceCap(p projectionSlices) int { return cap(p.Data[63]) }

//export llgo_test_string_len
func projectionStringLen(p projectionStrings) int { return len(p.Data[0]) }

//go:linkname cProjectionSlices C.llgo_test_projection_slices
func cProjectionSlices() projectionSlices

//go:linkname cProjectionStrings C.llgo_test_projection_strings
func cProjectionStrings() projectionStrings

//export llgo_test_result_slice_len
func resultSliceLen() int { return len(cProjectionSlices().Data[0]) }

//export llgo_test_result_slice_cap
func resultSliceCap() int { return cap(cProjectionSlices().Data[63]) }

//export llgo_test_result_string_len
func resultStringLen() int { return len(cProjectionStrings().Data[0]) }

//export llgo_test_projection_slice_slot
func projectionSliceSlot(p projectionSlices) uint64 {
	return uint64(*(*uintptr)(unsafe.Pointer(&p.Data[0])))
}

//export llgo_test_projection_result_slot
func projectionResultSlot() uint64 {
	p := cProjectionStrings()
	return uint64(*(*uintptr)(unsafe.Pointer(&p.Data[0])))
}

//go:linkname callHeaderProjections C.llgo_test_call_header_projections
func callHeaderProjections() int32

//llgo:cdecl
type stringReader func(string) int32

//go:linkname cReadString C.llgo_test_read_string
func cReadString(stringReader) int32

func callbackStringByte(s *string, i int) int32 {
	if !stringPointerSlotOK(s) || *s != "wasm" {
		return -1
	}
	return int32((*s)[i])
}

//export llgo_test_go_string
func firstStringByte(s string) int32 { return callbackStringByte(&s, 0) }
func lastStringByte(s string) int32  { return callbackStringByte(&s, 3) }

//go:noinline
func callStringReader(last bool) int32 {
	read := firstStringByte
	if last {
		read = lastStringByte
	}
	return cReadString(stringReader(read))
}

//go:linkname cReadSource C.llgo_test_read_source
func cReadSource(callback.StringSource) int32

//go:noinline
func callGoStringSource(f func() string) int32 { return cReadSource(callback.StringSource(f)) }

//go:linkname cStringSource C.llgo_test_string_source
func cStringSource(second int32) callback.StringSource

//go:linkname cSliceSource C.llgo_test_slice_source
func cSliceSource(second int32) callback.SliceSource

func roundTripSources(second int32) bool {
	return callback.MatchSources(second, cStringSource(second), cSliceSource(second))
}

//go:linkname genericSource C.llgo_test_generic_source
func genericSource(callback.Box[func(left byte)]) string

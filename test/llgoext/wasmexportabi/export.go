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
	if len(s) != 3 || cap(s) != 5 || !slicePointerSlotOK(s) || s[0] != 7 {
		return nil
	}
	return s[1:4]
}

//go:noinline
func slicePointerSlotOK(s []byte) bool {
	// Read the whole Go storage word as well as the actual pointer. On
	// Memory32, incoming C padding must never become the high half of the word.
	return *(*uintptr)(unsafe.Pointer(&s)) == uintptr(unsafe.Pointer(unsafe.SliceData(s)))
}

//go:linkname callGoExports C.llgo_test_call_go_exports
func callGoExports() int32

//go:linkname cSlice C.llgo_test_c_slice
func cSlice([]byte) []byte

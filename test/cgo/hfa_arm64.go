//go:build llgo && arm64 && (linux || darwin || windows)

package cgo

/*
#include "hfa_arm64.h"
*/
import "C"

import "unsafe"

//llgo:cdecl
type hfaPairEntry func(float64, float64, float64, float64, float64, float64, float64, float64, C.llgo_hfa_pair, float64) float64

//go:linkname linkedHFAPair C.llgo_test_hfa_pair
func linkedHFAPair(float64, float64, float64, float64, float64, float64, float64, float64, C.llgo_hfa_pair, float64) float64

func directHFAPair() float64 {
	return float64(C.llgo_test_hfa_pair(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_hfa_pair{first: 1, second: 2}, 3))
}
func indirectHFAPair() float64 {
	address := C.llgo_hfa_pair_address()
	function := *(*hfaPairEntry)(unsafe.Pointer(&address))
	return function(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_hfa_pair{first: 1, second: 2}, 3)
}
func linkedHFAPairCall() float64 {
	return linkedHFAPair(1, 2, 3, 4, 5, 6, 7, 8, C.llgo_hfa_pair{first: 1, second: 2}, 3)
}
func partialHFAPair() float64 {
	return float64(C.llgo_test_hfa_partial(1, 2, 3, 4, 5, 6, 7, C.llgo_hfa_pair{first: 6, second: 7}, 1))
}
func stackedHFATriple() float64 {
	return float64(C.llgo_test_hfa_triple(1, 2, 3, 4, 5, 6, 7, 8, 1,
		C.llgo_hfa_triple{first: 1, rest: [2]C.float{1, 1}}, 2))
}
func stackedHFAQuad() float64 {
	return float64(C.llgo_test_hfa_quad(1, 2, 3, 4, 5, 6, 7, 8,
		C.llgo_hfa_quad{values: [4]C.double{1, 1, 1, 1}}, 2))
}
func exportedHFA() float64 { return float64(C.llgo_hfa_export_call()) }

//export GoHFAStack
func GoHFAStack(a, b, c, d, e, f, g, h C.double, gap C.float, triple C.llgo_hfa_triple, tail C.double) C.double {
	return a + b + c + d + e + f + g + h + C.double(gap+triple.first+triple.rest[0]+triple.rest[1]) + tail
}

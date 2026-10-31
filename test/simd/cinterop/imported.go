//go:build cgo && (amd64 || arm64)

package cinterop

/*
#include "wrap/vector.h"
*/
import "C"

type cFloat = C.float
type cFloat4 = C.llgo_f4
type cFloat4Alias = C.llgo_f4_alias
type cDouble2 = C.llgo_d2
type cUint4 = C.llgo_u4
type cBytes16 = C.llgo_b16
type cByte = C.uint8_t
type cPair = C.llgo_vpair
type cMixed = C.llgo_vmixed
type cTriple = C.llgo_vtriple
type cBox = C.llgo_vbox
type cPlainBox = C.llgo_plain_box

func makeFloat4(a, b, c, d float32) cFloat4 {
	return C.llgo_vmake(C.float(a), C.float(b), C.float(c), C.float(d))
}
func addFloat4(a, b cFloat4) cFloat4          { return C.llgo_vadd(a, b) }
func xorUint4(a, b cUint4) cUint4             { return C.llgo_vxor(a, b) }
func addBytes16(a cBytes16) cBytes16          { return C.llgo_vbytes(a) }
func subDouble2(a, b cDouble2) cDouble2       { return C.llgo_vdouble(a, b) }
func aliasFloat4(a cFloat4Alias) cFloat4Alias { return C.llgo_valias(a) }
func laneFloat4(a cFloat4, i int) float32     { return float32(C.llgo_vlane(a, C.int(i))) }
func sumPair(a cPair) float32                 { return float32(C.llgo_pair_sum(a)) }
func swapPair(a cPair) cPair                  { return C.llgo_pair_swap(a) }
func changeMixed(a cMixed) cMixed             { return C.llgo_mixed_swap(a) }
func rotateTriple(a cTriple) cTriple          { return C.llgo_triple_rotate(a) }
func readBox(a *cBox) float32                 { return float32(C.llgo_box_read(a)) }
func changeBox(a cBox) cBox                   { return C.llgo_box_value(a) }
func sumPlain(a cPlainBox) float32            { return float32(C.llgo_plain_value(a)) }
func spillPair(a, b cFloat4) float32 {
	return float32(C.llgo_pair_spill(a, b, a, b, a, b, a, cPair{a: a, b: b}, b))
}
func manyVectors(a, b cFloat4) cFloat4 { return C.llgo_many_vectors(a, b, a, b, a, b, a, b, a, b) }
func nativeBoxLayout() uint64          { return uint64(C.llgo_box_layout()) }
func globalFloat4() cFloat4            { return C.llgo_vglobal() }

//go:noinline
func bytesFromSlice(s []cByte) cBytes16 { return cBytes16(s) }

//go:noinline
func byteVectorPointer(s []cByte) *cBytes16 { return (*cBytes16)(s) }

//go:noinline
func loadByteVector(p *cBytes16) cBytes16 { return *p }

//go:noinline
func storeByteVector(p *cBytes16, v cBytes16) { *p = v }

//go:noinline
func floatsFromSlice(s []cFloat) cFloat4 { return cFloat4(s) }

//go:noinline
func floatVectorPointer(s []cFloat) *cFloat4 { return (*cFloat4)(s) }

//go:noinline
func loadFloatVector(p *cFloat4) cFloat4 { return *p }

//go:noinline
func storeFloatVector(p *cFloat4, v cFloat4) { *p = v }

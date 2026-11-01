package main

/*
typedef unsigned char B16 __attribute__((vector_size(16)));
typedef float F4 __attribute__((vector_size(16)));
*/
import "C"

var Owned C.B16

//go:noinline
func FromBytes(s []C.uchar) C.B16 { return C.B16(s) }

//go:noinline
func FromBytePointer(s []C.uchar) C.B16 { return *(*C.B16)(s) }

//go:noinline
func LoadBytes(p *C.B16) C.B16 { return *p }

//go:noinline
func StoreBytes(p *C.B16, v C.B16) { *p = v }

//go:noinline
func FromFloats(s []C.float) C.F4 { return C.F4(s) }

//go:noinline
func LoadFloats(p *C.F4) C.F4 { return *p }

//go:noinline
func StoreFloats(p *C.F4, v C.F4) { *p = v }

func main() {}

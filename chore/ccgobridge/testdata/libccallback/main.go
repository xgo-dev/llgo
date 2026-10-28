package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"modernc.org/libc"
)

// comparePC returns the assembly entry, not a Go funcval box. With Go this is
// physical ABI0; with LLGo both producer and consumer use its translated ABI.
func comparePC() uintptr

//go:noescape
func compare(tls *libc.TLS, left, right uintptr) int32

// The explicit assembly bridge preserves the caller's uintptr signature, then
// passes the identical ABI0 words to a pointer-typed implementation. Merely
// casting an untyped integer entry to an incompatible function is not the test.
//
//go:noescape
func typedCompare(tls *libc.TLS, left, right *int64) int32

func main() {
	tls := libc.NewTLS()
	defer tls.Close()
	for attempt := 0; attempt < 20; attempt++ {
		values := []int64{31, -2, 7, 0, 7, -44}
		libc.Yqsort(tls, uintptr(unsafe.Pointer(&values[0])), libc.Tsize_t(len(values)), 8, comparePC())
		for i, want := range []int64{-44, -2, 0, 7, 7, 31} {
			if values[i] != want {
				panic(fmt.Sprintf("qsort[%d] = %d, want %d", i, values[i], want))
			}
		}
		if tls.StackSlots() != 0 {
			panic("ABI0 callback leaked a TLS frame")
		}
		runtime.KeepAlive(values)
		runtime.GC()
	}
	fmt.Println("libc Yqsort ABI0 callback and TLS reuse verified")
}

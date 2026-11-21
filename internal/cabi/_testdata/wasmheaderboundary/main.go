package main

import "unsafe"

type Result[T any] struct{ Rows T }

//go:linkname stackResult C.stackResult
func stackResult() Result[[9]string]

//go:linkname heapResult C.heapResult
func heapResult() Result[[256]string]

//go:linkname largeResult C.largeResult
func largeResult() Result[[32768]string]

// Inspect the entire data slot, including the padding supplied by C.
//
//export ReadStackResult
func ReadStackResult() uintptr {
	v := stackResult()
	return *(*uintptr)(unsafe.Pointer(&v.Rows[0]))
}

//export ReadHeapResult
func ReadHeapResult() uintptr {
	v := heapResult()
	return *(*uintptr)(unsafe.Pointer(&v.Rows[0]))
}

//export ReadLargeResult
func ReadLargeResult() uintptr {
	v := largeResult()
	return *(*uintptr)(unsafe.Pointer(&v.Rows[0]))
}

//go:noinline
func readLargeParam(v Result[[32768]string]) uintptr { return *(*uintptr)(unsafe.Pointer(&v.Rows[0])) }

//export ReadNestedParam
func ReadNestedParam(v [4][4][4][]byte) uintptr { return *(*uintptr)(unsafe.Pointer(&v[0][0][0])) }

func main() { println(readLargeParam(Result[[32768]string]{})) }

//export DiscardResult
func DiscardResult() { heapResult() }

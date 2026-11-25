package main

import _ "unsafe"

type Sparse struct {
	Data [8192]uint64
	Text string
}

//go:linkname sparseResult C.sparseResult
func sparseResult() Sparse

//export CopySparseResult
func CopySparseResult(dst *Sparse) { *dst = sparseResult() }

//export CopySparseParam
func CopySparseParam(dst *Sparse, src Sparse) { *dst = src }

// This 4032-byte array is below the struct-copy threshold, but still uses
// memory copies because a multi-element array is too large for SSA.
type SparseArray [2]struct {
	Data [250]uint64
	Text string
}

//go:linkname sparseArrayResult C.sparseArrayResult
func sparseArrayResult() SparseArray

//export CopySparseArrayResult
func CopySparseArrayResult(dst *SparseArray) { *dst = sparseArrayResult() }

//export CopySparseArrayParam
func CopySparseArrayParam(dst *SparseArray, src SparseArray) { *dst = src }

type Wrapped struct {
	Rows SparseArray
	Tag  uint32
}

//go:linkname wrappedResult C.wrappedResult
func wrappedResult() Wrapped

//export CopyWrappedResult
func CopyWrappedResult(dst *SparseArray) { *dst = wrappedResult().Rows }

//export CopyWrappedParam
func CopyWrappedParam(dst *SparseArray, src Wrapped) { *dst = src.Rows }

func main() {}

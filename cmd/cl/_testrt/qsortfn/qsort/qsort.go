package qsort

import (
	"unsafe"
)

const (
	LLGoPackage = "decl"
)

// llgo:cdecl
type Comp func(a, b unsafe.Pointer) int32

//go:linkname Qsort C.qsort
func Qsort(base unsafe.Pointer, count, elem uintptr, compar Comp)

//go:linkname Qsort2 C.qsort
func Qsort2(base unsafe.Pointer, count, elem uintptr, compar func(a, b unsafe.Pointer) int32)

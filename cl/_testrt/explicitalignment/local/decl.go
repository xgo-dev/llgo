package local

import (
	"structs"
	_ "unsafe"
)

const (
	LLGoFiles   = "_wrap/alignment.c"
	LLGoPackage = "link"
)

type Record struct {
	_     structs.HostLayout
	_     structs.Align64
	Tag   byte
	Value uint64
	Tail  uint32
}

//go:linkname Check C.llgo_explicit_alignment_check
func Check(p *Record, size, alignment, valueOffset, tailOffset uintptr) int32

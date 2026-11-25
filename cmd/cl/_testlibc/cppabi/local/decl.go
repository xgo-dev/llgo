package local

import _ "unsafe"

const (
	LLGoFiles   = "_wrap/bridge.cpp"
	LLGoPackage = "link"
)

//go:linkname Add C.llgo_test_cpp_add
func Add(left, right int64) int64

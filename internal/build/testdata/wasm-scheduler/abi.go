package main

import _ "unsafe"

const LLGoFiles = "abi.c"

//go:linkname cLongSize C.llgo_test_sizeof_long
func cLongSize() uintptr

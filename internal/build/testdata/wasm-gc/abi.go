package main

import _ "unsafe"

const LLGoFiles = "abi.c"

//go:linkname testAlignedAlloc C.llgo_test_gc_aligned_alloc
func testAlignedAlloc() int32

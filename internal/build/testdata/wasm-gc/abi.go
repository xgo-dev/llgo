package main

import _ "unsafe"

const LLGoFiles = "abi.c"

//go:linkname testAlignedAlloc C.llgo_test_gc_aligned_alloc
func testAlignedAlloc() int32

//go:linkname testSetCGlobalRoot C.llgo_test_gc_set_c_root
func testSetCGlobalRoot()

//go:linkname testReadCGlobalRoot C.llgo_test_gc_read_c_root
func testReadCGlobalRoot() uint64

//go:linkname testClearCGlobalRoot C.llgo_test_gc_clear_c_root
func testClearCGlobalRoot()

//go:linkname testCGlobalRootWordOffset C.llgo_test_gc_c_root_word_offset
func testCGlobalRootWordOffset() uint32

//go:linkname testCPointerSize C.llgo_test_gc_c_pointer_size
func testCPointerSize() uint32

//go:linkname testClobberCStack C.llgo_test_gc_clobber_c_stack
func testClobberCStack(depth uint32)

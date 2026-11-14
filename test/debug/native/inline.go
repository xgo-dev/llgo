package main

import _ "unsafe"

const LLGoFiles = "_wrap/opaque.c"

// A volatile host side effect keeps the leaf's call alive even under full
// LTO. A write-only Go global can legally disappear with its entire call.
//go:linkname inlineOpaque C.llgo_debug_inline_opaque
func inlineOpaque(value int) int

func inlineLeaf(value int) int {
	adjusted := inlineOpaque(value) + 2 // LLDB_STOP: inline_leaf
	return adjusted
}

func inlineMiddle(value int) int {
	doubled := inlineLeaf(value) * 2 // LLDB_STOP: inline_middle
	return doubled
}

//go:noinline
func optimizedInlineCaller(value int) int {
	result := inlineMiddle(value) + 1 // LLDB_STOP: inline_caller
	result++                          // LLDB_STOP: inline_after_call
	return result
}

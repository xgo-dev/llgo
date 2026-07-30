package main

import _ "unsafe"

const LLGoFiles = "abi.c"

//go:linkname cLongSize C.llgo_test_sizeof_long
func cLongSize() uintptr

//go:linkname schedulerDeadlockMode C.llgo_test_scheduler_deadlock
func schedulerDeadlockMode() int32

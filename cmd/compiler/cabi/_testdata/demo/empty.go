package main

import "unsafe"

const (
	LLGoFiles = "../wrap/empty.c"
)

//go:linkname printf C.printf
func printf(format *byte, __llgo_va_list ...any) int32

func assert(info string, b bool) {
	if !b {
		printf(unsafe.StringData("Assertion failed: %s\n\000"), unsafe.StringData(info))
	}
}

func main() {}

type empty struct {
}

//go:linkname cdemo0 C.demo0
func cdemo0(empty) empty

func demo0(a empty) empty {
	return a
}

func init() {
	assert("cdemo0", cdemo0(empty{}) == empty{})
	assert("demo0", demo0(empty{}) == empty{})
}

//go:linkname cdemo1 C.demo1
func cdemo1(empty, int32) empty

func demo1(a empty, b int32) empty {
	return a
}

func init() {
	assert("cdemo1", cdemo1(empty{}, 1) == empty{})
	assert("demo1", demo1(empty{}, 2) == empty{})
}

//go:linkname cdemo2 C.demo2
func cdemo2(int32, empty) int32

func demo2(v int32, a empty) int32 {
	return v
}

func init() {
	assert("cdemo2", cdemo2(100, empty{}) == 100)
	assert("demo2", demo2(100, empty{}) == 100)
}

//go:linkname cdemo3 C.demo3
func cdemo3(int32, empty, int32) int32

func demo3(v int32, a empty, v2 int32) int32 {
	return v + v2
}

func init() {
	assert("cdemo3", cdemo3(1, empty{}, 100) == 101)
	assert("demo3", demo3(1, empty{}, 100) == 101)
}

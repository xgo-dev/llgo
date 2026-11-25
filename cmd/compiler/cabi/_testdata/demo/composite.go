package main

import "unsafe"

const (
	LLGoFiles = "../wrap/composite.c"
)

//go:linkname printf C.printf
func printf(format *byte, __llgo_va_list ...any) int32

func assert(info string, b bool) {
	if !b {
		printf(unsafe.StringData("Assertion failed: %s\n\000"), unsafe.StringData(info))
	}
}

func main() {
}

type basearray1 struct {
	x [1]int32
}

type array10 struct {
	base basearray1
}

//go:linkname cdemo_array10 C.demo_array10
func cdemo_array10(a array10) array10

func demo_array10(a array10) array10 {
	return a
}

func init() {
	v := array10{basearray1{[1]int32{1}}}
	assert("cdemo_array10\000", cdemo_array10(v) == v)
	assert("demo_array10\000", demo_array10(v) == v)
}

type array11 struct {
	base basearray1
	z    int32
}

//go:linkname cdemo_array11 C.demo_array11
func cdemo_array11(a array11) array11

func demo_array11(a array11) array11 {
	return a
}

func init() {
	v := array11{basearray1{[1]int32{1}}, 3}
	assert("cdemo_array11\000", cdemo_array11(v) == v)
	assert("demo_array11\000", demo_array11(v) == v)
}

type basepoint1 struct {
	x int32
}

type point10 struct {
	base basepoint1
}

//go:linkname cdemo_point10 C.demo_point10
func cdemo_point10(a point10) point10

func demo_point10(a point10) point10 {
	return a
}

func init() {
	v := point10{basepoint1{1}}
	assert("cdemo_point10\000", cdemo_point10(v) == v)
	assert("demo_point10\000", demo_point10(v) == v)
}

type point11 struct {
	base basepoint1
	z    int32
}

//go:linkname cdemo_point11 C.demo_point11
func cdemo_point11(a point11) point11

func demo_point11(a point11) point11 {
	return a
}

func init() {
	v := point11{basepoint1{1}, 3}
	assert("cdemo_point11\000", cdemo_point11(v) == v)
	assert("demo_point11\000", demo_point11(v) == v)
}

type basearray2 struct {
	x [2]int32
}

type array20 struct {
	base basearray2
}

//go:linkname cdemo_array20 C.demo_array20
func cdemo_array20(a array20) array20

func demo_array20(a array20) array20 {
	return a
}

func init() {
	v := array20{basearray2{[2]int32{1, 2}}}
	assert("cdemo_array20\000", cdemo_array20(v) == v)
	assert("demo_array20\000", demo_array20(v) == v)
}

type array21 struct {
	base basearray2
	z    int32
}

//go:linkname cdemo_array21 C.demo_array21
func cdemo_array21(a array21) array21

func demo_array21(a array21) array21 {
	return a
}

func init() {
	v := array21{basearray2{[2]int32{1, 2}}, 3}
	assert("cdemo_array21\000", cdemo_array21(v) == v)
	assert("demo_array21\000", demo_array21(v) == v)
}

type basepoint2 struct {
	x int32
	y int32
}

type point20 struct {
	base basepoint2
}

//go:linkname cdemo_point20 C.demo_point20
func cdemo_point20(a point20) point20

func demo_point20(a point20) point20 {
	return a
}

func init() {
	v := point20{basepoint2{1, 2}}
	assert("cdemo_point20\000", cdemo_point20(v) == v)
	assert("demo_point20\000", demo_point20(v) == v)
}

type point21 struct {
	base basepoint2
	z    int32
}

//go:linkname cdemo_point21 C.demo_point21
func cdemo_point21(a point21) point21

func demo_point21(a point21) point21 {
	return a
}

func init() {
	v := point21{basepoint2{1, 2}, 3}
	assert("cdemo_point21\000", cdemo_point21(v) == v)
	assert("demo_point21\000", demo_point21(v) == v)
}

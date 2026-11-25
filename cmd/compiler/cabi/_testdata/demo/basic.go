package main

import "unsafe"

const (
	LLGoFiles = "../wrap/basic.c"
)

type pointer = *int8

//go:linkname printf C.printf
func printf(format *byte, __llgo_va_list ...any) int32

func assert(info string, b bool) {
	if !b {
		printf(unsafe.StringData("Assertion failed: %s\n\000"), unsafe.StringData(info))
	}
}

func main() {
}

//go:linkname cbasic_int8 C.basic_int8
func cbasic_int8(a int8) int8

func basic_int8(a int8) int8 {
	return a
}

func init() {
	assert("cbasic_int8\000", cbasic_int8(100) == 100)
	assert("basic_int8\000", basic_int8(100) == 100)
}

//go:linkname cbasic_int16 C.basic_int16
func cbasic_int16(a int16) int16

func basic_int16(a int16) int16 {
	return a
}

func init() {
	assert("cbasic_int16\000", cbasic_int16(100) == 100)
	assert("basic_int16\000", basic_int16(100) == 100)
}

//go:linkname cbasic_int32 C.basic_int32
func cbasic_int32(a int32) int32

func basic_int32(a int32) int32 {
	return a
}

func init() {
	assert("cbasic_int32\000", cbasic_int32(100) == 100)
	assert("basic_int32\000", basic_int32(100) == 100)
}

//go:linkname cbasic_int64 C.basic_int64
func cbasic_int64(a int64) int64

func basic_int64(a int64) int64 {
	return a
}

func init() {
	assert("cbasic_int64\000", cbasic_int64(100) == 100)
	assert("basic_int64\000", basic_int64(100) == 100)
}

//go:linkname cbasic_float32 C.basic_float32
func cbasic_float32(a float32) float32

func basic_float32(a float32) float32 {
	return a
}

func init() {
	assert("cbasic_float32\000", cbasic_float32(100) == 100)
	assert("basic_float32\000", basic_float32(100) == 100)
}

//go:linkname cbasic_float64 C.basic_float64
func cbasic_float64(a float64) float64

func basic_float64(a float64) float64 {
	return a
}

func init() {
	assert("cbasic_float64\000", cbasic_float64(100) == 100)
	assert("basic_float64\000", basic_float64(100) == 100)
}

//go:linkname cbasic_pointer C.basic_pointer
func cbasic_pointer(a pointer) pointer

func basic_pointer(a pointer) pointer {
	return a
}

func init() {
	var p int8
	assert("cbasic_pointer\000", cbasic_pointer(&p) == &p)
	assert("basic_pointer\000", basic_pointer(&p) == &p)
}

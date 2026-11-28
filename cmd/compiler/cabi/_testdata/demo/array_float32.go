package main

import "unsafe"

const (
	LLGoFiles = "../wrap/array_float32.c"
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

type array1 struct {
	x [1]float32
}

//go:linkname cdemo1 C.demo1
func cdemo1(a array1) array1

func demo1(a array1) array1 {
	return a
}

func init() {
	a := array1{x: [1]float32{1}}
	assert("cdemo1\000", cdemo1(a) == a)
	assert("demo1\000", demo1(a) == a)
}

type array2 struct {
	x [2]float32
}

//go:linkname cdemo2 C.demo2
func cdemo2(a array2) array2

func demo2(a array2) array2 {
	return a
}

func init() {
	a := array2{x: [2]float32{1, 2}}
	assert("cdemo2\000", cdemo2(a) == a)
	assert("demo2\000", demo2(a) == a)
}

type array3 struct {
	x [3]float32
}

//go:linkname cdemo3 C.demo3
func cdemo3(a array3) array3

func demo3(a array3) array3 {
	return a
}

func init() {
	a := array3{x: [3]float32{1, 2, 3}}
	assert("cdemo3\000", cdemo3(a) == a)
	assert("demo3\000", demo3(a) == a)
}

type array4 struct {
	x [4]float32
}

//go:linkname cdemo4 C.demo4
func cdemo4(a array4) array4

func demo4(a array4) array4 {
	return a
}

func init() {
	a := array4{x: [4]float32{1, 2, 3, 4}}
	assert("cdemo4\000", cdemo4(a) == a)
	assert("demo4\000", demo4(a) == a)
}

type array5 struct {
	x [5]float32
}

//go:linkname cdemo5 C.demo5
func cdemo5(a array5) array5

func demo5(a array5) array5 {
	return a
}

func init() {
	a := array5{x: [5]float32{1, 2, 3, 4, 5}}
	assert("cdemo5\000", cdemo5(a) == a)
	assert("demo5\000", demo5(a) == a)
}

type array6 struct {
	x [6]float32
}

//go:linkname cdemo6 C.demo6
func cdemo6(a array6) array6

func demo6(a array6) array6 {
	return a
}

func init() {
	a := array6{x: [6]float32{1, 2, 3, 4, 5, 6}}
	assert("cdemo6\000", cdemo6(a) == a)
	assert("demo6\000", demo6(a) == a)
}

type array7 struct {
	x [7]float32
}

//go:linkname cdemo7 C.demo7
func cdemo7(a array7) array7

func demo7(a array7) array7 {
	return a
}

func init() {
	a := array7{x: [7]float32{1, 2, 3, 4, 5, 6, 7}}
	assert("cdemo7\000", cdemo7(a) == a)
	assert("demo7\000", demo7(a) == a)
}

type array8 struct {
	x [8]float32
}

//go:linkname cdemo8 C.demo8
func cdemo8(a array8) array8

func demo8(a array8) array8 {
	return a
}

func init() {
	a := array8{x: [8]float32{1, 2, 3, 4, 5, 6, 7, 8}}
	assert("cdemo8\000", cdemo8(a) == a)
	assert("demo8\000", demo8(a) == a)
}

type array9 struct {
	x [9]float32
}

//go:linkname cdemo9 C.demo9
func cdemo9(a array9) array9

func demo9(a array9) array9 {
	return a
}

func init() {
	a := array9{x: [9]float32{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	assert("cdemo9\000", cdemo9(a) == a)
	assert("demo9\000", demo9(a) == a)
}

type array10 struct {
	x [10]float32
}

//go:linkname cdemo10 C.demo10
func cdemo10(a array10) array10

func demo10(a array10) array10 {
	return a
}

func init() {
	a := array10{x: [10]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	assert("cdemo10\000", cdemo10(a) == a)
	assert("demo10\000", demo10(a) == a)
}

type array11 struct {
	x [11]float32
}

//go:linkname cdemo11 C.demo11
func cdemo11(a array11) array11

func demo11(a array11) array11 {
	return a
}

func init() {
	a := array11{x: [11]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}
	assert("cdemo11\000", cdemo11(a) == a)
	assert("demo11\000", demo11(a) == a)
}

type array12 struct {
	x [12]float32
}

//go:linkname cdemo12 C.demo12
func cdemo12(a array12) array12

func demo12(a array12) array12 {
	return a
}

func init() {
	a := array12{x: [12]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	assert("cdemo12\000", cdemo12(a) == a)
	assert("demo12\000", demo12(a) == a)
}

type array13 struct {
	x [13]float32
}

//go:linkname cdemo13 C.demo13
func cdemo13(a array13) array13

func demo13(a array13) array13 {
	return a
}

func init() {
	a := array13{x: [13]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}}
	assert("cdemo13\000", cdemo13(a) == a)
	assert("demo13\000", demo13(a) == a)
}

type array14 struct {
	x [14]float32
}

//go:linkname cdemo14 C.demo14
func cdemo14(a array14) array14

func demo14(a array14) array14 {
	return a
}

func init() {
	a := array14{x: [14]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}}
	assert("cdemo14\000", cdemo14(a) == a)
	assert("demo14\000", demo14(a) == a)
}

type array15 struct {
	x [15]float32
}

//go:linkname cdemo15 C.demo15
func cdemo15(a array15) array15

func demo15(a array15) array15 {
	return a
}

func init() {
	a := array15{x: [15]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}}
	assert("cdemo15\000", cdemo15(a) == a)
	assert("demo15\000", demo15(a) == a)
}

type array16 struct {
	x [16]float32
}

//go:linkname cdemo16 C.demo16
func cdemo16(a array16) array16

func demo16(a array16) array16 {
	return a
}

func init() {
	a := array16{x: [16]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}}
	assert("cdemo16\000", cdemo16(a) == a)
	assert("demo16\000", demo16(a) == a)
}

type array17 struct {
	x [17]float32
}

//go:linkname cdemo17 C.demo17
func cdemo17(a array17) array17

func demo17(a array17) array17 {
	return a
}

func init() {
	a := array17{x: [17]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}}
	assert("cdemo17\000", cdemo17(a) == a)
	assert("demo17\000", demo17(a) == a)
}

type array18 struct {
	x [18]float32
}

//go:linkname cdemo18 C.demo18
func cdemo18(a array18) array18

func demo18(a array18) array18 {
	return a
}

func init() {
	a := array18{x: [18]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}}
	assert("cdemo18\000", cdemo18(a) == a)
	assert("demo18\000", demo18(a) == a)
}

type array19 struct {
	x [19]float32
}

//go:linkname cdemo19 C.demo19
func cdemo19(a array19) array19

func demo19(a array19) array19 {
	return a
}

func init() {
	a := array19{x: [19]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}}
	assert("cdemo19\000", cdemo19(a) == a)
	assert("demo19\000", demo19(a) == a)
}

type array20 struct {
	x [20]float32
}

//go:linkname cdemo20 C.demo20
func cdemo20(a array20) array20

func demo20(a array20) array20 {
	return a
}

func init() {
	a := array20{x: [20]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}}
	assert("cdemo20\000", cdemo20(a) == a)
	assert("demo20\000", demo20(a) == a)
}

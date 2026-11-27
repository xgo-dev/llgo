package main

import (
	_ "unsafe"
)

const (
	LLGoFiles = "../wrap/demo.c"
)

type point struct {
	x int32
	y int32
}

//go:linkname pt C.pt
func pt(pt point) point

type point1 struct {
	x int32
	y int32
	z int32
}

//go:linkname pt1 C.pt1
func pt1(pt point1) point1

type point2 struct {
	x int8
	y int32
	z int32
}

//go:linkname pt2 C.pt2
func pt2(pt point2) point2

type point3 struct {
	x int8
	y int8
	z int8
}

//go:linkname pt3 C.pt3
func pt3(pt point3) point3

type point4 struct {
	x int8
	y int8
	z int8
	m int32
}

//go:linkname pt4 C.pt4
func pt4(pt point4) point4

type point5 struct {
	x int8
	y int8
	z int8
	m int8
	n int8
}

//go:linkname pt5 C.pt5
func pt5(pt point5) point5

type point6 struct {
	x int8
	y int8
	z int8
	m int8
	n int8
	k int32
}

//go:linkname pt6 C.pt6
func pt6(pt point6) point6

type point7 struct {
	x int8
	y int8
	z int8
	m int8
	n int8
	k int32
	o int8
}

//go:linkname pt7 C.pt7
func pt7(pt point7) point7

type data1 struct {
	x int8
	y int64
}

//go:linkname fn1 C.fn1
func fn1(data1) data1

type data2 struct {
	x int32
	y int64
}

//go:linkname fn2 C.fn2
func fn2(data2) data2

type data3 struct {
	x int64
	y int8
}

//go:linkname fn3 C.fn3
func fn3(data3) data3

type fdata1 struct {
	x float32
}

//go:linkname ff1 C.ff1
func ff1(fdata1) fdata1

type fdata2 struct {
	x float32
	y float32
}

//go:linkname ff2 C.ff2
func ff2(fdata2) fdata2

type fdata2i struct {
	x float32
	y int32
}

//go:linkname ff2i C.ff2i
func ff2i(fdata2i) fdata2i

type fdata3 struct {
	x float32
	y float32
	z float32
}

//go:linkname ff3 C.ff3
func ff3(fdata3) fdata3

type fdata4 struct {
	x float32
	y float32
	z float32
	m float32
}

//go:linkname ff4 C.ff4
func ff4(fdata4) fdata4

type fdata5 struct {
	x float32
	y float32
	z float32
	m float32
	n float32
}

//go:linkname ff5 C.ff5
func ff5(fdata5) fdata5

type fdata2id struct {
	x int8
	y int8
	z float64
}

//go:linkname ff2id C.ff2id
func ff2id(fdata2id) fdata2id

type fdata7if struct {
	x [7]int8
	y float32
}

//go:linkname ff7if C.ff7if
func ff7if(fdata7if) fdata7if

type fdata4if struct {
	x float32
	y int8
	z float32
	m float32
}

//go:linkname ff4if C.ff4if
func ff4if(fdata4if) fdata4if

type array struct {
	x [8]int32
}

//go:linkname demo64 C.demo64
func demo64(n int64) int64

//go:linkname demo32 C.demo32
func demo32(n int32) int32

type struct32 struct {
	v int32
}

//go:linkname demo32s C.demo32s
func demo32s(v struct32) struct32

type point64 struct {
	x int64
	y int64
}

//go:linkname pt64 C.pt64
func pt64(pt point64) point64

//go:linkname demo C.demo
func demo(a array) array

//go:linkname demo2 C.demo2
func demo2(x int32) array

type ddata1 struct {
	x float64
}

//go:linkname dd1 C.dd1
func dd1(d ddata1) ddata1

type ddata2 struct {
	x float64
	y float64
}

//go:linkname dd2 C.dd2
func dd2(d ddata2) ddata2

type ddata3 struct {
	x float64
	y float64
	z float64
}

//go:linkname dd3 C.dd3
func dd3(d ddata3) ddata3

//llgo:cdecl
type Callback func(array, point, point1) array

//go:linkname callback C.callback
func callback(fn Callback, ar array)

//llgo:cdecl
type Callback1 func(array, point, point1) point

//go:linkname callback1 C.callback1
func callback1(fn Callback1, ar array)

//go:linkname mycallback C.mycallback
func mycallback(ar array, pt point, pt1 point1) point

func myfn1(ar array, pt point, pt1 point1) point {
	println("=>", ar.x[0], ar.x[1], ar.x[7], pt.x, pt.y, pt1.x, pt1.y, pt1.z)
	return point{100, 200}
}

//export export_demo
func export_demo(ar array) array {
	println("=> export", ar.x[0], ar.x[1], ar.x[7])
	return ar
}

func main() {
	cabi_demo()
	callback_demo()
}

func callback_demo() {
	export_demo(array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}})

	callback(func(ar array, pt point, pt1 point1) array {
		println("=> callback", ar.x[0], ar.x[1], ar.x[7], pt.x, pt.y, pt1.x, pt1.y, pt1.z)
		return array{x: [8]int32{8, 7, 6, 5, 4, 3, 2, 1}}
	}, array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}})

	callback1(func(ar array, pt point, pt1 point1) point {
		println("=> callback1", ar.x[0], ar.x[1], ar.x[7], pt.x, pt.y, pt1.x, pt1.y, pt1.z)
		return point{100, 200}
	}, array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}})
	ret := mycallback(array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}}, point{1, 2}, point1{1, 2, 3})
	println("=> mycallback", ret.x, ret.y)
	callback1(myfn1, array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}})
	callback1(myfn1, array{x: [8]int32{8, 7, 6, 5, 4, 3, 2, 1}})
	callback1(mycallback, array{x: [8]int32{10, 20, 30, 40, 50, 60, 70, 80}})
}

func cabi_demo() {
	i32 := demo32(1024)
	println("=> demo32", i32)

	s32 := demo32s(struct32{100})
	println("=> demo32s", s32.v)

	i64 := demo64(1024)
	println("=> demo64", i64)

	p64 := pt64(point64{1024, -1024})
	println("=> pt64", p64.x, p64.y)

	r := demo(array{x: [8]int32{1, 2, 3, 4, 5, 6, 7, 8}})
	println("=> demo", r.x[0], r.x[1])

	r2 := demo2(100)
	println("=> demo2", r2.x[0], r2.x[1], r2.x[7])

	p0 := pt(point{1, 2})
	println("=> pt0", p0.x, p0.y)

	p1 := pt1(point1{1, 2, 3})
	println("=> pt1", p1.x, p1.y, p1.z)

	p2 := pt2(point2{1, 2, 3})
	println("=> pt2", p2.x, p2.y, p2.z)

	p3 := pt3(point3{1, 2, 3})
	println("=> pt3", p3.x, p3.y, p3.z)

	p4 := pt4(point4{1, 2, 3, 4})
	println("=> pt4", p4.x, p4.y, p4.z, p4.m)

	p5 := pt5(point5{1, 2, 3, 4, 5})
	println("=> pt5", p5.x, p5.y, p5.z, p5.m, p5.n)

	p6 := pt6(point6{1, 2, 3, 4, 5, 6})
	println("=> pt6", p6.x, p6.y, p6.z, p6.m, p6.n, p6.k)

	p7 := pt7(point7{1, 2, 3, 4, 5, 6, 7})
	println("=> pt7", p7.x, p7.y, p7.z, p7.m, p7.n, p7.k, p7.o)

	// skip wrap
	fd1 := fn1(data1{1, 2})
	println("=> fd1", fd1.x, fd1.y)

	fd2 := fn2(data2{1, 2})
	println("=> fd2", fd2.x, fd2.y)

	fd3 := fn3(data3{1, 2})
	println("=> fd3", fd3.x, fd3.y)

	// float
	f1 := ff1(fdata1{1.1})
	println("=> f1", f1.x)

	// float
	f2 := ff2(fdata2{1.1, 2.1})
	println("=> f2", f2.x, f2.y)

	// float
	f2i := ff2i(fdata2i{1.1, 2})
	println("=> f2i", f2i.x, f2i.y)

	// float
	f3 := ff3(fdata3{1.1, 2.1, 3.1})
	println("=> f3", f3.x, f3.y, f3.z)

	// float
	f4 := ff4(fdata4{1.1, 2.1, 3.1, 4.1})
	println("=> f4", f4.x, f4.y, f4.z, f4.m)

	// float
	f5 := ff5(fdata5{1.1, 2.1, 3.1, 4.1, 5.1})
	println("=> f5", f5.x, f5.y, f5.z, f5.m, f5.n)

	f2id := ff2id(fdata2id{1, 2, 3.1})
	println("=> f2id", f2id.x, f2id.y, f2id.z)

	f7if := ff7if(fdata7if{[7]int8{1, 2, 3, 4, 5, 6, 7}, 3.1})
	println("=> f7if", f7if.x[0], f7if.x[1], f7if.y)

	f4if := ff4if(fdata4if{1.1, 2, 3.1, 4.1})
	println("=> f4if", f4if.x, f4if.y, f4if.z, f4if.m)

	d1 := dd1(ddata1{1.1})
	println("=> dd1", d1.x)

	d2 := dd2(ddata2{1.1, 2.1})
	println("=> dd2", d2.x, d2.y)

	d3 := dd3(ddata3{1.1, 2.1, 3.1})
	println("=> dd3", d3.x, d3.y, d3.z)
}

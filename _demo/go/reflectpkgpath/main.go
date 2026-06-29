package main

import (
	"reflect"
	"unsafe"
)

func main() {
	demo1()
	demo2()
}

func demo1() {
	type T unsafe.Pointer
	t := reflect.TypeOf(unsafe.Pointer(nil))
	t1 := reflect.TypeOf(T(nil))
	if t.Name() != "Pointer" {
		panic("error: " + t.Name())
	}
	if t.PkgPath() != "unsafe" {
		panic("error: " + t.PkgPath())
	}
	if t1.Name() != "T" {
		panic("error: " + t1.Name())
	}
	if t1.PkgPath() == "unsafe" {
		panic("error: bad pkgpath")
	}
}

type Point struct {
	X int
	Y int
}

func (pt *Point) Set(x, y int) {
	pt.X, pt.Y = x, y
}

type My interface {
	Demo()
}

func demo2() {
	typ1 := reflect.TypeOf((*My)(nil)).Elem()
	typ2 := reflect.TypeOf((*Point)(nil)).Elem()
	if typ1.Name() != "My" {
		panic("error typ1 name")
	}
	if typ2.Name() != "Point" {
		panic("error typ2 name")
	}
	if typ1.PkgPath() == "" {
		panic("error typ1 pkgpath")
	}
	if typ2.PkgPath() == "" {
		panic("error typ2 pkgpath")
	}
	if typ1.PkgPath() != typ2.PkgPath() {
		panic("error pkgpath")
	}
	if typ1.NumMethod() != 1 {
		panic("error typ1 num method")
	}
	if typ2.NumMethod() != 0 {
		panic("error typ2 num method")
	}
	if reflect.PointerTo(typ2).NumMethod() != 1 {
		panic("error *typ2 num method")
	}
}

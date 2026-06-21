package main

import (
	"reflect"
	"unsafe"
)

func main() {
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

package main

import _ "unsafe"

const LLGoFiles = "_wrap/check.c"

type Receiver int32

//llgo:link Receiver.Check C.review_check
func (Receiver) Check(__llgo_va_list ...any) int32 { return 0 }

type Embedded struct{ Receiver }

type Checker interface {
	Check(__llgo_va_list ...any) int32
}

//go:noinline
func check(v Checker) int32 {
	return v.Check(int32(100), int32(200))
}

func main() {
	e := Embedded{7}
	println(check(&e))
}

package main

import _ "unsafe"

const LLGoFiles = "_wrap/check.c"

type Receiver int32

//llgo:link (*Receiver).Check C.variadic_method_check
func (*Receiver) Check(__llgo_va_list ...any) int32 { return 0 }

type Embedded struct {
	padding [3]int64
	*Receiver
}

type Nested struct {
	padding int64
	*Embedded
}

type Checker interface {
	Check(__llgo_va_list ...any) int32
}

//go:noinline
func check(value Checker) {
	// More arguments than the native GP/FP register sets can hold: both
	// register and stack ellipsis arguments must survive receiver adjustment.
	if value.Check(
		int32(100), int32(101), int32(102), int32(103), int32(104),
		int32(105), int32(106), int32(107), int32(108), int32(109),
		0.25, 1.25, 2.25, 3.25, 4.25, 5.25, 6.25, 7.25, 8.25, 9.25,
	) != 42 {
		panic("variadic method arguments were corrupted")
	}
}

//go:noinline
func empty(value Checker) {
	if value.Check() != 42 {
		panic("empty variadic method call failed")
	}
}

func checkNil(value Checker) {
	defer func() {
		if recover() == nil {
			panic("nil promoted receiver did not panic")
		}
	}()
	check(value)
}

func main() {
	marker := Receiver(7)
	embedded := Embedded{Receiver: &marker}
	nested := Nested{Embedded: &embedded}
	check(&embedded)
	check(embedded)
	check(&nested)
	check(nested)
	marker = 0
	empty(&embedded)
	empty(&nested)
	checkNil((*Embedded)(nil))
	checkNil(&Nested{})
	println("variadic-method-ok")
}

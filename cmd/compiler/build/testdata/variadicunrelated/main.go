package main

import _ "unsafe"

const LLGoFiles = "_wrap/check.c"

type T int32

//llgo:link T.Check C.review_unused_check
func (T) Check(__llgo_va_list ...any) int32 { return 0 }

func (T) String() string { return "ok" }

type Wrapper struct{ T }
type Stringer interface{ String() string }

var global Stringer = T(7)

//go:noinline
func box[V any](value V) any { return value }

//go:noinline
func check(value Stringer) {
	if value.String() != "ok" {
		panic("unrelated interface dispatch")
	}
}

func main() {
	value := T(7)
	wrapper := Wrapper{value}
	check(value)
	check(&value)
	check(wrapper)
	check(&wrapper)
	check(global)
	if box(value).(T) != value || box(&wrapper).(*Wrapper) != &wrapper {
		panic("any round trip")
	}
	println("variadic-unrelated-ok")
}

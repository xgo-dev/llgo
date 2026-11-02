package main

import "github.com/xgo-dev/llgo/internal/build/testdata/variadicimport/receiver"

type Embedded struct{ receiver.Receiver }

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

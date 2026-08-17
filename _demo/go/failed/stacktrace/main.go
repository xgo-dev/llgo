package main

import (
	"fmt"
)

type MyStruct[T any] struct {
	value T
}

func (m *MyStruct[T]) Method() {
	fmt.Println("In generic method")
	genericFunc[T](m.value)
}

func genericFunc[T any](v T) {
	fmt.Println("In generic function")
	normalFunc()
}

func normalFunc() {
	fmt.Println("In normal function")
	panic("panic occurs here")
}

func main() {
	m := &MyStruct[string]{value: "hello"}
	m.Method()
}

//Expected:
// In generic method
// In generic function
// In normal function
// panic: panic occurs here

// [0x00C6D310 github.com/xgo-dev/llgo/internal/runtime.Rethrow+0x2f, SP = 0x60]
// [0x00C6CF44 github.com/xgo-dev/llgo/internal/runtime.Panic+0x2d, SP = 0x50]
// [0x00C69420 main.normalFunc+0xf, SP = 0xa8]
// [0x00C69564 main.genericFunc[string]+0x18, SP = 0x74]
// [0x00C694A8 main.(*MyStruct[string]).Method+0x1f, SP = 0x84]
// [0x00C6936C main+0x4, SP = 0x40]

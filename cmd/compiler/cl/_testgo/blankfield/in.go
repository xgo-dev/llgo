// LITTEST
package main

import "unsafe"

var calls int

func sideEffect() int {
	calls++
	return 42
}

type Nested struct {
	Left  int
	_     int
	Right int
}

// CHECK-LABEL: define void @{{(".*blankfield\.main"|main\.main)}}()
// CHECK: call i64 @{{(".*blankfield\.sideEffect"|main\.sideEffect)}}()
// CHECK: call i64 @{{(".*blankfield\.sideEffect"|main\.sideEffect)}}()
// CHECK: call i64 @{{(".*blankfield\.sideEffect"|main\.sideEffect)}}()

func main() {
	value := struct {
		_    int
		Keep int
	}{sideEffect(), 7}
	nestedValue := struct {
		_    Nested
		Keep int
	}{Nested{sideEffect(), 6, 7}, 8}
	arrayValue := struct {
		_    [2]int
		Keep int
	}{[2]int{sideEffect(), 9}, 10}

	if calls != 3 {
		panic("blank field initializer side effect was not evaluated")
	}
	if value.Keep != 7 {
		panic("non-blank field initializer was lost")
	}
	nestedWords := (*[4]int)(unsafe.Pointer(&nestedValue))
	for i := 0; i < 3; i++ {
		if nestedWords[i] != 0 {
			panic("nested blank field was not zeroed")
		}
	}
	if nestedWords[3] != 8 {
		panic("nested non-blank field initializer was lost")
	}
	arrayWords := (*[3]int)(unsafe.Pointer(&arrayValue))
	for i := 0; i < 2; i++ {
		if arrayWords[i] != 0 {
			panic("blank array field was not zeroed")
		}
	}
	if arrayWords[2] != 10 {
		panic("array non-blank field initializer was lost")
	}
	println("ok")
}

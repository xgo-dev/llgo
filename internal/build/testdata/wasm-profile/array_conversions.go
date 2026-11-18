package main

import (
	"runtime"
	"unsafe"
)

//llgo:cdecl
type conversionWords [2]uintptr

//llgo:cdecl
type conversionNested [2][2]uintptr

//llgo:cdecl
type conversionPointers [2]*byte

//llgo:cdecl
type conversionCallbacks [2]func(int32) int32

//llgo:cdecl
type conversionCallback func(int32) int32

//llgo:cdecl
type conversionNamedCallbacks [2]conversionCallback

//llgo:cdecl
type conversionEmpty [0]uintptr

var conversionGlobalGo = [2]uintptr{7, 9}
var conversionGlobalNative conversionWords = conversionGlobalGo
var conversionGlobalBack [2]uintptr = conversionGlobalNative

//go:noinline
func wordsFromGo(a [2]uintptr) conversionWords { return conversionWords(a) }

//go:noinline
func wordsToGo(a conversionWords) [2]uintptr { return [2]uintptr(a) }

//go:noinline
func assignWordsFromGo(a [2]uintptr) conversionWords {
	var out conversionWords
	out = a
	return out
}

//go:noinline
func assignWordsToGo(a conversionWords) [2]uintptr {
	var out [2]uintptr
	out = a
	return out
}

//go:noinline
func nestedFromGo(a [2][2]uintptr) conversionNested { return conversionNested(a) }

//go:noinline
func nestedToGo(a conversionNested) [2][2]uintptr { return [2][2]uintptr(a) }

//go:noinline
func pointersFromGo(a [2]*byte) conversionPointers { return conversionPointers(a) }

//go:noinline
func pointersToGo(a conversionPointers) [2]*byte { return [2]*byte(a) }

//go:noinline
func callbacksFromGo(a [2]func(int32) int32) conversionCallbacks { return conversionCallbacks(a) }

//go:noinline
func callbacksToGo(a conversionCallbacks) [2]func(int32) int32 { return [2]func(int32) int32(a) }

//go:noinline
func namedCallbacksFromGo(a [2]conversionCallback) conversionNamedCallbacks {
	return conversionNamedCallbacks(a)
}

//go:noinline
func namedCallbacksToGo(a conversionNamedCallbacks) [2]conversionCallback {
	return [2]conversionCallback(a)
}

//go:noinline
func emptyFromGo(a [0]uintptr) conversionEmpty { return conversionEmpty(a) }

//go:noinline
func emptyToGo(a conversionEmpty) [0]uintptr { return [0]uintptr(a) }

func conversionAddOne(x int32) int32 { return x + 1 }

func expectArrayConversionPanic(fn func()) {
	defer func() {
		if recover() == nil {
			panic("native array conversion did not reject an invalid value")
		}
	}()
	fn()
}

func checkArrayConversions() {
	if conversionGlobalNative != (conversionWords{7, 9}) || conversionGlobalBack != conversionGlobalGo {
		panic("native array global assignment corrupted a value")
	}
	for _, source := range [][2]uintptr{{7, 9}, {uintptr(^uint32(0)), uintptr(^uint32(0)) - 1}} {
		native := wordsFromGo(source)
		if wordsToGo(native) != source || assignWordsToGo(assignWordsFromGo(source)) != source {
			panic("native array word conversion corrupted a value")
		}
		native[0] = 17
		if source[0] == 17 {
			panic("array value conversion aliased its source")
		}
	}
	nested := [2][2]uintptr{{7, 9}, {11, uintptr(^uint32(0))}}
	if nestedToGo(nestedFromGo(nested)) != nested {
		panic("nested native array conversion corrupted a value")
	}
	a, b := byte(7), byte(9)
	pointers := pointersToGo(pointersFromGo([2]*byte{&a, &b}))
	runtime.GC()
	if pointers != [2]*byte{&a, &b} || *pointers[0] != 7 || *pointers[1] != 9 {
		panic("native array pointer conversion corrupted a value")
	}
	callbacks := callbacksToGo(callbacksFromGo([2]func(int32) int32{conversionAddOne, conversionAddOne}))
	if callbacks[0](41) != 42 || callbacks[1](7) != 8 {
		panic("native array callback conversion corrupted a value")
	}
	named := namedCallbacksToGo(namedCallbacksFromGo([2]conversionCallback{conversionAddOne, conversionAddOne}))
	if named[0](41) != 42 || named[1](7) != 8 {
		panic("native array named callback conversion corrupted a value")
	}
	if unsafe.Sizeof(conversionCallbacks{}) < unsafe.Sizeof([2]func(int32) int32{}) {
		var captured int32
		fn := func(x int32) int32 { captured++; return x + captured }
		expectArrayConversionPanic(func() { _ = callbacksFromGo([2]func(int32) int32{fn, conversionAddOne}) })
	}
	if emptyToGo(emptyFromGo([0]uintptr{})) != [0]uintptr{} {
		panic("empty native array conversion failed")
	}
	if unsafe.Sizeof(conversionWords{}) < unsafe.Sizeof([2]uintptr{}) {
		high := uint64(1) << 32
		bad := [2]uintptr{7, uintptr(high)}
		expectArrayConversionPanic(func() { _ = wordsFromGo(bad) })
		expectArrayConversionPanic(func() { _ = assignWordsFromGo(bad) })
		expectArrayConversionPanic(func() { _ = nestedFromGo([2][2]uintptr{{7, 9}, bad}) })
	}
}

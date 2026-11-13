package main

import "unsafe"

type OffsetGo struct {
	Fn   func()
	Tail uint64
}

//llgo:cdecl
type OffsetNative OffsetGo

const cachedNativeOffset = unsafe.Offsetof(OffsetNative{}.Tail)

type cachedOffsetBuffer [cachedNativeOffset]byte

func init() {
	var value OffsetNative
	if uintptr(len(cachedOffsetBuffer{})) != uintptr(unsafe.Pointer(&value.Tail))-uintptr(unsafe.Pointer(&value)) {
		panic("cached native offset")
	}
}

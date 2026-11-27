package main

import _ "unsafe"

const LLGoFiles = "_wrap/probe.cpp"

//go:linkname cppProbe C.llgo_debug_cpp_probe
func cppProbe(int32) int32

//go:noinline
func goProbe(input int32) int32 {
	cppResult := cppProbe(input)
	return cppResult * 2
}

func main() {
	if got := goProbe(14); got != 42 {
		panic("debug probe returned the wrong value")
	}
	println("wasm debug ok")
}

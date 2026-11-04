package main

import "unsafe"

var entry uintptr

//go:linkname tableByte callbackTable
var tableByte byte

var tableAddress = uintptr(unsafe.Pointer(&tableByte))

func Ordinary()

func main() {
	Ordinary()
	if entry == 0 || tableAddress == 0 || entry == tableAddress {
		panic("selected native component addresses were not linked independently")
	}
	println("native component addresses verified")
}

package main

var entry uintptr

func main() {
	if entry == 0 {
		panic("assembly code address was not linked into the Go global")
	}
	println("native address relocation verified")
}

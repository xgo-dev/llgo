//go:build !wasm

package main

import _ "unsafe"

//go:linkname mainDemo main.demo
func mainDemo() int

func main() {
	mainDemo()
}

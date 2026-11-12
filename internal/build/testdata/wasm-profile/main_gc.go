//go:build !llgo

package main

func main() {
	checkGoLayout()
	checkArrayConversions()
	println("wasm Go layout ok")
}

//go:build !llgo

package main

func main() {
	checkGoLayout()
	println("wasm Go layout ok")
}

//go:build llgo && !wasm

package c11

/*
#cgo CFLAGS: -std=c11

static int add_one(int value) { return value + 1; }
*/
import "C"

func addOne(value int) int {
	return int(C.add_one(C.int(value)))
}

//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package test

import "testing"

func TestStringConcatWasm32AddressLimit(t *testing.T) {
	// Both lengths fit the 64-bit Go word. Their sum cannot be represented by
	// wasm32's allocator/memcpy size, so reject it without allocating 4 GiB.
	checkStringConcatLengthPanic(t, 1<<31, 1<<31)
}

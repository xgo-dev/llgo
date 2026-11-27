//go:build !esp

package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/cmd/cl/_testlibc/callbackmethod/local"
)

func rejectCaptured(node local.Node, data unsafe.Pointer, offset int32) {
	defer func() {
		err, ok := recover().(error)
		if !ok || err.Error() != "runtime error: C callback must not capture variables" {
			panic("capturing C callback was not rejected")
		}
	}()
	node.Visit(func(node local.Node, _ unsafe.Pointer) int32 {
		return node.Data + offset
	}, data)
}

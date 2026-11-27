//go:build esp

package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/cmd/cl/_testlibc/callbackmethod/local"
)

// Bare-metal ESP reports a fatal error for a panic instead of unwinding it.
func rejectCaptured(local.Node, unsafe.Pointer, int32) {}

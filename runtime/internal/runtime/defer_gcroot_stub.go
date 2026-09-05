//go:build !llgo

package runtime

import "unsafe"

// SetDeferGCRoot keeps compiler-only host tests type-correct.
func SetDeferGCRoot(*Defer, unsafe.Pointer) {}

// RestoreDeferGCRoot keeps compiler-only host tests type-correct.
func RestoreDeferGCRoot(*Defer) {}

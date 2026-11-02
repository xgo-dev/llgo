//go:build !darwin && !windows

package main

import "unsafe"

//go:linkname stderr C.stderr
var stderr unsafe.Pointer

//go:build darwin

package main

import "unsafe"

//go:linkname stderr C.__stderrp
var stderr unsafe.Pointer

//go:build windows

package main

import "unsafe"

//go:linkname acrtIobFunc C.__acrt_iob_func
func acrtIobFunc(index uint32) unsafe.Pointer

var stderr = acrtIobFunc(2)

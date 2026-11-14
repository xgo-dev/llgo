//go:build llgo

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

// Package sourcelocation demonstrates reusable FILE/LINE bindings. Each read,
// including a read in an importing package, produces that read's source location.
package sourcelocation

import _ "unsafe" // for go:linkname

// File is a NUL-terminated C string naming the file containing the read.
//
//go:linkname File llgo.file
var File *int8

// Line is the line number of the read, with C int width on every target.
//
//go:linkname Line llgo.line
var Line int32

// Current demonstrates that a wrapper reports its own body, not its caller.
func Current() (*int8, int32) {
//line wrapper.xgo:80
	return File, Line
}

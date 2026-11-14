//go:build llgo

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package sourcelocation_test

import (
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/goplus/lib/c"
	location "github.com/xgo-dev/llgo/test/llgoext/sourcelocation"
)

// This read uses the physical source file, without a //line override.
var physicalFile = location.File

//line initializer.xgo:40
var initialFile, initialLine = location.File, location.Line

func checkLocation(t *testing.T, file *int8, line int32, wantFile string, wantLine int32) {
	t.Helper()
	if file == nil {
		t.Fatal("nil filename")
	}
	// Use the pointer through a C string API, including its NUL terminator.
	name := c.GoString(file)
	if filepath.Base(name) != wantFile || line != wantLine {
		t.Fatalf("location = %s:%d, want %s:%d", name, line, wantFile, wantLine)
	}
	if n := c.Strlen(file); n != uintptr(len(name)) {
		t.Fatalf("C filename length = %d, Go length = %d", n, len(name))
	}
}

func TestReadLocations(t *testing.T) {
//line read.xgo:100
	file, first := location.File, location.Line
	next := location.Line
	left, right := location.Line, location.Line
	again := location.File
	checkLocation(t, file, first, "read.xgo", 100)
	if next != 101 || left != 102 || right != 102 {
		t.Fatalf("successive/same-line reads = %d, %d, %d", next, left, right)
	}
	if file != again {
		t.Fatal("identical filenames were not pooled in this module")
	}
//line other.xgo:200
	otherFile, otherLine := location.File, location.Line
	checkLocation(t, otherFile, otherLine, "other.xgo", 200)
	if file == otherFile {
		t.Fatal("different filenames share the same string")
	}
	if physicalFile == nil || !strings.HasSuffix(c.GoString(physicalFile), "location_test.go") {
		t.Fatal("default filename did not retain the physical source path")
	}
	if unsafe.Sizeof(location.Line) != 4 {
		t.Fatal("Line must retain C int width")
	}
	// The value can be copied, modified and addressed like an ordinary value.
	copy := first
	p := &copy
	*p += 1
	if copy != 101 || first != 100 {
		t.Fatal("modifying a copy changed the captured location")
	}
}

func genericLocation[T any](value T) (*int8, int32) {
//line generic.xgo:300
	return location.File, location.Line
}

func TestInitializerWrapperGenericAndClosure(t *testing.T) {
	checkLocation(t, initialFile, initialLine, "initializer.xgo", 40)
	file, line := location.Current()
	checkLocation(t, file, line, "wrapper.xgo", 80)
	file, line = genericLocation(1)
	checkLocation(t, file, line, "generic.xgo", 300)
	file, line = genericLocation("another instantiation")
	checkLocation(t, file, line, "generic.xgo", 300)
//line closure.xgo:400
	closure := func() (*int8, int32) { return location.File, location.Line }
	file, line = closure()
	checkLocation(t, file, line, "closure.xgo", 400)
}

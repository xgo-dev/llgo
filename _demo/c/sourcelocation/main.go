package main

import (
	"strings"
	_ "unsafe" // for go:linkname

	"github.com/goplus/lib/c"
)

// These names are local bindings, not additions to the c package.
//
//go:linkname File llgo.file
var File *c.Char

//go:linkname Line llgo.line
var Line c.Int

// Pass File and Line at the call site so a logging helper reports its caller.
func logAt(file *c.Char, line c.Int, message *c.Char) {
	c.Printf(c.Str("%s:%d: %s\n"), file, line, message)
}

func main() {
	logAt(File, Line, c.Str("source locations need no runtime stack walk"))
	first := Line
	next := Line
	if first <= 0 || next != first+1 {
		panic("Line must describe each read, not the binding declaration")
	}
	if File == nil || !strings.HasSuffix(c.GoString(File), "main.go") {
		panic("File must be the source filename as a C string")
	}
	// Generated code can provide its original source location.
//line generated.xgo:200
	file, line := File, Line
	if line != 200 || !strings.HasSuffix(c.GoString(file), "generated.xgo") {
		panic("source location must honor //line")
	}
	logAt(file, line, c.Str("location preserved from generated code"))
}

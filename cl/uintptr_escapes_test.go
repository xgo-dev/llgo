//go:build !llgo

package cl_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestCompileUintptrEscapesRoots(t *testing.T) {
	const src = `package main
import "unsafe"
func collect()
//go:uintptrescapes
func marked(p uintptr) { collect(); _ = *(*int)(unsafe.Pointer(p)) }
func integer(p uintptr) { collect() }
func source() unsafe.Pointer
func laterArgument() uintptr
//go:uintptrescapes
func two(uintptr, uintptr)
func caller() { two(uintptr(source()), laterArgument()) }
`
	ir := cltest.CompileIREx(t, src, "uintptr_escapes.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	function := func(name string) string {
		t.Helper()
		start := strings.Index(ir, "define void @main."+name+"(")
		if start < 0 {
			t.Fatalf("missing %s:\n%s", name, ir)
		}
		body := ir[start:]
		end := strings.Index(body, "\n}")
		if end < 0 {
			t.Fatalf("unterminated %s:\n%s", name, body)
		}
		return body[:end]
	}
	marked := function("marked")
	if !strings.Contains(marked, "inttoptr i64 %0 to ptr") || !strings.Contains(marked, "@llvm_gc_root_chain") {
		t.Fatalf("pragma uintptr parameter was not rooted:\n%s", marked)
	}
	if integer := function("integer"); strings.Contains(integer, "@llvm_gc_root_chain") {
		t.Fatalf("ordinary integer parameter became a root:\n%s", integer)
	}
	caller := function("caller")
	source := strings.Index(caller, "call ptr @main.source()")
	if source < 0 {
		t.Fatalf("missing pointer-producing call:\n%s", caller)
	}
	store := strings.Index(caller[source:], "store ptr ")
	later := strings.Index(caller, "call i64 @main.laterArgument()")
	if store < 0 || later < 0 || source+store > later {
		t.Fatalf("source pointer not published before subsequent argument evaluation:\n%s", caller)
	}
}

func TestCompileUintptrEscapesRootsDisabled(t *testing.T) {
	const src = `package main
func collect()
//go:uintptrescapes
func marked(p uintptr) { collect() }
`
	ir := cltest.CompileIREx(t, src, "uintptr_escapes_disabled.go", false, nil)
	if strings.Contains(ir, "@llvm_gc_root_chain") || strings.Contains(ir, "inttoptr") {
		t.Fatalf("pragma changed the native conservative collector path:\n%s", ir)
	}
}

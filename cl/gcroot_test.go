//go:build !llgo

package cl_test

import (
	"strings"
	"testing"

	"github.com/goplus/llgo/cl/cltest"
	llssa "github.com/goplus/llgo/ssa"
)

func TestCompileDirectGCRoots(t *testing.T) {
	const src = `package main

func use(*int)

func keep(p *int) *int {
	use(p)
	return p
}

func choose(cond bool, a, b *int) *int {
	var p *int
	if cond {
		p = a
	} else {
		p = b
	}
	use(p)
	return p
}
`
	ir := cltest.CompileIREx(t, src, "gcroot.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	if !strings.Contains(ir, `@llvm_gc_root_chain`) {
		t.Fatalf("compiler-maintained root is missing:\n%s", ir)
	}
	if strings.Contains(ir, `llvm.gcroot`) || strings.Contains(ir, `gc "shadow-stack"`) {
		t.Fatalf("compiler emitted roots that still require backend lowering:\n%s", ir)
	}
	if !strings.Contains(ir, `store ptr %0`) {
		t.Fatalf("pointer parameter was not published:\n%s", ir)
	}
}

func TestCompileAggregateGCRoots(t *testing.T) {
	const src = `package main

type holder struct {
	p *int
	s []byte
	text string
	any any
	fn func()
	array [2]*int
}

func useHolder(holder)

func keep(h holder) holder {
	useHolder(h)
	return h
}
`
	ir := cltest.CompileIREx(t, src, "gcroot_aggregate.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	if !strings.Contains(ir, `[7 x ptr]`) {
		t.Fatalf("aggregate did not emit one seven-root frame:\n%s", ir)
	}
}

func TestCompileGCRootsDisabled(t *testing.T) {
	const src = `package main

func keep(p *int) *int { return p }
`
	ir := cltest.CompileIREx(t, src, "gcroot_disabled.go", false, nil)
	if strings.Contains(ir, `llvm_gc_root_chain`) || strings.Contains(ir, `llvm.gcroot`) ||
		strings.Contains(ir, `gc "shadow-stack"`) {
		t.Fatalf("disabled GC roots changed ordinary code:\n%s", ir)
	}
}

func TestCompileThreadLocalGCRoots(t *testing.T) {
	const src = `package main

func use(*int)

func keep(p *int) *int {
	use(p)
	return p
}
`
	ir := cltest.CompileIREx(t, src, "gcroot_tls.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
		prog.EnableThreadLocalGCRoots(true)
	})
	if !strings.Contains(ir, `thread_local`) ||
		!strings.Contains(ir, `github.com/goplus/llgo/runtime/internal/gcroot.currentRootChain`) {
		t.Fatalf("thread-local compiler root chain is missing:\n%s", ir)
	}
	if strings.Contains(ir, `@llvm_gc_root_chain`) {
		t.Fatalf("thread-local roots also emitted the single-worker chain:\n%s", ir)
	}
}

func TestCompileGCRootPlanning(t *testing.T) {
	const pure = `package main
func keep(p *int) *int { return p }
`
	ir := cltest.CompileIREx(t, pure, "gcroot_pure.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	if strings.Contains(ir, `llvm_gc_root_chain`) || strings.Contains(ir, `llvm.gcroot`) ||
		strings.Contains(ir, `gc "shadow-stack"`) {
		t.Fatalf("function without a safepoint emitted roots:\n%s", ir)
	}

	const allocating = `package main
func keep(p *int, n int) *int {
	_ = make([]byte, n)
	return p
}
`
	ir = cltest.CompileIREx(t, allocating, "gcroot_allocating.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	if !strings.Contains(ir, `[1 x ptr]`) {
		t.Fatalf("pointer live across allocation did not emit one root:\n%s", ir)
	}

	const closure = `package main
func use(*int)
func keep(p *int) func() {
	return func() { use(p) }
}
`
	ir = cltest.CompileIREx(t, closure, "gcroot_closure.go", false, func(prog llssa.Program) {
		prog.EnableGCRoots(true)
	})
	if !strings.Contains(ir, `@llvm_gc_root_chain`) {
		t.Fatalf("closure context live across a call was not rooted:\n%s", ir)
	}
}

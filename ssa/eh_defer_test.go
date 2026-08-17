//go:build !llgo
// +build !llgo

package ssa_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
)

func TestExplicitDeferStackIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	stack := b.BuiltinCall("ssa:deferstack")
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)
	b.DeferTo(fn, stack, callee.Expr, ssa.Builder.Call)
	b.DeferStackDrain()
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected explicit defer stack node cleanup in IR, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected explicit defer stack setup in IR, got:\n%s", ir)
	}
}

func TestExplicitDeferStackFallbackAndNilBuiltin(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	stack := b.BuiltinCall("ssa:deferstack")
	if stack.Type != prog.VoidPtr() {
		t.Fatalf("ssa:deferstack without recover returned %v, want %v", stack.Type, prog.VoidPtr())
	}
	b.DeferTo(nil, stack, callee.Expr, ssa.Builder.Call)
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "setjmp") {
		t.Fatalf("unexpected defer stack setup without recover, got:\n%s", ir)
	}
}

func TestExplicitDeferStackDrainWithoutLoopCases(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())

	_ = b.BuiltinCall("ssa:deferstack")
	b.DeferStackDrain()
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("unexpected explicit defer cleanup without loop cases, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected defer stack setup with recover, got:\n%s", ir)
	}
}

func TestExplicitDeferStackDrainWithoutRecoverNoop(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	b.DeferStackDrain()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") || strings.Contains(ir, "sigsetjmp") || strings.Contains(ir, "setjmp") {
		t.Fatalf("unexpected defer stack machinery without recover, got:\n%s", ir)
	}
}

func TestPlainDeferWithoutSavedArgsIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	b.Defer(ssa.DeferAlways, callee.Expr, ssa.Builder.Call)
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("plain zero-arg defer should not allocate defer nodes, got:\n%s", ir)
	}
	if !strings.Contains(ir, "call void @callee()") {
		t.Fatalf("expected direct deferred call in IR, got:\n%s", ir)
	}
}

func TestConditionalDeferIR(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
	b := fn.MakeBody(1)
	fn.SetRecover(fn.MakeBlock())
	b.Return()
	b.SetBlockEx(fn.Block(0), ssa.BeforeLast, true)
	b.Defer(ssa.DeferInCond, callee.Expr, ssa.Builder.Call)
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "or i64") || !strings.Contains(ir, "and i64") {
		t.Fatalf("expected conditional defer bitmask operations in IR, got:\n%s", ir)
	}
}

func TestDeferContinuationDispatch(t *testing.T) {
	tests := []struct {
		name        string
		target      *ssa.Target
		switchWidth string
		wasm        bool
	}{
		{name: "linux-amd64", target: &ssa.Target{GOOS: "linux", GOARCH: "amd64"}, switchWidth: "i64"},
		{name: "darwin-arm64", target: &ssa.Target{GOOS: "darwin", GOARCH: "arm64"}, switchWidth: "i64"},
		{name: "wasip1-wasm", target: &ssa.Target{GOOS: "wasip1", GOARCH: "wasm"}, switchWidth: "i32", wasm: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := ssatest.NewProgram(t, tt.target)
			for _, field := range []int{3, 4} {
				if got := prog.Field(prog.Defer(), field); got != prog.VoidPtr() {
					t.Fatalf("runtime.Defer field %d type = %v, want unsafe.Pointer", field, got)
				}
			}
			pkg := prog.NewPackage("foo", "foo")

			callee := pkg.NewFunc("callee", ssa.NoArgsNoRet, ssa.InGo)
			cb := callee.MakeBody(1)
			cb.Return()
			cb.EndBuild()

			fn := pkg.NewFunc("main", ssa.NoArgsNoRet, ssa.InGo)
			b := fn.MakeBody(1)
			fn.SetRecover(fn.MakeBlock())
			b.Defer(ssa.DeferAlways, callee.Expr, ssa.Builder.Call)
			b.Defer(ssa.DeferAlways, callee.Expr, ssa.Builder.Call)
			b.RunDefers()
			b.RunDefers()
			b.Return()
			b.EndBuild()

			ir := pkg.Module().String()
			if tt.wasm {
				if strings.Contains(ir, "blockaddress") || strings.Contains(ir, "indirectbr") {
					t.Fatalf("wasm defer continuations must not expose block addresses:\n%s", ir)
				}
				if got := strings.Count(ir, "switch "+tt.switchWidth); got != 2 {
					t.Fatalf("got %d %s switches, want wasm RunDefers and Rethrow dispatch:\n%s", got, tt.switchWidth, ir)
				}
				if got := strings.Count(ir, "unreachable"); got < 2 {
					t.Fatalf("got %d unreachable defaults, want one per wasm defer dispatch:\n%s", got, ir)
				}
			} else {
				if !strings.Contains(ir, "blockaddress") {
					t.Fatalf("native defer continuations must retain block addresses:\n%s", ir)
				}
				if got := strings.Count(ir, "indirectbr"); got != 2 {
					t.Fatalf("got %d indirect branches, want native RunDefers and Rethrow dispatch:\n%s", got, ir)
				}
				if strings.Contains(ir, "switch "+tt.switchWidth) {
					t.Fatalf("native defer continuations must not use selector switches:\n%s", ir)
				}
			}

			// RunDefers can establish defer state before any defer statement has
			// been registered. Its Reth selector still has to dispatch to procBlk.
			empty := pkg.NewFunc("empty", ssa.NoArgsNoRet, ssa.InGo)
			eb := empty.MakeBody(1)
			empty.SetRecover(empty.MakeBlock())
			eb.Return()
			eb.SetBlockEx(empty.Block(0), ssa.BeforeLast, true)
			eb.RunDefers()
			eb.Return()
			eb.EndBuild()
		})
	}
}

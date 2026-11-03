package gotest

import (
	"runtime"
	"testing"
)

var efficientClosureRoot func(int) int

func TestCapturedLiteralClosureCallSemantics(t *testing.T) {
	base := 40
	fn := func(delta int) int {
		base += delta
		return base
	}
	base++
	efficientClosureRoot = fn
	t.Cleanup(func() { efficientClosureRoot = nil })
	evaluations := 0
	argument := func() int { evaluations++; return 1 }
	if got := fn(argument()); got != 42 || evaluations != 1 {
		t.Fatalf("captured call = %d, evaluations = %d", got, evaluations)
	}
	runtime.GC()
	if got := efficientClosureRoot(1); got != 43 || base != 43 {
		t.Fatalf("saved captured call = %d, cell = %d", got, base)
	}
}

type efficientClosureHandler struct{ fn func() int }

func (h *efficientClosureHandler) set(fn func() int) { h.fn = fn }

func TestDeferredClosureArgumentSnapshot(t *testing.T) {
	var handler efficientClosureHandler
	func() {
		fn := func() int { return 17 }
		defer handler.set(fn)
		fn = func() int { return 99 }
	}()
	if got := handler.fn(); got != 17 {
		t.Fatalf("deferred static argument = %d, want 17", got)
	}
	func() {
		value := 40
		fn := func() int { return value }
		defer handler.set(fn)
		value = 42
		runtime.GC()
	}()
	runtime.GC()
	if got := handler.fn(); got != 42 {
		t.Fatalf("deferred captured argument = %d, want 42", got)
	}
}

func TestDeferredNilClosureArgument(t *testing.T) {
	var handler efficientClosureHandler
	func() {
		var fn func() int
		defer handler.set(fn)
	}()
	if handler.fn != nil {
		t.Fatal("deferred nil function argument became nonnil")
	}
}

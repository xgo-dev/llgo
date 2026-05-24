package gotest

import (
	"runtime"
	"strings"
	"testing"
)

type fixedbug4066Panic struct{}

func fixedbug4066NamedReturn() (val int) {
	val = 0
	defer func() {
		if x := recover(); x != nil {
			_ = x.(fixedbug4066Panic)
		}
	}()
	for {
		val = 2
		fixedbug4066Throw()
	}
}

func fixedbug4066Throw() {
	panic(fixedbug4066Panic{})
}

func TestRecoverFixedbug4066NamedReturn(t *testing.T) {
	if got := fixedbug4066NamedReturn(); got != 2 {
		t.Fatalf("named return after recover = %d, want 2", got)
	}
}

func TestRecoverFixedbugDirectDeferredFuncValue(t *testing.T) {
	recovered := false
	func() {
		f := func() {
			if recover() != nil {
				recovered = true
			}
		}
		defer f()
		panic("direct deferred func value")
	}()
	if !recovered {
		t.Fatal("direct deferred func value did not recover")
	}
}

var fixedbug73916Recovered bool

func fixedbug73916CallRecover() {
	if recover() != nil {
		fixedbug73916Recovered = true
	}
}

func fixedbug73916Deferred(int) {
	fixedbug73916CallRecover()
}

func fixedbug73916MustPanic(t *testing.T, fn func()) any {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("deferred indirect recover swallowed panic")
		}
	}()
	fn()
	return nil
}

func TestRecoverFixedbug73916IndirectRecoverDoesNotRecover(t *testing.T) {
	skipBeforeGo126(t)
	fixedbug73916Recovered = false
	fixedbug73916MustPanic(t, func() {
		defer fixedbug73916Deferred(1)
		panic("fixedbug73916")
	})
	if fixedbug73916Recovered {
		t.Fatal("indirect recover returned non-nil")
	}
}

var fixedbug73916bRecovered bool

func fixedbug73916bCallRecover() {
	func() {
		if recover() != nil {
			fixedbug73916bRecovered = true
		}
	}()
}

func fixedbug73916bDeferred() int {
	fixedbug73916bCallRecover()
	return 0
}

func TestRecoverFixedbug73916NestedRecoverDoesNotRecover(t *testing.T) {
	skipBeforeGo126(t)
	fixedbug73916bRecovered = false
	fixedbug73916MustPanic(t, func() {
		defer fixedbug73916bDeferred()
		panic("fixedbug73916b")
	})
	if fixedbug73916bRecovered {
		t.Fatal("nested recover returned non-nil")
	}
}

func skipBeforeGo126(t *testing.T) {
	t.Helper()
	version := runtime.Version()
	if strings.HasPrefix(version, "go1.26") || strings.HasPrefix(version, "devel") {
		return
	}
	t.Skip("requires Go 1.26 recover semantics")
}

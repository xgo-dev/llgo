package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func immediateClosurePointer(p *int) *int { return func() *int { runtime.GC(); return p }() }

//go:noinline
func immediateClosureAddress(x int) *int { return func() *int { return &x }() }

func TestImmediateClosureLifetime(t *testing.T) {
	p := new(int)
	*p = 42
	if got := immediateClosurePointer(p); got != p || *got != 42 {
		t.Fatal("captured pointer lost")
	}
	a, b := immediateClosureAddress(7), immediateClosureAddress(9)
	runtime.GC()
	if a == b || *a != 7 || *b != 9 {
		t.Fatal("escaping capture reused")
	}
	var closures []func() int
	for i := 0; i < 16; i++ {
		x := i
		func() { x++ }()
		closures = append(closures, func() int { return x })
	}
	runtime.GC()
	for i, f := range closures {
		if f() != i+1 {
			t.Fatal("loop capture reused")
		}
	}
}

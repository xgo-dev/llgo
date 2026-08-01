package b

import "github.com/goplus/llgo/internal/build/testdata/unexportedmethodidentity/a"

type T struct{ a.T }

func (T) m() string { return "b" }

func F1(i interface{ m() string }) string { return i.m() }

//go:noinline
func F2(i interface {
	m() string
	a.I
}) string {
	return i.m()
}

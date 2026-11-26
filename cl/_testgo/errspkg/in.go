// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import (
	"errors"
	"strings"
)

type wrapErr struct {
	msg string
	err error
}

func (w *wrapErr) Error() string { return w.msg }
func (w *wrapErr) Unwrap() error { return w.err }

type myErr struct{ code int }

func (e *myErr) Error() string { return "myerr" }

func main() {
	e1 := errors.New("boom")
	println("new", e1.Error())
	e2 := errors.New("boom")
	println("is", errors.Is(e1, e1), errors.Is(e1, e2))
	w := &wrapErr{"wrap", e1}
	println("unwrap", errors.Is(w, e1))
	var target *myErr
	w2 := &wrapErr{"wrap2", &myErr{code: 7}}
	println("as", errors.As(w2, &target), target != nil, target.code)
	j := errors.Join(errors.New("a"), errors.New("b"))
	println("join", j != nil, len(j.Error()) == 3, strings.Contains(j.Error(), "\n"))
}

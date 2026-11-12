package main

import (
	"example.com/frontprobe/dep"
	"example.com/frontprobe/generic"
	"example.com/frontprobe/inline"
	"reflect"
	"strings"
)

type runner struct{ N int }

func (r runner) Run(p *int) { *p += r.N }
func crash(out *int)        { defer dep.Catch(out); panic("expected") }
func main() {
	h := dep.New(29)
	if h.Value() != 29 {
		panic("method")
	}
	var n int
	crash(&n)
	if n != 13 {
		panic("recover")
	}
	if !strings.HasSuffix(dep.Caller(), "main.go") {
		panic("caller file")
	}
	f := reflect.ValueOf(func(a int) int { return a + 3 })
	if f.Call([]reflect.Value{reflect.ValueOf(12)})[0].Int() != 15 {
		panic("reflect")
	}
	if dep.Third != 1.0/3 {
		panic("constant")
	}
	generic.Apply(runner{7}, &n)
	if inline.Sum(n, 2) != 22 {
		panic("body import")
	}
	println("frontend-ok")
}

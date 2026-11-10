package main

import (
	"example.com/frontprobe/dep"
	"example.com/frontprobe/generic"
	"example.com/frontprobe/inline"
	"reflect"
	"strings"
	"unsafe"
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
	frame := struct {
		Value    dep.Native
		Sentinel uint64
	}{Value: dep.Native{Tail: 7}, Sentinel: 123456}
	copy := (*dep.DerivedNative)(&frame.Value)
	copy.Tail = 99
	if frame.Value.Tail != 99 || frame.Sentinel != 123456 {
		panic("cached native representation")
	}
	if reflect.TypeOf(*copy).Size() != unsafe.Sizeof(*copy) || reflect.TypeOf(*copy).Field(1).Offset != unsafe.Offsetof(copy.Tail) {
		panic("cached native layout metadata")
	}
	alias := dep.NativeAlias{Tail: 17}
	derived := (*dep.DerivedAlias)(&alias)
	derived.Tail = 23
	if alias.Tail != 23 || reflect.TypeOf(*derived).Size() != unsafe.Sizeof(*derived) {
		panic("cached anonymous native layout")
	}
	println("frontend-ok")
}

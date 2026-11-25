package goescape

import (
	"testing"

	"github.com/xgo-dev/llgo/ssa"
)

func TestResolvedCallsAndCallbackCaptures(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"interface borrow", `type I interface{M()};type T struct{P *int};func(T)M(){};func f(x int){var v I=T{&x};v.M()}`, false},
		{"asserted interface", `type I interface{M()};type T struct{P *int};func(T)M(){};func f(x int){var v any=T{&x};v.(I).M()}`, false},
		{"comma-ok interface", `type I interface{M()};type T struct{P *int};func(T)M(){};func f(x int){var v any=T{&x};if i,ok:=v.(I);ok{i.M()}}`, false},
		{"interface publication", `var sink *int;type I interface{M()};type T struct{P *int};func(t T)M(){sink=t.P};func f(x int){var v I=T{&x};v.M()}`, true},
		{"interface phi", `type I interface{M()};type T struct{P *int};type U struct{P *int};func(T)M(){};func(U)M(){};func f(x int,c bool){var v I=T{&x};if c{v=U{&x}};v.M()}`, false},
		{"unknown interface phi", `type I interface{M()};type T struct{P *int};func(T)M(){};func f(x int,c bool,u I){var v I=T{&x};if c{v=u};v.M()}`, true},
		{"function phi", `func a(*int){};func b(*int){};func f(x int,c bool){fn:=a;if c{fn=b};fn(&x)}`, false},
		{"unknown function phi", `func a(*int){};func f(x int,c bool,u func(*int)){fn:=a;if c{fn=u};fn(&x)}`, true},
		{"callback borrow", `func run(fn func()){fn()};func f(x int)int{run(func(){x++});return x}`, false},
		{"callback converted signature", `type F func();func run(fn F){(func())(fn)()};func f(x int)int{run(func(){x++});return x}`, false},
		{"callback publication", `var sink *int;func run(fn func()){fn()};func f(x int){run(func(){sink=&x})}`, true},
		{"callback result publication", `var sink *int;func run(fn func()*int){sink=fn()};func f(x int){run(func()*int{return &x})}`, true},
		{"callback goroutine", `func run(fn func()){go fn()};func f(x int){run(func(){x++})}`, true},
		{"method callback", `type T struct{P *int};func(t T)M(){*t.P++};func run(fn func()){fn()};func f(x int){t:=T{&x};run(t.M)}`, false},
		{"forwarded callback borrow", `func invoke(fn func()){fn()};func relay(fn func()){invoke(fn)};func f(x int)int{relay(func(){x++});return x}`, false},
		{"forwarded callback three hops", `func a(fn func()){fn()};func b(fn func()){a(fn)};func c(fn func()){b(fn)};func f(x int)int{c(func(){x++});return x}`, false},
		{"forwarded callback with defer", `type C struct{a bool};type H struct{n int};func(h *H)expire(c *C){c.a=false;h.n++};func(h *H)invoke(c *C,fn func(*C)){defer h.expire(c);fn(c)};func(h *H)wrap(fn func(*C)){h.invoke(&C{a:true},fn)};func f(h *H)int{var x int;h.wrap(func(c *C){x=h.n});return x}`, false},
		{"forwarded callback converted signature", `type F func();func invoke(fn func()){fn()};func relay(fn F){invoke((func())(fn))};func f(x int)int{relay(func(){x++});return x}`, false},
		{"forwarded callback publication", `var sink *int;func invoke(fn func()){fn()};func relay(fn func()){invoke(fn)};func f(x int){relay(func(){sink=&x})}`, true},
		{"forwarded callback result publication", `var sink *int;func invoke(fn func()*int){sink=fn()};func relay(fn func()*int){invoke(fn)};func f(x int){relay(func()*int{return &x})}`, true},
		{"forwarded callback goroutine", `func invoke(fn func()){go fn()};func relay(fn func()){invoke(fn)};func f(x int){relay(func(){x++})}`, true},
		{"forwarded callback loop defer", `func invoke(fn func()){fn()};func relay(fn func()){for i:=0;i<2;i++{defer invoke(fn)}};func f(x int)int{relay(func(){x++});return x}`, true},
		{"forwarded callback stored in callee", `var saved func();func invoke(fn func()){saved=fn;fn()};func relay(fn func()){invoke(fn)};func f(x int){relay(func(){x++})}`, true},
		{"forwarded callback unknown caller arg", `func invoke(fn func()){fn()};func relay(fn func()){invoke(fn)};func f(x int,u func())int{fn:=func(){x++};if u!=nil{fn=u};relay(fn);return x}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, a := range sitesOf[*ssa.Alloc](fn) {
				if a.Comment == "x" {
					found = true
					if d := r.Decision(a); d.Heap != tc.heap {
						t.Errorf("x: %+v, want heap %v", d, tc.heap)
					}
				}
			}
			if !found {
				t.Fatal("missing x allocation")
			}
		})
	}
}

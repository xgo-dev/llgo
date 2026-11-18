package goescape

import (
	"testing"

	"golang.org/x/tools/go/ssa"
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
		{"callback publication", `var sink *int;func run(fn func()){fn()};func f(x int){run(func(){sink=&x})}`, true},
		{"callback result publication", `var sink *int;func run(fn func()*int){sink=fn()};func f(x int){run(func()*int{return &x})}`, true},
		{"callback goroutine", `func run(fn func()){go fn()};func f(x int){run(func(){x++})}`, true},
		{"method callback", `type T struct{P *int};func(t T)M(){*t.P++};func run(fn func()){fn()};func f(x int){t:=T{&x};run(t.M)}`, false},
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

package goescape

import (
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestCallSiteCallbackArguments(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"interface argument", `type I interface{Use(*int)int};type T struct{};func(T)Use(p *int)int{return *p};func apply(v I,p *int)int{return v.Use(p)};func f(x int)int{return apply(T{},&x)}`, false},
		{"interface retains", `var sink *int;type I interface{Use(*int)};type T struct{};func(T)Use(p *int){sink=p};func apply(v I,p *int){v.Use(p)};func f(x int){apply(T{},&x)}`, true},
		{"interface receiver contents", `var sink *int;type I interface{Use(*int)};type T struct{P *int;N int};func(t T)Use(p *int){sink=t.P};func apply(v I,p *int){v.Use(p)};func f(x int){apply(T{&x,0},nil)}`, true},
		{"borrow", `func use(p *int)int{return *p};func apply(fn func(*int)int,p *int)int{return fn(p)};func f(x int)int{return apply(use,&x)}`, false},
		{"retain", `var sink *int;func use(p *int){sink=p};func apply(fn func(*int),p *int){fn(p)};func f(x int){apply(use,&x)}`, true},
		{"closure", `func apply(fn func(*int)int,p *int)int{return fn(p)};func f(x int)int{y:=3;return apply(func(p *int)int{return *p+y},&x)}`, false},
		{"closure retains argument", `func apply(fn func(*int),p *int){fn(p)};func f(x int)func()int{var y *int;apply(func(p *int){y=p},&x);return func()int{return *y}}`, true},
		{"forward", `func use(p *int)int{return *p};func apply(fn func(*int)int,p *int)int{return fn(p)};func relay(fn func(*int)int,p *int)int{return apply(fn,p)};func f(x int)int{return relay(use,&x)}`, false},
		{"return", `func use(p *int)*int{return p};func apply(fn func(*int)*int,p *int)*int{return fn(p)};func f(x int)*int{return apply(use,&x)}`, true},
		{"return borrowed", `func use(p *int)*int{return p};func apply(fn func(*int)*int,p *int)*int{return fn(p)};func f(x int)int{return *apply(use,&x)}`, false},
		{"asynchronous", `func use(*int){};func apply(fn func(*int),p *int){go fn(p)};func f(x int){apply(use,&x)}`, true},
		{"generic heap body", `func use(**int){};func apply(fn func(**int),p *int){x:=new(*int);*x=p;fn(x)};func f(x int){apply(use,&x)}`, true},
		{"unknown", `func apply(fn func(*int),p *int){fn(p)};func f(x int,fn func(*int)){apply(fn,&x)}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, alloc := range sitesOf[*ssa.Alloc](fn) {
				if alloc.Comment == "x" {
					found = true
					if d := r.Decision(alloc); d.Heap != tc.heap {
						t.Errorf("x: %+v, want heap %v", d, tc.heap)
					}
				}
			}
			if !found {
				t.Fatal("missing x allocation")
			}
			apply := fn.Pkg.Func("apply")
			s, _ := r.Summary(apply)
			if len(s.Parameters) == 2 && s.Parameters[1].Heap == 0 {
				t.Fatal("call-site proof contaminated generic summary")
			}
		})
	}
}

func TestCopyLiveRegion(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		borrow       bool
	}{
		{"branches", `func f(b []byte,a,c string,x bool)bool{s:=string(b);if x{return s==a};return s==c}`, true},
		{"read helper", `func equal(a,b string)bool{return a==b};func f(b []byte,a string)bool{s:=string(b);return equal(s,a)}`, true},
		{"write branch", `func f(b []byte,a string,x bool)bool{s:=string(b);if x{b[0]++};return s==a}`, false},
		{"unrelated write", `var buf []byte;func f(b []byte,a string)bool{s:=string(b);buf[0]++;return s==a}`, false},
		{"helper global alias", `var buf []byte;func equal(a,b string)bool{buf[0]++;return a==b};func f(b []byte,a string)bool{s:=string(b);return equal(s,a)}`, false},
		{"helper retain", `var sink string;func equal(a,b string)bool{sink=a;return a==b};func f(b []byte,a string)bool{return equal(string(b),a)}`, false},
		{"callback alias", `func f(b []byte,a string,call func())bool{s:=string(b);call();return s==a}`, false},
		{"loop snapshot", `func f(b []byte,a string)bool{s:=string(b);for i:=0;i<2;i++{if s==a{return true};b[0]++};return false}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			for _, c := range sitesOf[*ssa.Convert](fn) {
				if d := r.Decision(c); d.ReadOnly != tc.borrow {
					t.Fatalf("%+v, want borrowed %v", d, tc.borrow)
				}
			}
		})
	}
}

func TestImmutableCapture(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		copy         bool
	}{
		{"immutable", `func f(x int)func()int{return func()int{return x}}`, true},
		{"pointer value", `func f(x *int)func()int{return func()int{return *x}}`, true},
		{"mutable", `func f(x int)func()int{return func()int{x++;return x}}`, false},
		{"after creation", `func f(x int)func()int{fn:=func()int{return x};x++;return fn}`, false},
		{"address", `func f(x int)func()*int{return func()*int{return &x}}`, false},
		{"shared", `func f(x int)(func()int,func()){return func()int{return x},func(){x++}}`, false},
		{"large", `func f(x [1024]int)func()int{return func()int{return x[0]}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			v := fn.AnonFuncs[0].FreeVars[0]
			if r.CaptureByValue(v) != tc.copy {
				t.Fatalf("copy=%v, want %v", r.CaptureByValue(v), tc.copy)
			}
		})
	}
}

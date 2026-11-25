package goescape

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestForwardedImmutableCapture(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		copy         bool
	}{
		{"scalar value", `func f(p int)func()func()int{return func()func()int{return func()int{return p}}}`, true},
		{"slice elements", `func f(p []int)func()int{return func()int{defer func(){p[0]++}();return p[0]}}`, true},
		{"deferred read", `var sink int;func f(p *int)func()int{return func()int{defer func(){sink=*p}();return *p}}`, true},
		{"pointee restore", `func f(p *int)func()int{return func()int{previous:=*p;defer func(){*p=previous}();(*p)++;return *p}}`, true},
		{"returned nested closure", `func f(p *int)func()func()int{return func()func()int{return func()int{return *p}}}`, true},
		{"read-only siblings", `func f(p *int)(func()int,func()int){return func()int{return *p},func()int{return *p+1}}`, true},
		{"nested reassignment", `func f(p *int)func()int{return func()int{defer func(){p=nil}();return *p}}`, false},
		{"nested address", `var sink **int;func f(p *int)func()int{return func()int{defer func(){sink=&p}();return *p}}`, false},
		{"address passed to helper", `func update(p **int){*p=nil};func f(p *int)func()int{return func()int{defer func(){update(&p)}();return *p}}`, false},
		{"writable sibling", `func f(p *int)(func()int,func()){return func()int{defer func(){_ = *p}();return *p},func(){p=nil}}`, false},
		{"assignment after creation", `func f(p *int)func()int{result:=func()int{defer func(){_ = *p}();return *p};p=nil;return result}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			assertCaptureGraph(t, r, "p", tc.copy)
			for _, alloc := range sitesOf[*ssa.Alloc](fn) {
				if alloc.Comment == "p" && r.Decision(alloc).Heap == tc.copy {
					t.Fatalf("p cell: %+v, copy=%v", r.Decision(alloc), tc.copy)
				}
			}
		})
	}
}

func assertCaptureGraph(t *testing.T, r *Result, name string, copy bool) {
	t.Helper()
	n := 0
	for fn := range r.summaries {
		for _, v := range fn.FreeVars {
			if v.Name() == name {
				n++
				if r.CaptureByValue(v) != copy {
					t.Errorf("%s.%s copy=%v, want %v", fn, name, r.CaptureByValue(v), copy)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("missing capture graph")
	}
}

func TestForwardedCaptureOpaqueBody(t *testing.T) {
	fn := sourceFunction(t, `func f(p *int)func()int{return func()int{defer func(){_ = *p}();return *p}}`)
	opaque := fn.AnonFuncs[0].AnonFuncs[0]
	r := Analyze([]*ssa.Function{fn}, Config{Inspect: func(fn *ssa.Function) bool { return fn != opaque }})
	assertCaptureGraph(t, r, "p", false)
}

func TestForwardedCaptureRetainsPointee(t *testing.T) {
	fn := sourceFunction(t, `func f(x int)func()int{p:=new(int);*p=x;return func()int{defer func(){(*p)++}();return *p}}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	assertCaptureGraph(t, r, "p", true)
	for _, alloc := range sitesOf[*ssa.Alloc](fn) {
		if want := alloc.Comment != "p"; r.Decision(alloc).Heap != want {
			t.Errorf("%s: %+v, want heap %v", alloc, r.Decision(alloc), want)
		}
	}
}

func TestForwardedCaptureWorkLimit(t *testing.T) {
	for _, depth := range []int{3, maxCaptureCells - 1, maxCaptureCells + 1} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			body := "return *p"
			for i := 1; i < depth; i++ {
				body = "return func()int{" + body + "}()"
			}
			fn := sourceFunction(t, "func f(p *int)func()int{return func()int{"+body+"}}")
			r := Analyze([]*ssa.Function{fn}, Config{})
			assertCaptureGraph(t, r, "p", depth < maxCaptureCells)
		})
	}
	// The limit also covers a wide graph, rather than restarting per sibling.
	fn := sourceFunction(t, "func f(p *int)[]func()int{return []func()int{"+
		strings.Repeat("func()int{return *p},", maxCaptureCells+1)+"}}")
	assertCaptureGraph(t, Analyze([]*ssa.Function{fn}, Config{}), "p", false)
}

func TestForwardedCaptureCache(t *testing.T) {
	const read = `var sink int;func f(p *int)func()int{return func()int{defer func(){sink=*p}();return *p}}`
	m := newMemoryCache()
	first := Analyze([]*ssa.Function{sourceFunction(t, read)}, Config{Cache: &m.cache})
	m.hits, m.misses = 0, 0
	second := Analyze([]*ssa.Function{sourceFunction(t, read)}, Config{Cache: &m.cache})
	if m.hits == 0 || m.misses != 0 {
		t.Fatalf("cold/hot capture facts differ: %d/%d", m.hits, m.misses)
	}
	assertSameFacts(t, first, second)

	// Only a descendant changes; ancestors must revoke their value captures too.
	write := strings.Replace(read, "sink=*p", "p=nil", 1)
	fn := sourceFunction(t, write)
	cached := Analyze([]*ssa.Function{fn}, Config{Cache: &m.cache})
	assertCaptureGraph(t, cached, "p", false)
	assertSameFacts(t, cached, Analyze([]*ssa.Function{sourceFunction(t, write)}, Config{}))
}

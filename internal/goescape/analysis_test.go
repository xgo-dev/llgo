package goescape

import (
	"go/types"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestAnalysisFactsAreImmutable(t *testing.T) {
	fn := sourceFunction(t, `func identity(p *int)*int{return p};func f(x int)*int{return identity(&x)}`)
	alloc := sitesOf[*ssa.Alloc](fn)[0]
	wasHeap := alloc.Heap
	r := Analyze([]*ssa.Function{fn}, Config{})
	if alloc.Heap != wasHeap {
		t.Fatal("analysis mutated shared Go SSA")
	}
	identity := fn.Pkg.Func("identity")
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 100 {
				summary, ok := r.Summary(identity)
				if !ok || summary.Parameters[0].Results[0] != 1 || !r.Decision(alloc).Heap {
					t.Error("concurrent reader observed changed facts")
					return
				}
				// Callers can edit an exported copy without changing the facts
				// consumed by another backend worker.
				summary.Parameters[0].Results[0] = 0
			}
		})
	}
	workers.Wait()
}

func TestAnalysisWorkLimitIsConservative(t *testing.T) {
	// A chain of nested interface boxes has quadratically many potential
	// paths. The work limit must retain all objects and discard parameter
	// proofs instead of exporting a partially solved summary.
	fn := sourceFunction(t, "func f(p *int) any { var x any=p;"+strings.Repeat("x=[1]any{x};", 1500)+"return x}")
	r := Analyze([]*ssa.Function{fn}, Config{})
	foundLimit := false
	for _, site := range sitesOf[*ssa.MakeInterface](fn) {
		d := r.Decision(site)
		if !d.Heap {
			t.Fatal("retained box moved to stack")
		}
		foundLimit = foundLimit || d.Reason == "analysis work limit"
	}
	if !foundLimit {
		t.Fatal("fixture no longer reaches the analysis work limit")
	}
	summary, ok := r.Summary(fn)
	if !ok || summary.Parameters[0].Heap != 1 {
		t.Fatalf("partial parameter proof escaped work limit: %+v", summary)
	}
}

func sitesOf[T ssa.Instruction](fn *ssa.Function) []T {
	var ret []T
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if site, ok := instr.(T); ok {
				ret = append(ret, site)
			}
		}
	}
	return ret
}

func TestInterfaceBoxAndPayload(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		heap       bool
	}{
		{"assert callback", `func f(callback func()) { any(callback).(func())() }`, false},
		{"helper callback", `func use(x any) { x.(func())() }; func f(callback func()) {use(callback)}`, false},
		{"return payload", `func f(callback func()) func() { return any(callback).(func()) }`, false},
		{"comma ok", `func f(callback func()) (func(), bool) { x, ok := any(callback).(func());return x,ok }`, false},
		{"return box", `func f(callback func()) any { return callback }`, true},
		{"publish box", `var sink any;func f(callback func()) { sink=callback }`, true},
		{"unknown", `func unknown(any);func f(callback func()) { unknown(callback) }`, true},
		{"returned alias", `func identity(x any) any{return x};func f(callback func()) any{return identity(callback)}`, true},
		{"borrowed alias", `func identity(x any) any{return x};func f(callback func()){identity(callback).(func())()}`, false},
		{"defer", `func use(x any){x.(func())()};func f(callback func()){defer use(callback)}`, false},
		{"goroutine", `func use(x any){x.(func())()};func f(callback func()){go use(callback)}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.body)
			r := Analyze([]*ssa.Function{fn}, Config{})
			boxes := sitesOf[*ssa.MakeInterface](fn)
			if len(boxes) != 1 {
				t.Fatalf("boxes=%d", len(boxes))
			}
			if d := r.Decision(boxes[0]); d.Heap != tc.heap {
				t.Fatalf("decision=%+v, want heap=%v", d, tc.heap)
			}
		})
	}
}

func TestBorrowedAndEscapingStorage(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"read", `func read(p *int)int{return *p};func f(x int)int{return read(&x)}`, false},
		{"write", `func write(p *int){*p=2};func f(x int)int{write(&x);return x}`, false},
		{"return", `func f(x int)*int{return &x}`, true},
		{"return through helper", `func identity(p *int)*int{return p};func f(x int)*int{return identity(&x)}`, true},
		{"borrow through helper", `func identity(p *int)*int{return p};func f(x int)int{return *identity(&x)}`, false},
		{"publish", `var sink *int;func f(x int){sink=&x}`, true},
		{"unknown", `func unknown(*int);func f(x int){unknown(&x)}`, true},
		{"unknown function", `func f(x int, call func(*int)){call(&x)}`, true},
		{"deferred borrow", `func read(*int){};func f(x int){defer read(&x)}`, false},
		{"deferred publish", `var sink *int;func keep(p *int){sink=p};func f(x int){defer keep(&x)}`, true},
		{"async borrow", `func read(*int){};func f(x int){go read(&x)}`, true},
		{"recursive borrow", `func read(p *int,n int)int{if n==0{return *p};return read(p,n-1)};func f(x int)int{return read(&x,4)}`, false},
		{"recursive publish", `var sink *int;func keep(p *int,n int){if n==0{sink=p;return};keep(p,n-1)};func f(x int){keep(&x,4)}`, true},
		{"mutual recursive publish", `var sink *int;func a(p *int,n int){if n==0{sink=p;return};b(p,n-1)};func b(p *int,n int){a(p,n)};func f(x int){b(&x,4)}`, true},
		{"closure returns address locally", `func f(x int)int{return *func()*int{return &x}()}`, false},
		{"closure returns address externally", `func f(x int)*int{return func()*int{return &x}()}`, true},
		{"escaping closure", `func f(x int)func()int{return func()int{return x}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			allocs := sitesOf[*ssa.Alloc](fn)
			if len(allocs) != 1 {
				t.Fatalf("allocs=%d", len(allocs))
			}
			if d := r.Decision(allocs[0]); d.Heap != tc.heap {
				t.Fatalf("decision=%+v, want heap=%v", d, tc.heap)
			}
		})
	}
}

func TestIterationStorage(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"borrow each iteration", `func use(*int){};func f(n int){for i:=0;i<n;i++ {x:=i;use(&x)}}`, false},
		{"retain preceding iteration", `func f(n int)bool{var previous *int;for i:=0;i<n;i++{current:=new(int);if previous==current{return true};previous=current};return false}`, true},
		{"defer each iteration", `func use(*int){};func f(n int){for i:=0;i<n;i++{x:=i;defer use(&x)}}`, true},
		{"append pointer", `func f(n int)[]*int{var out []*int;for i:=0;i<n;i++{x:=i;out=append(out,&x)};return out}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, alloc := range sitesOf[*ssa.Alloc](fn) {
				if _, array := alloc.Type().Underlying().(*types.Pointer).Elem().Underlying().(*types.Array); array {
					continue
				}
				found = true
				if d := r.Decision(alloc); d.Heap != tc.heap {
					t.Fatalf("%s decision=%+v, want heap=%v", alloc, d, tc.heap)
				}
			}
			if !found {
				t.Fatal("missing allocation")
			}
		})
	}
}

func TestForcedHeapPropagatesToContents(t *testing.T) {
	fn := sourceFunction(t, `func f()int{x:=1;container:=[1]*int{&x};return *container[0]}`)
	allocs := sitesOf[*ssa.Alloc](fn)
	if len(allocs) != 3 {
		t.Fatalf("allocs=%d", len(allocs))
	}
	for _, force := range []bool{false, true} {
		r := Analyze([]*ssa.Function{fn}, Config{ForceHeap: func(_ ssa.Instruction, typ types.Type) string {
			if _, array := typ.(*types.Array); force && array {
				return "backend layout restriction"
			}
			return ""
		}})
		for _, alloc := range allocs {
			if d := r.Decision(alloc); d.Heap != force {
				t.Fatalf("force=%v %s decision=%+v", force, alloc, d)
			}
		}
	}
}

func TestSummaryResultsAndEffects(t *testing.T) {
	fn := sourceFunction(t, `func f(p **int, callback func())(*int,func()){**p=2;callback();return *p,callback}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	summary, ok := r.Summary(fn)
	if !ok {
		t.Fatal("missing summary")
	}
	p, callback := summary.Parameters[0], summary.Parameters[1]
	if p.Heap != 0 || p.Mutator != 2 || p.Results[0] != 2 || p.Results[1] != 0 {
		t.Fatalf("pointer effects=%+v", p)
	}
	if callback.Callee != 1 || callback.Results[0] != 0 || callback.Results[1] != 1 {
		t.Fatalf("callback effects=%+v", callback)
	}
	// Public summaries cannot mutate facts subsequently read by another worker.
	summary.Parameters[0].Results[0] = 999
	again, _ := r.Summary(fn)
	if again.Parameters[0].Results[0] != 2 {
		t.Fatal("summary shares mutable backing storage")
	}
}

func TestResolvedBodyAndImportedSummary(t *testing.T) {
	fn := sourceFunction(t, `func helper(*int){};func f(x int){helper(&x)}`)
	alloc := sitesOf[*ssa.Alloc](fn)[0]
	for _, mode := range []string{"body", "replaced", "contract", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			config := Config{Inspect: func(callee *ssa.Function) bool { return mode == "body" || callee == fn }}
			if mode == "contract" || mode == "malformed" {
				config.External = func(callee *ssa.Function) (Summary, bool) {
					if mode == "malformed" {
						return Summary{}, true
					}
					return emptySummary(callee), true
				}
			}
			r := Analyze([]*ssa.Function{fn}, config)
			want := mode == "replaced" || mode == "malformed"
			if d := r.Decision(alloc); d.Heap != want {
				t.Fatalf("decision=%+v, want heap=%v", d, want)
			}
		})
	}
}

func TestAggregateRetention(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"local slice copy", `func f(x int)int{a:=[1]*int{&x};s:=a[:];return *s[0]}`, false},
		{"escaping slice", `func f(x int)[]*int{a:=[1]*int{&x};return a[:]}`, true},
		{"copy retains elements", `func f(x int,dst []*int){a:=[1]*int{&x};copy(dst,a[:])}`, true},
		{"heap slice elements", `func f(x int,n int)int{s:=make([]*int,n);s[0]=&x;return *s[0]}`, true},
		{"map element", `func f(x int)int{m:=make(map[int]*int);m[0]=&x;return *m[0]}`, true},
		{"map key", `func f(x int)bool{m:=make(map[*int]bool);m[&x]=true;return m[&x]}`, true},
		{"channel send", `func f(x int,c chan *int){c <- &x}`, true},
		{"select send", `func f(x int,c chan *int){select {case c <- &x:default:}}`, true},
		{"panic", `func f(x int){panic(&x)}`, true},
		{"uintptr publication", `import "unsafe";func f(x int)uintptr{return uintptr(unsafe.Pointer(&x))}`, true},
		{"unsafe borrowed view", `import "unsafe";func f(x int)int{return *(*int)(unsafe.Pointer(&x))}`, false},
		{"named pointer view", `type P *int;func use(P){};func f(x int){use(P(&x))}`, false},
		{"returned struct", `type S struct{P *int};func f(x int)S{return S{&x}}`, true},
		{"borrowed struct", `type S struct{P *int};func f(x int)int{s:=S{&x};return *s.P}`, false},
		{"array value return", `func f(x int)[1]*int{a:=[1]*int{&x};return a}`, true},
		{"slice array pointer", `func f(x int)[1]*int{a:=[1]*int{&x};s:=a[:];p:=(*[1]*int)(s);return *p}`, true},
		{"interface pointer wrapper", `type P struct{P *int};func f(x int)any{return P{&x}}`, true},
		{"interface assertion preserves box", `func f(x int)any{v:=any([2]*int{&x,nil});return v.(interface{})}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			var target *ssa.Alloc
			for _, alloc := range sitesOf[*ssa.Alloc](fn) {
				if alloc.Comment == "x" {
					target = alloc
				}
			}
			if target == nil {
				t.Fatal("missing x storage")
			}
			if decision := r.Decision(target); decision.Heap != tc.heap {
				t.Fatalf("decision=%+v, want heap=%v", decision, tc.heap)
			}
		})
	}
}

func TestAppendBorrowsArrayButRetainsElements(t *testing.T) {
	fn := sourceFunction(t, `func f(x int,dst []*int)[]*int{return append(dst,&x)}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	allocs := sitesOf[*ssa.Alloc](fn)
	if len(allocs) != 2 {
		t.Fatalf("allocs=%d", len(allocs))
	}
	for _, alloc := range allocs {
		want := alloc.Comment == "x"
		if d := r.Decision(alloc); d.Heap != want {
			t.Fatalf("%s decision=%+v, want heap=%v", alloc, d, want)
		}
	}
}

func TestIndependentResultFlows(t *testing.T) {
	fn := sourceFunction(t, `var sink *int
func pair(a,b *int)(*int,*int){return a,b}
func f(x,y int)int{a,b:=pair(&x,&y);sink=b;return *a}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	allocs := sitesOf[*ssa.Alloc](fn)
	if len(allocs) != 2 {
		t.Fatalf("allocs=%d", len(allocs))
	}
	for _, alloc := range allocs {
		want := alloc.Comment == "y"
		if d := r.Decision(alloc); d.Heap != want {
			t.Fatalf("%s decision=%+v, want heap=%v", alloc, d, want)
		}
	}
}

func TestForeignDeferOwner(t *testing.T) {
	fn := sourceFunction(t, `func consume(*int){};func f(sequence func(func(int)bool)){for x:=range sequence {defer consume(&x)}}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	seen := false
	for _, nested := range fn.AnonFuncs {
		for _, alloc := range sitesOf[*ssa.Alloc](nested) {
			if alloc.Comment == "x" {
				seen = true
				if d := r.Decision(alloc); !d.Heap {
					t.Fatalf("foreign defer owner reused yield frame: %+v", d)
				}
			}
		}
	}
	if !seen {
		t.Fatal("missing yield variable")
	}
}

func TestMethodAndGenericSummaries(t *testing.T) {
	fn := sourceFunction(t, `type T int
func (p *T) read()T{return *p}
func id[A any](p *A)*A{return p}
func f(x T)T{return id(&x).read()}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	allocs := sitesOf[*ssa.Alloc](fn)
	if len(allocs) != 1 || r.Decision(allocs[0]).Heap {
		t.Fatal("method/generic borrowing lost")
	}
}

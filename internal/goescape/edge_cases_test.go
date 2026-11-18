package goescape

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestStorageThroughAliasesAndContainers(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"local phi", `func f(x int,c bool)int{a,b:=new(*int),new(*int);p:=a;if c{p=b};*p=&x;return **p}`, false},
		{"external phi", `func f(x int,c bool,out **int){p:=new(*int);if c{p=out};*p=&x}`, true},
		{"slice store", `func f(x int)int{var a [1]*int;s:=a[:];s[0]=&x;return *s[0]}`, false},
		{"array pointer store", `func f(x int)int{var a [1]*int;s:=a[:];p:=(*[1]*int)(s);p[0]=&x;return *p[0]}`, false},
		{"named pointer store", `type P *[1]*int;func f(x int)int{var a [1]*int;p:=P(&a);p[0]=&x;return *p[0]}`, false},
		{"array value result", `func f(x int)*int{a:=[1]*int{&x};return a[0]}`, true},
		{"channel publication", `func f(x int){c:=make(chan *int,1);c<-&x;close(c)}`, true},
		{"resolved interface dispatch", `type I interface{M()};type T struct{P *int};func(T)M(){};func f(x int){var v I=T{&x};v.M()}`, false},
		{"clear local slice", `func f(x int)int{var a [1]*int;s:=a[:];s[0]=&x;clear(s);return x}`, false},
		{"self phi", `func f(x int,n int)int{a,b:=new(*int),new(*int);p:=a;for n>0{if n%2==0{p=b};*p=&x;n--};*p=&x;return **p}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, alloc := range sitesOf[*ssa.Alloc](fn) {
				if alloc.Comment == "x" {
					found = true
					if d := r.Decision(alloc); d.Heap != tc.heap {
						t.Fatalf("x: %+v, want heap %v", d, tc.heap)
					}
				}
			}
			if !found {
				t.Fatal("missing x allocation")
			}
		})
	}
}

func TestRecursiveIterationLimitDiscardsPartialEffects(t *testing.T) {
	// The discovery edge passes nil; the reverse edge propagates the pointer
	// against the SCC traversal order, by one caller per iteration. The
	// cycle exceeds the limit, including a parameter that never leaks in the
	// source, so its heap effect proves the conservative fallback was used.
	var src strings.Builder
	src.WriteString("var sink *int\n")
	const count = 70
	for i := 0; i < count; i++ {
		fmt.Fprintf(&src, "func a%02d(p,q *int,n int){", i)
		if i == 0 {
			src.WriteString("if n==0{sink=p;return};")
		}
		fmt.Fprintf(&src, "a%02d(nil,nil,n-1);a%02d(p,q,n-1)}\n", (i+1)%count, (i+count-1)%count)
	}
	src.WriteString("func f(x,y int){a00(&x,&y,100)}")
	fn := sourceFunction(t, src.String())
	r := Analyze([]*ssa.Function{fn}, Config{})
	for _, alloc := range sitesOf[*ssa.Alloc](fn) {
		if !r.Decision(alloc).Heap {
			t.Fatalf("partial recursive proof accepted for %s", alloc)
		}
	}
	summary, ok := r.Summary(fn.Pkg.Func("a00"))
	if !ok || summary.Parameters[1].Heap != 1 {
		t.Fatalf("fallback did not replace partial summaries: %+v", summary)
	}
}

// Embedding the original instruction preserves its SSA ownership/operands but
// models an opcode added by a future frontend that this analysis cannot handle.
type futureInstruction struct{ ssa.Instruction }

func TestUnknownInstructionRetainsStorageAndOperands(t *testing.T) {
	fn := sourceFunction(t, `func f(p *int)*int{x:=1; a:=[1]*int{&x}; _=a;return p}`)
	block := fn.Blocks[len(fn.Blocks)-1]
	block.Instrs[len(block.Instrs)-1] = &futureInstruction{block.Instrs[len(block.Instrs)-1]}
	r := Analyze([]*ssa.Function{fn}, Config{})
	if summary, _ := r.Summary(fn); summary.Parameters[0].Heap != 1 {
		t.Fatal("unknown instruction did not retain its pointer operand")
	}
	if len(sitesOf[*ssa.Alloc](fn)) == 0 {
		t.Fatal("missing allocation")
	}
	for _, alloc := range sitesOf[*ssa.Alloc](fn) {
		d := r.Decision(alloc)
		if !d.Heap || d.Reason != "unsupported SSA instruction" {
			t.Fatalf("future instruction accepted: %+v", d)
		}
	}
}

func TestInterfaceRepresentationContract(t *testing.T) {
	fn := sourceFunction(t, `func f(p *int)any{return p}`)
	r := Analyze([]*ssa.Function{fn}, Config{IndirectInterface: func(types.Type) bool { return true }})
	box := sitesOf[*ssa.MakeInterface](fn)[0]
	if d := r.Decision(box); !d.Heap {
		t.Fatalf("indirect representation returned a stack box: %+v", d)
	}
}

func TestInterfaceConversionAndBuiltinEffects(t *testing.T) {
	for _, src := range []string{
		`type I interface{M()};func f(p I)any{return p}`,
		`func f(p *int)int{return min(*p,max(*p,2))}`,
		`func f(p *int)int{a:=[1]int{*p};return a[0]}`,
		`func f(m map[int]*int){delete(m,1);println(len(m))}`,
		`func f[T ~string|~[]byte](p T)string{return string(p)}`,
	} {
		fn := sourceFunction(t, src)
		r := Analyze([]*ssa.Function{fn}, Config{})
		summary, ok := r.Summary(fn)
		if !ok || summary.Parameters[0].Heap != 0 {
			t.Fatalf("borrow-only operation retained parameter: %s %+v", src, summary)
		}
	}
}

func TestScopeBudgetAndIrreducibleObjects(t *testing.T) {
	fn := sourceFunction(t, `func use(*int){};func f(n int,enter bool){if enter{goto B};A: {x:=new(int);use(x)};n--;if n==0{return};B: {y:=new(int);use(y)};n--;if n!=0{goto A}}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	if len(sitesOf[*ssa.Alloc](fn)) == 0 {
		t.Fatal("missing allocation")
	}
	for _, alloc := range sitesOf[*ssa.Alloc](fn) {
		d := r.Decision(alloc)
		if !d.Heap || d.Reason != "irreducible or over-budget control flow" {
			t.Fatalf("irreducible storage accepted: %+v", d)
		}
	}
	s := functionScopes(fn)
	s.budget = 0
	s.refine(fn.Blocks, s.invocation)
	for _, block := range fn.Blocks {
		if !s.uncertain[block] {
			t.Fatal("exhausted scope budget accepted a block")
		}
	}
	single := sourceFunction(t, `func f(){for{}}`)
	if len(cyclicComponents(single.Blocks)) != 1 {
		t.Fatal("single-block backedge lost its iteration lifetime")
	}
}

func TestMissingAndMalformedSummaries(t *testing.T) {
	var missing *Result
	if _, ok := missing.LookupDecision(nil); ok {
		t.Fatal("nil result returned a placement proof")
	}
	if _, ok := missing.Summary(nil); ok {
		t.Fatal("nil result returned a call proof")
	}
	fn := sourceFunction(t, `func external(*int)*int;func f(x int)*int{return external(&x)}`)
	r := Analyze([]*ssa.Function{fn}, Config{External: func(*ssa.Function) (Summary, bool) { return Summary{Parameters: []Parameter{{}}}, true }})
	if !r.Decision(sitesOf[*ssa.Alloc](fn)[0]).Heap {
		t.Fatal("malformed result effects accepted")
	}
}

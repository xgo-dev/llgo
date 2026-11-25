package goescape

import (
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestUnsafeViewsPreserveStorage(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"Add local", `func f(x int)int{a:=[2]int{x,x+1};return *(*int)(unsafe.Add(unsafe.Pointer(&a[0]),unsafe.Sizeof(x)))}`, false},
		{"Add result", `func f(x int)unsafe.Pointer{a:=[2]int{x,x+1};return unsafe.Add(unsafe.Pointer(&a[0]),unsafe.Sizeof(x))}`, true},
		{"Slice local", `func f(x int)int{a:=[2]int{x,x+1};return unsafe.Slice(&a[0],2)[1]}`, false},
		{"Slice result", `func f(x int)[]int{a:=[2]int{x,x+1};return unsafe.Slice(&a[0],2)}`, true},
		{"SliceData local", `func f(x int)int{a:=[2]int{x,x+1};return *unsafe.SliceData(a[:])}`, false},
		{"SliceData result", `func f(x int)*int{a:=[2]int{x,x+1};return unsafe.SliceData(a[:])}`, true},
		{"String local", `func f(x byte)byte{a:=[2]byte{x,x+1};return unsafe.String(&a[0],2)[1]}`, false},
		{"String result", `func f(x byte)string{a:=[2]byte{x,x+1};return unsafe.String(&a[0],2)}`, true},
		{"StringData local", `func f(x byte)byte{a:=[2]byte{x,x+1};return *unsafe.StringData(unsafe.String(&a[0],2))}`, false},
		{"StringData result", `func f(x byte)*byte{a:=[2]byte{x,x+1};return unsafe.StringData(unsafe.String(&a[0],2))}`, true},
		{"published contents", `var sink []*int;func f(x int){a:=[1]*int{&x};sink=unsafe.Slice(&a[0],1)}`, true},
		{"unknown consumer", `func retain(unsafe.Pointer);func f(x int){a:=[2]int{x,x+1};retain(unsafe.Add(unsafe.Pointer(&a[0]),unsafe.Sizeof(x)))}`, true},
		{"integer conversion", `func f(x int)int{a:=[2]int{x,x+1};p:=uintptr(unsafe.Pointer(&a[0]));return *(*int)(unsafe.Pointer(p))}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, `import "unsafe";`+tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			allocs := sitesOf[*ssa.Alloc](fn)
			if len(allocs) == 0 {
				t.Fatal("missing source storage")
			}
			found := false
			for _, alloc := range allocs {
				// SSA also materializes a temporary composite value before
				// copying it into a; that temporary can stay local in all cases.
				if alloc.Comment != "a" && alloc.Comment != "x" {
					continue
				}
				found = true
				if d := r.Decision(alloc); d.Heap != tc.heap {
					t.Errorf("%s: %+v, want heap %v", alloc, d, tc.heap)
				}
			}
			if !found {
				t.Fatal("missing backing array or retained pointee")
			}
		})
	}
}

func TestUnsafeViewParameterEffects(t *testing.T) {
	for _, source := range []string{
		`func f(p unsafe.Pointer)unsafe.Pointer{return unsafe.Add(p,1)}`,
		`func f(p *int)[]int{return unsafe.Slice(p,1)}`,
		`func f(p []int)*int{return unsafe.SliceData(p)}`,
		`func f(p *byte)string{return unsafe.String(p,1)}`,
		`func f(p string)*byte{return unsafe.StringData(p)}`,
	} {
		fn := sourceFunction(t, `import "unsafe";`+source)
		r := Analyze([]*ssa.Function{fn}, Config{})
		summary, ok := r.Summary(fn)
		if !ok {
			t.Fatal("missing summary")
		}
		p := summary.Parameters[0]
		if p.Heap != 0 || p.Results[0] != 1 || p.Mutator != 0 || p.Callee != 0 {
			t.Fatalf("%s: view must alias its input without publication or mutation: %+v", source, p)
		}
	}
}

func TestUnsafeViewsKeepMutationEffects(t *testing.T) {
	fn := sourceFunction(t, `import "unsafe";func f(p []int){*unsafe.SliceData(p)=1}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	summary, _ := r.Summary(fn)
	if p := summary.Parameters[0]; p.Heap != 0 || p.Mutator != 1 {
		t.Fatalf("derived-pointer mutation lost: %+v", p)
	}
	fn = sourceFunction(t, `import "unsafe";func f(s string)byte{b:=[]byte(s);*unsafe.SliceData(b)=3;return b[0]}`)
	r = Analyze([]*ssa.Function{fn}, Config{})
	conversions := sitesOf[*ssa.Convert](fn)
	if len(conversions) != 1 || r.Decision(conversions[0]).ReadOnly {
		t.Fatal("unsafe mutation must not borrow immutable string bytes")
	}
}

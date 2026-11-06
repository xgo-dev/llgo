package goescape

import (
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestTransientStorageAndCopies(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		heap         bool
	}{
		{"slice borrow", `func f(n int)int{s:=make([]int,n);s[0]=2;return s[0]}`, false},
		{"slice return", `func f(n int)[]int{return make([]int,n)}`, true},
		{"map borrow", `func f(n int)int{m:=make(map[int]int);m[0]=n;return m[0]}`, false},
		{"map return", `func f()map[int]int{return make(map[int]int)}`, true},
		{"string conversion borrow", `func f(b []byte)int{s:=string(b);return len(s)}`, false},
		{"string conversion return", `func f(b []byte)string{return string(b)}`, true},
		{"bytes conversion borrow", `func f(s string)byte{b:=[]byte(s);return b[0]}`, false},
		{"bytes conversion return", `func f(s string)[]byte{return []byte(s)}`, true},
		{"runes borrow", `func f(s string)rune{r:=[]rune(s);return r[0]}`, false},
		{"runes return", `func f(s string)[]rune{return []rune(s)}`, true},
		{"runes string borrow", `func f(r []rune)int{return len(string(r))}`, false},
		{"runes string return", `func f(r []rune)string{return string(r)}`, true},
		{"concat borrow", `func f(a,b string)int{return len(a+b)}`, false},
		{"concat return", `func f(a,b string)string{return a+b}`, true},
		{"integer string borrow", `func f(n int)int{return len(string(n))}`, false},
		{"integer string return", `func f(n int)string{return string(n)}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.source)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, b := range fn.Blocks {
				for _, i := range b.Instrs {
					switch i.(type) {
					case *ssa.MakeSlice, *ssa.MakeMap, *ssa.Convert, *ssa.BinOp:
						if d, ok := r.LookupDecision(i); ok {
							found = true
							if d.Heap != tc.heap {
								t.Errorf("%s: %+v, want heap %v", i, d, tc.heap)
							}
						}
					}
				}
			}
			if !found {
				t.Fatal("missing transient allocation")
			}
			s, _ := r.Summary(fn)
			for _, p := range s.Parameters {
				if p.Heap != 0 {
					t.Errorf("copy/borrow retained source: %+v", s)
				}
				for _, e := range p.Results {
					if e != 0 {
						t.Errorf("fresh copy aliases source: %+v", s)
					}
				}
			}
		})
	}
}

func TestHybridSliceRetainsPointerElements(t *testing.T) {
	fn := sourceFunction(t, `func f(x,n int)int {s:=make([]*int,n);s[0]=&x;return *s[0]}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	for _, a := range sitesOf[*ssa.Alloc](fn) {
		if a.Comment == "x" && !r.Decision(a).Heap {
			t.Fatal("heap fallback can retain stack pointer")
		}
	}
	for _, s := range sitesOf[*ssa.MakeSlice](fn) {
		if r.Decision(s).Heap {
			t.Fatal("local slice lost small stack-buffer path")
		}
	}
}

func TestCopiedResultDoesNotRetainSourceArray(t *testing.T) {
	fn := sourceFunction(t, `func f()string{a:=[4]byte{'t','e','s','t'};return string(a[:])}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	for _, a := range sitesOf[*ssa.Alloc](fn) {
		if r.Decision(a).Heap {
			t.Errorf("copied source escapes: %+v", r.Decision(a))
		}
	}
	for _, c := range sitesOf[*ssa.Convert](fn) {
		if d, ok := r.LookupDecision(c); ok && !d.Heap {
			t.Fatal("returned string buffer moved to stack")
		}
	}
}

func TestReadOnlyStringByteProof(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		readOnly  bool
	}{
		{"read", `func f(s string)byte{return []byte(s)[0]}`, true},
		{"borrow", `func use(b []byte)byte{return b[0]};func f(s string)byte{return use([]byte(s))}`, true},
		{"write", `func f(s string)byte{b:=[]byte(s);b[0]++;return b[0]}`, false},
		{"mutator", `func use(b []byte){b[0]++};func f(s string){use([]byte(s))}`, false},
		{"callback mutator", `func run(fn func()){fn()};func f(s string){b:=[]byte(s);run(func(){b[0]++})}`, false},
		{"clear", `func f(s string){clear([]byte(s))}`, false},
		{"return", `func f(s string)[]byte{return []byte(s)}`, false},
		{"unknown", `func use([]byte);func f(s string){use([]byte(s))}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.src)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, c := range sitesOf[*ssa.Convert](fn) {
				if d, ok := r.LookupDecision(c); ok {
					found = true
					if d.ReadOnly != tc.readOnly {
						t.Errorf("%+v, want readonly %v", d, tc.readOnly)
					}
				}
			}
			if !found {
				t.Fatal("missing string conversion")
			}
		})
	}
}

func TestTemporaryByteStringProof(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		borrowed  bool
	}{
		{"compare", `func f(b []byte,s string)bool{return string(b)==s}`, true},
		{"lookup", `func f(m map[string]int,b []byte)int{return m[string(b)]}`, true},
		{"length", `func f(b []byte)int{return len(string(b))}`, true},
		{"return", `func f(b []byte)string{return string(b)}`, false},
		{"source write", `func f(b []byte,s string)bool{x:=string(b);b[0]++;return x==s}`, false},
		{"intervening call", `func change([]byte);func f(b []byte,s string)bool{x:=string(b);change(b);return x==s}`, false},
		{"multiple uses", `func f(b []byte,a,c string)bool{x:=string(b);return x==a||x==c}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn := sourceFunction(t, tc.src)
			r := Analyze([]*ssa.Function{fn}, Config{})
			found := false
			for _, c := range sitesOf[*ssa.Convert](fn) {
				if d, ok := r.LookupDecision(c); ok {
					found = true
					if d.ReadOnly != tc.borrowed {
						t.Errorf("%+v, want borrowed %v", d, tc.borrowed)
					}
				}
			}
			if !found {
				t.Fatal("missing conversion")
			}
		})
	}
}

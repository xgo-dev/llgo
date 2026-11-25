package goescape

import (
	"go/types"
	"strconv"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func testStackLayout(_ ssa.Instruction, typ types.Type) (uint64, uint64) {
	sizes := types.SizesFor("gc", "amd64")
	return uint64(sizes.Sizeof(typ)), uint64(sizes.Alignof(typ))
}

func TestFunctionStackBudgetAndRetainedContents(t *testing.T) {
	fn := sourceFunction(t, `type Box struct{P *int}
func touch(p *[16]byte){p[0]++}
func boxUse(b *Box){*b.P++}
func f(p *int){a:=new([16]byte);touch(a);b:=&Box{p};boxUse(b)}
func caller(){x:=new(int);f(x)}`)
	for _, limit := range []uint64{16, 24, 0} {
		t.Run(strconv.FormatUint(limit, 10), func(t *testing.T) {
			r := Analyze([]*ssa.Function{fn.Pkg.Func("caller")}, Config{MaxStackBytes: limit, StackLayout: testStackLayout})
			allocs := sitesOf[*ssa.Alloc](fn)
			if len(allocs) != 2 {
				t.Fatalf("unexpected storage: %v", allocs)
			}
			wantHeap := limit == 16
			if d := r.Decision(allocs[1]); d.Heap != wantHeap || (wantHeap && d.Reason != "function stack budget") {
				t.Fatalf("container decision=%+v", d)
			}
			s, _ := r.Summary(fn)
			if (s.Parameters[0].Heap != 0) != wantHeap {
				t.Fatalf("budget constraint did not reach parameter effects: %+v", s)
			}
			for _, a := range sitesOf[*ssa.Alloc](fn.Pkg.Func("caller")) {
				if r.Decision(a).Heap != wantHeap {
					t.Fatalf("caller lifetime disagrees with forced container: %+v", r.Decision(a))
				}
			}
			if limit != 0 && r.StackBytes(fn) > limit {
				t.Fatalf("reservation exceeds budget: %d", r.StackBytes(fn))
			}
		})
	}
}

func TestStackBudgetAlignmentAndOverflow(t *testing.T) {
	fn := sourceFunction(t, `func use(*[8]byte){};func f(){use(new([8]byte));use(new([8]byte));use(new([8]byte))}`)
	for _, tc := range []struct{ size, align, limit, wantStack uint64 }{
		{8, 8, 16, 2}, {7, 8, 15, 2}, {7, 8, 14, 1}, {8, 3, 64, 0}, {^uint64(0), 8, 64, 0},
	} {
		r := Analyze([]*ssa.Function{fn}, Config{MaxStackBytes: tc.limit, StackLayout: func(ssa.Instruction, types.Type) (uint64, uint64) { return tc.size, tc.align }})
		var count uint64
		for _, a := range sitesOf[*ssa.Alloc](fn) {
			if !r.Decision(a).Heap {
				count++
			}
		}
		if count != tc.wantStack || r.StackBytes(fn) > tc.limit {
			t.Fatalf("%+v: stack=%d, reserved=%d", tc, count, r.StackBytes(fn))
		}
	}
}

func TestBorrowedViewsDoNotConsumeStackBudget(t *testing.T) {
	var missing *Result
	if missing.StackBytes(nil) != 0 {
		t.Fatal("missing analysis reserves stack storage")
	}
	fn := sourceFunction(t, `func f(s string)byte{return []byte(s)[0]}`)
	r := Analyze([]*ssa.Function{fn}, Config{MaxStackBytes: 1, StackLayout: func(ssa.Instruction, types.Type) (uint64, uint64) { return 32, 1 }})
	for _, c := range sitesOf[*ssa.Convert](fn) {
		if d := r.Decision(c); d.Heap || !d.ReadOnly {
			t.Fatalf("borrowed view was forced to heap: %+v", d)
		}
	}
	if r.StackBytes(fn) != 0 {
		t.Fatal("borrowed view reserves character storage")
	}
}

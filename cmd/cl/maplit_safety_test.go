package cl

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestMapLitPublicationStoreOrdering(t *testing.T) {
	for _, timing := range []string{"before", "between", "after", "other block"} {
		t.Run(timing, func(t *testing.T) {
			var src strings.Builder
			src.WriteString("package foo\nvar published map[int]int\nfunc observe() { println(len(published)) }\nfunc F(cond bool) map[int]int { m := make(map[int]int)\n")
			if timing == "before" {
				src.WriteString("published = m\n")
			}
			for i := 0; i <= maxUnrolledMapUpdates; i++ {
				fmt.Fprintf(&src, "m[%d] = %d\n", i, i)
				if timing == "between" && i == 0 {
					src.WriteString("published = m\nobserve()\n")
				}
			}
			if timing == "after" {
				src.WriteString("published = m\n")
			}
			if timing == "other block" {
				src.WriteString("if cond { published = m }\n")
			}
			src.WriteString("return m }")
			fn := buildTestSSA(t, src.String())
			var updates []*ssa.MapUpdate
			var publication *ssa.Store
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					switch instr := instr.(type) {
					case *ssa.MapUpdate:
						updates = append(updates, instr)
					case *ssa.Store:
						if _, ok := instr.Val.(*ssa.MakeMap); ok {
							publication = instr
						}
					}
				}
			}
			if len(updates) != maxUnrolledMapUpdates+1 || publication == nil {
				t.Fatal("missing map updates or publication store")
			}
			want := timing == "after"
			if got := mapLitSafeToDelay(updates[0].Map.(*ssa.MakeMap), updates); got != want {
				t.Fatalf("mapLitSafeToDelay = %v, want %v", got, want)
			}
			plans := collectLargeMapLits(fn)
			if got := len(plans) != 0; got != want {
				t.Fatalf("collected plans = %v, want %v", got, want)
			}
		})
	}
}

func TestMapValueComplitStoreSafety(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before string
		after  string
		want   bool
	}{
		{"before load", "s.X = 1", "", true},
		{"after load", "", "s.X = side()", false},
		{"other block", "if cond { s.X = side() }", "", false},
		{"duplicate field", "s.X = side(); s.X = 2", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var src strings.Builder
			fmt.Fprintf(&src, `package foo
var effects int
func side() int { effects++; return effects }
type item struct { X int }
func F(cond bool) map[int]item {
var s item
%s
m := map[int]item{0: s}
%s
`, tc.before, tc.after)
			for i := 1; i <= maxUnrolledMapUpdates; i++ {
				fmt.Fprintf(&src, "m[%d] = item{X: %d}\n", i, i)
			}
			src.WriteString("return m }")
			fn := buildTestSSA(t, src.String())
			var first *ssa.MapUpdate
			var effect *ssa.Call
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					if update, ok := instr.(*ssa.MapUpdate); ok && first == nil {
						first = update
					}
					if call, ok := instr.(*ssa.Call); ok && call.Call.StaticCallee() == fn.Pkg.Func("side") {
						effect = call
					}
				}
			}
			if first == nil {
				t.Fatal("missing composite map update")
			}
			if _, _, _, _, got := mapValueComplit(first.Value); got != tc.want {
				t.Fatalf("mapValueComplit = %v, want %v", got, tc.want)
			}
			plans := collectLargeMapLits(fn)
			if tc.want {
				plan := firstPlan(plans)
				if plan == nil || plan.kind != mapLitStaticComplit {
					t.Fatal("safe composite did not retain static lowering")
				}
				return
			}
			if effect == nil {
				t.Fatal("missing side-effecting call in unsafe composite")
			}
			if len(plans) != 0 {
				t.Fatal("unsafe composite produced a plan that could discard stores or side effects")
			}
			p := &context{}
			p.applyMapLitPlans(plans)
			if len(p.staticInitInstrs) != 0 {
				t.Fatal("unsafe composite instructions were marked for skipping")
			}
		})
	}
}

func TestMapLitDetachedInstructions(t *testing.T) {
	fn := buildTestSSA(t, `package foo
func F() map[int]int { m := make(map[int]int); m[0] = 1; return m }
`)
	block := fn.Blocks[0]
	var update *ssa.MapUpdate
	for _, instr := range block.Instrs {
		if u, ok := instr.(*ssa.MapUpdate); ok {
			update = u
		}
	}
	if update == nil {
		t.Fatal("missing update")
	}
	original := block.Instrs
	defer func() { block.Instrs = original }()
	block.Instrs = nil
	if instrIndex(update) != -1 {
		t.Fatal("detached instruction has an index")
	}
	if mapLitSafeToDelay(update.Map.(*ssa.MakeMap), []*ssa.MapUpdate{update}) {
		t.Fatal("detached update accepted")
	}
}

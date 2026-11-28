package heap2stack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// $GOROOT/test/convinline.go creates a long boolean PHI chain. Attributor's recursive
// potential-value updates can exhaust a Windows thread's native stack while
// checking the liveness of an allocation used at the end of that chain.
func TestLongBooleanPHIChain(t *testing.T) {
	const length = 2048
	var source strings.Builder
	source.WriteString(`declare nonnull ptr @alloc(i64) allocsize(0)
declare void @observe()
declare void @consume(ptr captures(none)) nofree
define void @phi_chain(ptr %conditions) {
entry:
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  br label %check0
`)
	previous := "true"
	for i := range length {
		fmt.Fprintf(&source, `check%[1]d:
  %%slot%[1]d = getelementptr i1, ptr %%conditions, i64 %[1]d
  %%cond%[1]d = load i1, ptr %%slot%[1]d
  br i1 %%cond%[1]d, label %%bad%[1]d, label %%merge%[1]d
bad%[1]d:
  call void @observe()
  br label %%merge%[1]d
merge%[1]d:
  %%ok%[1]d = phi i1 [false, %%bad%[1]d], [%[2]s, %%check%[1]d]
`, i, previous)
		previous = fmt.Sprintf("%%ok%d", i)
		if i+1 < length {
			fmt.Fprintf(&source, "  br label %%check%d\n", i+1)
		}
	}
	fmt.Fprintf(&source, "  br i1 %s, label %%done, label %%used\n", previous)
	source.WriteString(`used:
  call void @consume(ptr %p)
  br label %done
done:
  ret void
}
define i64 @local() {
  %p = call ptr @alloc(i64 16) allockind("alloc,uninitialized")
  store i64 7, ptr %p
  %v = load i64, ptr %p
  ret i64 %v
}
`)
	for _, attrs := range [][]string{
		nil, {"noinline"}, {"alwaysinline"}, {"optsize"}, {"minsize"},
		{"optsize", "minsize"}, {"noinline", "optsize", "minsize"},
		{"alwaysinline", "optsize", "minsize"},
	} {
		name := strings.Join(attrs, "+")
		if name == "" {
			name = "none"
		}
		t.Run(name, func(t *testing.T) {
			mod := parseModule(t, source.String())
			tm := tailTarget(t, mod)
			fn := mod.NamedFunction("phi_chain")
			if !exceedsAnalysisBudget(fn) || exceedsAnalysisBudget(mod.NamedFunction("local")) {
				t.Fatal("fixture must cover both sides of the analysis budget")
			}
			wantAttributes := make(map[string]bool)
			for _, attr := range attrs {
				fn.AddFunctionAttr(mod.Context().CreateEnumAttribute(llvm.AttributeKindID(attr), 0))
				wantAttributes[attr] = true
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			// Verify the exact temporary state used during Attributor, when
			// optnone must not coexist with alwaysinline/optsize/minsize.
			entry := saveSkippedFunction(fn)
			entry.skipAnalysis(mod.Context())
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			entry.restore()
			if err := Run(mod, tm); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(fn.String(), "call ptr @alloc(") {
				t.Fatal("large function must conservatively retain heap storage")
			}
			if !fn.GetEnumFunctionAttribute(llvm.AttributeKindID("optnone")).IsNil() {
				t.Fatal("analysis exclusion leaked into the ordinary optimization pipeline")
			}
			for _, name := range []string{"noinline", "alwaysinline", "optsize", "minsize"} {
				if got := !fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil(); got != wantAttributes[name] {
					t.Fatalf("%s was not restored", name)
				}
			}
			if strings.Contains(mod.NamedFunction("local").String(), "call ptr @alloc(") {
				t.Fatal("small function lost heap-to-stack optimization")
			}
		})
	}
}

func TestPostInlineAggregateChainBudget(t *testing.T) {
	var source strings.Builder
	fmt.Fprintf(&source, "declare nonnull ptr @alloc(i64) allocsize(0)\ndeclare void @consume([%d x i8])\n", maxInsertValueDepth+1)
	cases := []struct {
		name       string
		lengths    []int
		retainHeap bool
	}{
		{"at_limit", []int{maxInsertValueDepth}, false},
		{"deep", []int{maxInsertValueDepth + 1}, true},
		{"independent", []int{maxInsertValueDepth, maxInsertValueDepth}, false},
	}
	for _, tc := range cases {
		fmt.Fprintf(&source, "define i8 @%s(i8 %%x) {\n", tc.name)
		source.WriteString("  %p = call ptr @alloc(i64 16) allockind(\"alloc,uninitialized\")\n")
		for k, n := range tc.lengths {
			prev := "poison"
			for i := range n {
				value := fmt.Sprintf("%%v%d_%d", k, i)
				fmt.Fprintf(&source, "  %s = insertvalue [%d x i8] %s, i8 %%x, %d\n", value, maxInsertValueDepth+1, prev, i)
				prev = value
			}
			fmt.Fprintf(&source, "  call void @consume([%d x i8] %s)\n", maxInsertValueDepth+1, prev)
		}
		source.WriteString("  store i8 %x, ptr %p\n  %r = load i8, ptr %p\n  ret i8 %r\n}\n")
	}
	mod := parseModule(t, source.String())
	if _, err := RunPostInline(mod, tailTarget(t, mod)); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		hasAllocation := strings.Contains(mod.NamedFunction(tc.name).String(), "call ptr @alloc(")
		if hasAllocation != tc.retainHeap {
			t.Fatalf("heap allocation in %s = %v", tc.name, hasAllocation)
		}
	}
}

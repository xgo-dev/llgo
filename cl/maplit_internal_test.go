package cl

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildTestSSA(t *testing.T, src string) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: gpackages.NewImporter(fset)}, fset,
		types.NewPackage("foo", "foo"), []*ast.File{file}, ssa.SanityCheckFunctions|ssa.InstantiateGenerics)
	if err != nil {
		t.Fatal(err)
	}
	fn := pkg.Func("F")
	if fn == nil {
		t.Fatal("missing F")
	}
	return fn
}

func firstPlan(plans map[*ssa.MapUpdate]*mapLitPlan) *mapLitPlan {
	for _, p := range plans {
		return p
	}
	return nil
}

func TestCollectLargeMapLits(t *testing.T) {
	var b strings.Builder
	b.WriteString("package foo\n\nfunc F() map[string]int {\n\treturn map[string]int{\n")
	for i := 0; i <= maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: %d,\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString("\t}\n}\n")
	plans := collectLargeMapLits(buildTestSSA(t, b.String()))
	if len(plans) != maxUnrolledMapUpdates+1 {
		t.Fatalf("collected %d MapUpdates, want %d", len(plans), maxUnrolledMapUpdates+1)
	}
}

func TestCollectLargeMapLitsSplitByBranchSkipped(t *testing.T) {
	var b strings.Builder
	b.WriteString("package foo\n\nfunc F(cond bool) map[string]int {\n\tm := make(map[string]int)\n")
	for i := 0; i < maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\tm[%q] = %d\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString("\tif cond { m[\"conditional\"] = 99 }\n")
	b.WriteString("\tm[\"last\"] = 100\n\treturn m\n}\n")
	if plans := collectLargeMapLits(buildTestSSA(t, b.String())); plans != nil {
		t.Fatalf("branch-split map literal was collected: %d", len(plans))
	}
}

func TestCollectSmallMapLitsSkipped(t *testing.T) {
	src := `package foo
func F() map[string]int {
	return map[string]int{"a": 1, "b": 2, "c": 3}
}
`
	if plans := collectLargeMapLits(buildTestSSA(t, src)); plans != nil {
		t.Fatalf("small map was collected: %d", len(plans))
	}
}

func TestCollectLargeMapLitsGenericKind(t *testing.T) {
	var b strings.Builder
	b.WriteString("package foo\n\nfunc F() map[string]int {\n\treturn map[string]int{\n")
	for i := 0; i <= maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: %d,\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString("\t}\n}\n")
	plan := firstPlan(collectLargeMapLits(buildTestSSA(t, b.String())))
	if plan == nil {
		t.Fatal("missing plan")
	}
	if plan.kind != mapLitGeneric {
		t.Fatalf("kind = %d, want generic", plan.kind)
	}
}

func TestCollectStaticComplitMapLits(t *testing.T) {
	var b strings.Builder
	b.WriteString(`package foo

type myInt int64

func boxInt(x int64) any { return myInt(x) }
func boxBool(v bool) any { return v }

func F() map[string]struct {
	Typ   string
	Value any
} {
	return map[string]struct {
		Typ   string
		Value any
	}{
`)
	for i := 0; i < maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: {\"int\", boxInt(%d)},\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString("\t\t\"k25\": {\"bool\", boxBool(true)},\n")
	b.WriteString("\t}\n}\n")
	fn := buildTestSSA(t, b.String())
	plans := collectLargeMapLits(fn)
	if len(plans) != maxUnrolledMapUpdates+1 {
		t.Fatalf("collected %d MapUpdates, want %d\n%s", len(plans), maxUnrolledMapUpdates+1, fn.String())
	}
	plan := firstPlan(plans)
	if plan.kind != mapLitStaticComplit {
		t.Fatalf("kind = %d, want static complit\n%s", plan.kind, fn.String())
	}
	if len(plan.skip) == 0 {
		t.Fatal("missing skip list")
	}
}

func TestCollectLiveIntValuesNotLowered(t *testing.T) {
	var b strings.Builder
	b.WriteString("package foo\n\nfunc F(n int) map[string]int {\n\treturn map[string]int{\n")
	for i := 0; i <= maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: n,\n", fmt.Sprintf("k%d", i))
	}
	b.WriteString("\t}\n}\n")
	if plans := collectLargeMapLits(buildTestSSA(t, b.String())); plans != nil {
		t.Fatalf("runtime int values were collected: %d", len(plans))
	}
}

func setMapLitRefs(v ssa.Value, refs ...ssa.Instruction) {
	*v.Referrers() = refs
}

func TestMapLitInstructionBoundaries(t *testing.T) {
	if instrIndex(&ssa.Store{}) != -1 {
		t.Fatal("instruction without a block was assigned an index")
	}
	if instrIsAfter(&ssa.Store{}, nil, 0) {
		t.Fatal("instruction without block was ordered")
	}
	fn := buildTestSSA(t, `package foo
func F() int { return 1 }
`)
	block := fn.Blocks[0]
	if len(block.Instrs) == 0 || instrIndex(block.Instrs[0]) < 0 {
		t.Fatal("SSA instruction was not indexed")
	}
	if instrIsAfter(block.Instrs[0], block, instrIndex(block.Instrs[0])) {
		t.Fatal("instruction was ordered after itself")
	}
}

func TestMapLitSafeToDelayRejectsInvalidUses(t *testing.T) {
	makeMap := &ssa.MakeMap{}
	update := &ssa.MapUpdate{Map: makeMap}
	block := &ssa.BasicBlock{Index: 0, Instrs: []ssa.Instruction{update}}
	_ = block
	cases := []struct {
		name string
		refs []ssa.Instruction
		want bool
	}{
		{"missing block", nil, false},
		{"foreign update", []ssa.Instruction{&ssa.MapUpdate{}}, false},
		{"wrong store", []ssa.Instruction{&ssa.Store{}}, false},
		{"before update", []ssa.Instruction{&ssa.Store{}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "missing block" {
				setMapLitRefs(makeMap, tc.refs...)
				if mapLitSafeToDelay(makeMap, []*ssa.MapUpdate{update}) {
					t.Fatal("map without a block was accepted")
				}
				return
			}
			setMapLitRefs(makeMap, tc.refs...)
			got := mapLitSafeToDelay(makeMap, []*ssa.MapUpdate{update})
			if got != tc.want {
				t.Fatalf("mapLitSafeToDelay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMapLitValueHelpersRejectMalformedSSA(t *testing.T) {
	if !allConstMapValues(nil) {
		t.Fatal("empty update list was not treated as vacuously constant")
	}
	if _, ok := constantMakeCall(&ssa.Const{}); ok {
		t.Fatal("constant was treated as a boxing call")
	}
	store := &ssa.Store{Val: &ssa.Const{}}
	if extra, ok := skipStoredValue(store); !ok || len(extra) != 0 {
		t.Fatalf("plain stored value = (%v, %v), want empty success", extra, ok)
	}
	call := &ssa.Call{}
	setMapLitRefs(call, store, &ssa.Store{})
	store.Val = call
	if _, ok := skipStoredValue(store); ok {
		t.Fatal("call with unavailable referrers was accepted")
	}
	if _, ok := constantMakeCall(call); ok {
		t.Fatal("malformed call was accepted as constant boxing")
	}
}

func TestMapValueComplitRejectsMalformedSSA(t *testing.T) {
	alloc := &ssa.Alloc{}
	load := &ssa.UnOp{Op: token.MUL, X: alloc}
	if _, _, _, _, ok := mapValueComplit(&ssa.Const{}); ok {
		t.Fatal("non-load value accepted")
	}
	setMapLitRefs(alloc, load)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("load without fields accepted")
	}
	field := &ssa.FieldAddr{X: alloc, Field: 0}
	setMapLitRefs(alloc, field, load)
	setMapLitRefs(field)
	if _, _, _, _, ok := mapValueComplit(load); ok {
		t.Fatal("field without store accepted")
	}
}

func TestCompileLargeMapLits(t *testing.T) {
	var b strings.Builder
	b.WriteString(`package foo

type item struct { Typ string; Value any }
func boxInt(x int64) any { return x }
func F() map[string]int {
	return map[string]int{
`)
	for i := 0; i <= maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: %d,\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString(`	}
}
func G() map[string]item {
	return map[string]item{
`)
	for i := 0; i <= maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: {Typ: %q, Value: boxInt(%d)},\n", fmt.Sprintf("k%d", i), "int", i)
	}
	b.WriteString("\t}\n}\n")
	_, mod := mustCompileLLPkgFromSrc(t, b.String())
	if mustNamedFunction(t, mod, "foo.F").IsNil() || mustNamedFunction(t, mod, "foo.G").IsNil() {
		t.Fatal("large map functions were not compiled")
	}
}

func TestCollectRuntimeValuesNotLowered(t *testing.T) {
	var b strings.Builder
	b.WriteString(`package foo

func boxInt(x int64) any { return x }
func boxStr(s string) any {
	if s == "" {
		return any("")
	}
	return s
}

func F() map[string]struct {
	Typ   string
	Value any
} {
	return map[string]struct {
		Typ   string
		Value any
	}{
`)
	for i := 0; i < maxUnrolledMapUpdates; i++ {
		fmt.Fprintf(&b, "\t\t%q: {\"int\", boxInt(%d)},\n", fmt.Sprintf("k%d", i), i)
	}
	b.WriteString("\t\t\"k25\": {\"str\", boxStr(\"x\")},\n")
	b.WriteString("\t}\n}\n")
	fn := buildTestSSA(t, b.String())
	if plans := collectLargeMapLits(fn); plans != nil {
		t.Fatalf("runtime-value map was collected: %d updates kind=%d\n%s",
			len(plans), firstPlan(plans).kind, fn.String())
	}
}

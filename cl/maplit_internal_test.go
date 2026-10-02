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

package goescape

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestPlacementReport(t *testing.T) {
	all := func(*ssa.Function) int { return 2 }
	selected := func(fn *ssa.Function) int {
		if fn.String() == "test.f" {
			return 2
		}
		return 0
	}
	var missing *Result
	if err := missing.WriteReport(failedWriter{}, all); err != nil {
		t.Fatalf("nil report wrote output: %v", err)
	}
	fn := sourceFunction(t, `func borrow(*int){}; func f(x int) *int {borrow(&x); return &x}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	var out bytes.Buffer
	if err := r.WriteReport(&out, selected); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "test.f: Alloc heap (returned address)") {
		t.Fatalf("missing return witness: %s", &out)
	}
	want := out.String()
	out.Reset()
	if err := r.WriteReport(&out, selected); err != nil || out.String() != want {
		t.Fatalf("unstable report: %v, %s", err, &out)
	}
	out.Reset()
	if err := r.WriteReport(&out, func(*ssa.Function) int { return 0 }); err != nil || out.Len() != 0 {
		t.Fatalf("package filter: %v, %s", err, &out)
	}
	if err := r.WriteReport(failedWriter{}, all); !errors.Is(err, errReport) {
		t.Fatalf("writer error = %v", err)
	}
}

func TestReportDistinguishesBorrowedAndFallbackStorage(t *testing.T) {
	fn := sourceFunction(t, `func f(s string,n int)int { b:=[]byte(s); c:=make([]byte,n); c[0]=b[0]; m:=make(map[int]int);m[0]=len(s);return int(c[0])+m[0]+len(string(n)) }`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	var detailed bytes.Buffer
	if err := r.WriteReport(&detailed, func(*ssa.Function) int { return 2 }); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Convert borrowed (read-only view)", "MakeSlice stack-buffer (heap fallback remains available)", "MakeMap stack (caller-owned map header; buckets may allocate)", "Convert stack-buffer (caller-owned rune encoding buffer)"} {
		if !strings.Contains(detailed.String(), want) {
			t.Errorf("missing %q:\n%s", want, &detailed)
		}
	}
	var basic bytes.Buffer
	if err := r.WriteReport(&basic, func(*ssa.Function) int { return 1 }); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(basic.String(), "(") || !strings.Contains(basic.String(), "Convert borrowed") || !strings.Contains(basic.String(), "MakeSlice stack-buffer") {
		t.Fatalf("basic report lost placements or includes detailed reasons: %s", &basic)
	}
}

var errReport = errors.New("report write")

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errReport }

func TestGraphWorkLimitRejectsPartialProof(t *testing.T) {
	g := newGraph()
	g.budget = 2
	scope := new(lifetime)
	a, b := g.location("a", scope), g.location("b", scope)
	g.flow(a, b, -1)
	g.flow(&g.heap, a, -1)
	if g.solve() {
		t.Fatal("accepted partial lifetime proof")
	}
}

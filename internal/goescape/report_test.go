package goescape

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestPlacementReport(t *testing.T) {
	var missing *Result
	if err := missing.WriteReport(failedWriter{}, ""); err != nil {
		t.Fatalf("nil report wrote output: %v", err)
	}
	fn := sourceFunction(t, `func borrow(*int){}; func f(x int) *int {borrow(&x); return &x}`)
	r := Analyze([]*ssa.Function{fn}, Config{})
	var out bytes.Buffer
	if err := r.WriteReport(&out, "test.f"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "test.f: Alloc heap (returned address)") {
		t.Fatalf("missing return witness: %s", &out)
	}
	want := out.String()
	out.Reset()
	if err := r.WriteReport(&out, "test.f"); err != nil || out.String() != want {
		t.Fatalf("unstable report: %v, %s", err, &out)
	}
	out.Reset()
	if err := r.WriteReport(&out, "other"); err != nil || out.Len() != 0 {
		t.Fatalf("package filter: %v, %s", err, &out)
	}
	if err := r.WriteReport(failedWriter{}, ""); !errors.Is(err, errReport) {
		t.Fatalf("writer error = %v", err)
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

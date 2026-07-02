package meta_test

import (
	"testing"

	"github.com/goplus/llgo/internal/meta"
)

// buildPkgMain builds a "main" package that references a symbol from "runtime"
// and converts a type to an interface defined locally.
func buildPkgMain(t *testing.T) *meta.PackageMeta {
	t.Helper()
	b := meta.NewBuilder()

	main := b.Sym("main.main")
	allocZ := b.Sym("runtime.AllocZ") // defined in runtime, referenced here
	myType := b.Sym("*main.Stringer") // defined here
	reader := b.Sym("main.Reader")    // interface defined here
	readT := b.Sym("_llgo_func$Read")

	// main calls runtime.AllocZ, converts *Stringer to Reader, calls Reader.Read
	b.AddEdge(main, allocZ, meta.EdgeOrdinary, 0)
	b.AddEdge(main, myType, meta.EdgeUseIface, 0)
	b.AddEdge(main, reader, meta.EdgeUseIfaceMethod, 0) // Reader.Read = index 0

	// Reader interface: { Read }
	b.AddIfaceMethod(reader, "Read", readT)

	// *Stringer concrete type: slot 0 = Read
	rifn := b.Sym("(*Stringer).Read$ifn")
	rtfn := b.Sym("(*Stringer).Read$tfn")
	b.AddMethodSlot(myType, "Read", readT, rifn, rtfn)

	pm, err := b.Build()
	if err != nil {
		t.Fatalf("build main: %v", err)
	}
	return pm
}

// buildPkgRuntime builds a "runtime" package that defines AllocZ.
func buildPkgRuntime(t *testing.T) *meta.PackageMeta {
	t.Helper()
	b := meta.NewBuilder()

	allocZ := b.Sym("runtime.AllocZ") // defined here, with a body edge
	mallocgc := b.Sym("runtime.mallocgc")
	b.AddEdge(allocZ, mallocgc, meta.EdgeOrdinary, 0)

	pm, err := b.Build()
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	return pm
}

func TestGlobalSummaryMerge(t *testing.T) {
	mainPkg := buildPkgMain(t)
	rtPkg := buildPkgRuntime(t)
	defer mainPkg.Close()
	defer rtPkg.Close()

	g, err := meta.NewGlobalSummary([]*meta.PackageMeta{mainPkg, rtPkg})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	sym := func(name string) meta.Symbol {
		s, ok := g.LookupSymbol(name)
		if !ok {
			t.Fatalf("LookupSymbol(%q) not found", name)
		}
		return s
	}

	main := sym("main.main")
	allocZ := sym("runtime.AllocZ")
	mallocgc := sym("runtime.mallocgc")
	myType := sym("*main.Stringer")
	reader := sym("main.Reader")

	// ── lazy OrdinaryEdges: main → runtime.AllocZ (cross-package) ──────────────
	mainEdges := g.OrdinaryEdges(main)
	if len(mainEdges) != 1 || mainEdges[0] != allocZ {
		t.Errorf("OrdinaryEdges(main) = %v, want [runtime.AllocZ=%d]", mainEdges, allocZ)
	}

	// ── allocZ's edges come from the runtime package (owner) ───────────────────
	azEdges := g.OrdinaryEdges(allocZ)
	if len(azEdges) != 1 || azEdges[0] != mallocgc {
		t.Errorf("OrdinaryEdges(allocZ) = %v, want [runtime.mallocgc=%d]", azEdges, mallocgc)
	}

	// ── UseIface: main converts *Stringer ──────────────────────────────────────
	ui := g.UseIface(main)
	if len(ui) != 1 || ui[0] != myType {
		t.Errorf("UseIface(main) = %v, want [*main.Stringer=%d]", ui, myType)
	}

	// ── UseIfaceMethod: main demands Reader.Read ───────────────────────────────
	demands := g.UseIfaceMethod(main)
	if len(demands) != 1 {
		t.Fatalf("UseIfaceMethod(main): got %d, want 1", len(demands))
	}
	if demands[0].Target != reader {
		t.Errorf("demand.Target = %d, want reader=%d", demands[0].Target, reader)
	}
	if g.Name(demands[0].Sig.Name) != "Read" {
		t.Errorf("demand.Sig.Name = %q, want \"Read\"", g.Name(demands[0].Sig.Name))
	}

	// ── MethodSlots: *Stringer has Read, name interned globally ────────────────
	slots := g.MethodSlots(myType)
	if len(slots) != 1 {
		t.Fatalf("MethodSlots(myType): got %d, want 1", len(slots))
	}
	if g.Name(slots[0].Name) != "Read" {
		t.Errorf("slot name = %q, want \"Read\"", g.Name(slots[0].Name))
	}
	// the method name "Read" must intern to the SAME global Name in both the
	// interface sig and the concrete slot, so DCE can match them.
	if slots[0].Name != demands[0].Sig.Name {
		t.Errorf("method name not unified: slot=%d demand=%d", slots[0].Name, demands[0].Sig.Name)
	}

	// ── enumeration ────────────────────────────────────────────────────────────
	if len(g.Interfaces()) != 1 || g.Interfaces()[0] != reader {
		t.Errorf("Interfaces() = %v, want [reader=%d]", g.Interfaces(), reader)
	}
	if len(g.MethodSlots(myType)) == 0 {
		t.Errorf("MethodSlots(myType) = empty, want non-empty")
	}
}

// TestGlobalSummaryLinkonce verifies first-wins for a symbol defined (with
// facts) in two packages — a linkonce type descriptor.
func TestGlobalSummaryLinkonce(t *testing.T) {
	build := func() *meta.PackageMeta {
		b := meta.NewBuilder()
		typ := b.Sym("*shared.Foo")
		child := b.Sym("shared.Bar")
		b.AddTypeChild(typ, child)
		mt := b.Sym("_llgo_func$M")
		b.AddMethodSlot(typ, "M", mt, b.Sym("ifn"), b.Sym("tfn"))
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}
	a, bp := build(), build()
	defer a.Close()
	defer bp.Close()

	g, err := meta.NewGlobalSummary([]*meta.PackageMeta{a, bp})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	foo, _ := g.LookupSymbol("*shared.Foo")

	// only one MethodInfo entry survives (first-wins), no duplicate concrete type
	if got := len(g.MethodSlots(foo)); got != 1 {
		t.Errorf("MethodSlots(foo) len = %d, want 1 (first-wins)", got)
	}
	if got := len(g.MethodSlots(foo)); got != 1 {
		t.Errorf("MethodSlots(foo) len = %d, want 1", got)
	}
	// TypeChildren resolves through the owner
	if got := len(g.TypeChildren(foo)); got != 1 {
		t.Errorf("TypeChildren(foo) len = %d, want 1", got)
	}
}

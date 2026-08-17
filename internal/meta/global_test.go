package meta_test

import (
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
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
	b.AddOrdinaryEdge(main, allocZ)
	b.AddIfaceUse(main, myType)
	b.AddIfaceMethodUse(main, reader, 0) // Reader.Read = index 0
	b.AddNamedMethodUse(main, "Close")
	b.MarkReflect(main)

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
	b.AddOrdinaryEdge(allocZ, mallocgc)

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
	if !g.HasFacts(main) {
		t.Fatal("HasFacts(main.main) = false, want true")
	}
	if got := g.SymbolName(main); got != "main.main" {
		t.Fatalf("SymbolName(main) = %q, want main.main", got)
	}

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

	// ── FuncDemands: all function-level facts share one globalized query ────────
	demands := g.FuncDemands(main)
	if len(demands) != 4 {
		t.Fatalf("FuncDemands(main): got %d, want 4", len(demands))
	}
	var ifaceMethod meta.FuncDemand
	seenUseIface, seenIfaceMethod, seenNamed, seenReflect := false, false, false, false
	for _, demand := range demands {
		switch demand.Kind {
		case meta.DemandUseIface:
			if demand.Target != myType {
				t.Errorf("UseIface target = %d, want *Stringer=%d", demand.Target, myType)
			}
			seenUseIface = true
		case meta.DemandIfaceMethod:
			if demand.Target != reader {
				t.Errorf("IfaceMethod target = %d, want reader=%d", demand.Target, reader)
			}
			if g.Name(demand.Sig.Name) != "Read" {
				t.Errorf("IfaceMethod signature name = %q, want Read", g.Name(demand.Sig.Name))
			}
			ifaceMethod = demand
			seenIfaceMethod = true
		case meta.DemandNamedMethod:
			if got := g.Name(demand.MethodName); got != "Close" {
				t.Errorf("NamedMethod name = %q, want Close", got)
			}
			seenNamed = true
		case meta.DemandReflectMethod:
			seenReflect = true
		default:
			t.Errorf("unexpected demand kind %d", demand.Kind)
		}
	}
	if !seenUseIface || !seenIfaceMethod || !seenNamed || !seenReflect {
		t.Errorf("FuncDemands missing kind: useIface=%t ifaceMethod=%t named=%t reflect=%t", seenUseIface, seenIfaceMethod, seenNamed, seenReflect)
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
	if slots[0].Name != ifaceMethod.Sig.Name {
		t.Errorf("method name not unified: slot=%d demand=%d", slots[0].Name, ifaceMethod.Sig.Name)
	}

	// ── enumeration ────────────────────────────────────────────────────────────
	if len(g.Ifaces()) != 1 || g.Ifaces()[0] != reader {
		t.Errorf("Ifaces() = %v, want [reader=%d]", g.Ifaces(), reader)
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

func TestGlobalSummaryOwnerPrefersInterfaceInfoOverOrdinaryEdges(t *testing.T) {
	descriptorPkg := func() *meta.PackageMeta {
		b := meta.NewBuilder()
		iface := b.Sym("_llgo_io.Reader")
		dep := b.Sym("runtime.descriptor")
		b.AddOrdinaryEdge(iface, dep)
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}()
	defPkg := func() *meta.PackageMeta {
		b := meta.NewBuilder()
		iface := b.Sym("_llgo_io.Reader")
		readT := b.Sym("_llgo_func$Read")
		b.AddIfaceMethod(iface, "Read", readT)
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}()
	defer descriptorPkg.Close()
	defer defPkg.Close()

	g, err := meta.NewGlobalSummary([]*meta.PackageMeta{descriptorPkg, defPkg})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	iface, ok := g.LookupSymbol("_llgo_io.Reader")
	if !ok {
		t.Fatal("LookupSymbol(_llgo_io.Reader) not found")
	}
	methods := g.IfaceMethods(iface)
	if len(methods) != 1 {
		t.Fatalf("IfaceMethods(_llgo_io.Reader) len = %d, want 1", len(methods))
	}
	if got := g.Name(methods[0].Name); got != "Read" {
		t.Fatalf("IfaceMethods(_llgo_io.Reader)[0].Name = %q, want Read", got)
	}
}

func TestGlobalSummaryOwnerPrefersFuncDemandOverOrdinaryEdges(t *testing.T) {
	refPkg := func() *meta.PackageMeta {
		b := meta.NewBuilder()
		fn := b.Sym("pkg.use")
		b.AddOrdinaryEdge(fn, b.Sym("pkg.stale"))
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}()
	defPkg := func() *meta.PackageMeta {
		b := meta.NewBuilder()
		fn := b.Sym("pkg.use")
		b.AddOrdinaryEdge(fn, b.Sym("pkg.live"))
		b.AddIfaceUse(fn, b.Sym("pkg.T"))
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}()
	defer refPkg.Close()
	defer defPkg.Close()

	g, err := meta.NewGlobalSummary([]*meta.PackageMeta{refPkg, defPkg})
	if err != nil {
		t.Fatal(err)
	}
	fn, _ := g.LookupSymbol("pkg.use")
	live, _ := g.LookupSymbol("pkg.live")
	typ, _ := g.LookupSymbol("pkg.T")

	edges := g.OrdinaryEdges(fn)
	if len(edges) != 1 || edges[0] != live {
		t.Fatalf("OrdinaryEdges(pkg.use) = %v, want [%d]", edges, live)
	}
	demands := g.FuncDemands(fn)
	if len(demands) != 1 || demands[0].Kind != meta.DemandUseIface || demands[0].Target != typ {
		t.Fatalf("FuncDemands(pkg.use) = %+v, want UseIface(%d)", demands, typ)
	}
}

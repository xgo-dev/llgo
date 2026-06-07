package metadata

import (
	"reflect"
	"testing"
)

func TestGlobalSummaryMergesLocalIDsByText(t *testing.T) {
	pkgA, refsA := buildGlobalSummaryPkgA()
	pkgB, refsB := buildGlobalSummaryPkgB()

	if refsA.intType == refsB.intType {
		t.Fatal("test setup should use different local Symbol IDs for _llgo_int")
	}

	summary, err := NewGlobalSummary([]*PackageMeta{pkgA, pkgB})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	intSymA, ok := summary.LookupSymbol("_llgo_int")
	if !ok {
		t.Fatal("LookupSymbol(_llgo_int) failed")
	}
	if got := summary.SymbolName(intSymA); got != "_llgo_int" {
		t.Fatalf("SymbolName(_llgo_int) = %q", got)
	}

	funcType, ok := summary.LookupSymbol("_llgo_func$X")
	if !ok {
		t.Fatal("LookupSymbol(_llgo_func$X) failed")
	}
	mainA := mustLookupSymbol(t, summary, "pkg/a.main")
	useA := mustLookupSymbol(t, summary, "pkg/a.use")
	typeB := mustLookupSymbol(t, summary, "_llgo_pkg/b.T")
	ifaceB := mustLookupSymbol(t, summary, "_llgo_iface$B")
	ifnB := mustLookupSymbol(t, summary, "pkg/b.(*T).M")
	tfnB := mustLookupSymbol(t, summary, "pkg/b.T.M")

	if got := summary.OrdinaryEdges(mainA); !reflect.DeepEqual(got, []Symbol{useA}) {
		t.Fatalf("OrdinaryEdges(pkg/a.main) = %#v, want %#v", got, []Symbol{useA})
	}
	if got := summary.TypeChildren(typeB); !reflect.DeepEqual(got, []Symbol{intSymA}) {
		t.Fatalf("TypeChildren(_llgo_pkg/b.T) = %#v, want %#v", got, []Symbol{intSymA})
	}
	if got := summary.UseIface(mainA); !reflect.DeepEqual(got, []Symbol{typeB}) {
		t.Fatalf("UseIface(pkg/a.main) = %#v, want %#v", got, []Symbol{typeB})
	}

	methodName := summary.UseNamedMethod(useA)
	if len(methodName) != 1 || summary.Name(methodName[0]) != "M" {
		t.Fatalf("UseNamedMethod(pkg/a.use) = %#v, want name M", methodName)
	}
	if _, ok := summary.LookupSymbol("M"); ok {
		t.Fatal("LookupSymbol found method name that only appeared as Name")
	}

	wantSig := MethodSig{Name: methodName[0], MType: funcType}
	if got := summary.InterfaceMethods(ifaceB); !reflect.DeepEqual(got, []MethodSig{wantSig}) {
		t.Fatalf("InterfaceMethods(_llgo_iface$B) = %#v, want %#v", got, []MethodSig{wantSig})
	}
	if got := summary.UseIfaceMethod(useA); !reflect.DeepEqual(got, []IfaceMethodDemand{{Target: ifaceB, Sig: wantSig}}) {
		t.Fatalf("UseIfaceMethod(pkg/a.use) = %#v", got)
	}
	if got := summary.MethodSlots(typeB); !reflect.DeepEqual(got, []MethodSlot{{Sig: wantSig, IFn: ifnB, TFn: tfnB}}) {
		t.Fatalf("MethodSlots(_llgo_pkg/b.T) = %#v", got)
	}
	if !summary.HasReflectMethod(useA) {
		t.Fatal("HasReflectMethod(pkg/a.use) = false, want true")
	}

	if got := summary.Interfaces(); !reflect.DeepEqual(got, []Symbol{ifaceB}) {
		t.Fatalf("Interfaces() = %#v, want %#v", got, []Symbol{ifaceB})
	}
	if got := summary.ConcreteTypes(); !reflect.DeepEqual(got, []Symbol{typeB}) {
		t.Fatalf("ConcreteTypes() = %#v, want %#v", got, []Symbol{typeB})
	}
}

func TestGlobalSummaryNameLookupsReturnEmptyStringForOutOfRangeIDs(t *testing.T) {
	summary, err := NewGlobalSummary([]*PackageMeta{buildGlobalDuplicateFactsPackage()})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	if got := summary.SymbolName(Symbol(len(summary.stringTable))); got != "" {
		t.Fatalf("SymbolName(out-of-range) = %q, want empty string", got)
	}
	if got := summary.Name(Name(len(summary.stringTable))); got != "" {
		t.Fatalf("Name(out-of-range) = %q, want empty string", got)
	}
}

func TestGlobalSummaryDeduplicatesMergedFacts(t *testing.T) {
	pkgA := buildGlobalDuplicateFactsPackage()
	pkgB := buildGlobalDuplicateFactsPackage()

	summary, err := NewGlobalSummary([]*PackageMeta{pkgA, pkgB})
	if err != nil {
		t.Fatalf("NewGlobalSummary: %v", err)
	}

	main := mustLookupSymbol(t, summary, "pkg.main")
	use := mustLookupSymbol(t, summary, "pkg.use")
	typ := mustLookupSymbol(t, summary, "_llgo_pkg.T")
	iface := mustLookupSymbol(t, summary, "_llgo_iface$I")
	funcType := mustLookupSymbol(t, summary, "_llgo_func$M")
	ifn := mustLookupSymbol(t, summary, "pkg.(*T).M")
	tfn := mustLookupSymbol(t, summary, "pkg.T.M")
	methodNames := summary.UseNamedMethod(use)
	if len(methodNames) != 1 {
		t.Fatalf("UseNamedMethod len = %d (%#v), want 1", len(methodNames), methodNames)
	}
	wantSig := MethodSig{Name: methodNames[0], MType: funcType}

	if got := summary.OrdinaryEdges(main); !reflect.DeepEqual(got, []Symbol{use}) {
		t.Fatalf("OrdinaryEdges(pkg.main) = %#v, want %#v", got, []Symbol{use})
	}
	if got := summary.TypeChildren(typ); !reflect.DeepEqual(got, []Symbol{funcType}) {
		t.Fatalf("TypeChildren(_llgo_pkg.T) = %#v, want %#v", got, []Symbol{funcType})
	}
	if got := summary.InterfaceMethods(iface); !reflect.DeepEqual(got, []MethodSig{wantSig}) {
		t.Fatalf("InterfaceMethods(_llgo_iface$I) = %#v, want %#v", got, []MethodSig{wantSig})
	}
	if got := summary.UseIface(main); !reflect.DeepEqual(got, []Symbol{typ}) {
		t.Fatalf("UseIface(pkg.main) = %#v, want %#v", got, []Symbol{typ})
	}
	wantDemand := IfaceMethodDemand{Target: iface, Sig: wantSig}
	if got := summary.UseIfaceMethod(use); !reflect.DeepEqual(got, []IfaceMethodDemand{wantDemand}) {
		t.Fatalf("UseIfaceMethod(pkg.use) = %#v, want %#v", got, []IfaceMethodDemand{wantDemand})
	}
	if got := summary.MethodSlots(typ); !reflect.DeepEqual(got, []MethodSlot{{Sig: wantSig, IFn: ifn, TFn: tfn}}) {
		t.Fatalf("MethodSlots(_llgo_pkg.T) = %#v", got)
	}
}

func TestGlobalSummaryRejectsConflictingMethodSlots(t *testing.T) {
	pkgA, _ := buildGlobalSummaryPkgB()

	// Rebuild pkgB with the same type and slot signature but a different TFn.
	b := NewBuilder()
	typ := b.Symbol("_llgo_pkg/b.T")
	methodName := b.Name("M")
	funcType := b.Symbol("_llgo_func$X")
	ifn := b.Symbol("pkg/b.(*T).M")
	tfn := b.Symbol("pkg/b.T.M.conflict")
	b.AddMethodInfo(typ, []MethodSlot{{
		Sig: MethodSig{Name: methodName, MType: funcType},
		IFn: ifn,
		TFn: tfn,
	}})
	pkgB := b.Build()

	if _, err := NewGlobalSummary([]*PackageMeta{pkgA, pkgB}); err == nil {
		t.Fatal("NewGlobalSummary accepted conflicting MethodInfo for the same type")
	}
}

func buildGlobalDuplicateFactsPackage() *PackageMeta {
	b := NewBuilder()
	main := b.Symbol("pkg.main")
	use := b.Symbol("pkg.use")
	typ := b.Symbol("_llgo_pkg.T")
	iface := b.Symbol("_llgo_iface$I")
	methodName := b.Name("M")
	funcType := b.Symbol("_llgo_func$M")
	ifn := b.Symbol("pkg.(*T).M")
	tfn := b.Symbol("pkg.T.M")
	sig := MethodSig{Name: methodName, MType: funcType}

	b.AddEdge(main, use)
	b.AddTypeChild(typ, funcType)
	b.AddIfaceEntry(iface, []MethodSig{sig})
	b.AddUseIface(main, []Symbol{typ})
	b.AddUseIfaceMethod(use, []IfaceMethodDemand{{Target: iface, Sig: sig}})
	b.AddUseNamedMethod(use, []Name{methodName})
	b.AddMethodInfo(typ, []MethodSlot{{Sig: sig, IFn: ifn, TFn: tfn}})
	return b.Build()
}

type globalSummaryRefs struct {
	intType Symbol
}

func buildGlobalSummaryPkgA() (*PackageMeta, globalSummaryRefs) {
	b := NewBuilder()
	main := b.Symbol("pkg/a.main")
	use := b.Symbol("pkg/a.use")
	methodName := b.Name("M")
	typ := b.Symbol("_llgo_pkg/b.T")
	iface := b.Symbol("_llgo_iface$B")
	funcType := b.Symbol("_llgo_func$X")
	intType := b.Symbol("_llgo_int")

	b.AddEdge(main, use)
	b.AddUseIface(main, []Symbol{typ})
	b.AddUseIfaceMethod(use, []IfaceMethodDemand{{
		Target: iface,
		Sig:    MethodSig{Name: methodName, MType: funcType},
	}})
	b.AddUseNamedMethod(use, []Name{methodName})
	b.AddReflectMethod(use)

	return b.Build(), globalSummaryRefs{intType: intType}
}

func buildGlobalSummaryPkgB() (*PackageMeta, globalSummaryRefs) {
	b := NewBuilder()
	other := b.Symbol("pkg/b.other")
	intType := b.Symbol("_llgo_int")
	typ := b.Symbol("_llgo_pkg/b.T")
	methodName := b.Name("M")
	funcType := b.Symbol("_llgo_func$X")
	iface := b.Symbol("_llgo_iface$B")
	ifn := b.Symbol("pkg/b.(*T).M")
	tfn := b.Symbol("pkg/b.T.M")

	b.AddEdge(other, intType)
	b.AddTypeChild(typ, intType)
	b.AddIfaceEntry(iface, []MethodSig{{Name: methodName, MType: funcType}})
	b.AddMethodInfo(typ, []MethodSlot{{
		Sig: MethodSig{Name: methodName, MType: funcType},
		IFn: ifn,
		TFn: tfn,
	}})

	return b.Build(), globalSummaryRefs{intType: intType}
}

func mustLookupSymbol(t *testing.T, summary *GlobalSummary, name string) Symbol {
	t.Helper()
	sym, ok := summary.LookupSymbol(name)
	if !ok {
		t.Fatalf("LookupSymbol(%q) failed", name)
	}
	return sym
}

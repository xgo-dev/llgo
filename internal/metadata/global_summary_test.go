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

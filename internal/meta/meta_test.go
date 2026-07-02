package meta

import (
	"os"
	"testing"
	"unsafe"
)

// TestWireLayout verifies the zero-copy structs match their on-disk byte layout:
// correct total size and field offsets. If these drift, unsafe reinterpretation
// of mmap bytes would silently corrupt — so we assert them explicitly.
func TestWireLayout(t *testing.T) {
	if got := unsafe.Sizeof(Edge{}); got != 12 {
		t.Errorf("sizeof(Edge) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(Edge{}.Target); got != 0 {
		t.Errorf("Edge.Target offset = %d, want 0", got)
	}
	if got := unsafe.Offsetof(Edge{}.Extra); got != 4 {
		t.Errorf("Edge.Extra offset = %d, want 4", got)
	}
	if got := unsafe.Offsetof(Edge{}.Kind); got != 8 {
		t.Errorf("Edge.Kind offset = %d, want 8", got)
	}

	if got := unsafe.Sizeof(MethodSlot{}); got != 20 {
		t.Errorf("sizeof(MethodSlot) = %d, want 20", got)
	}
	if got := unsafe.Offsetof(MethodSlot{}.MType); got != 8 {
		t.Errorf("MethodSlot.MType offset = %d, want 8", got)
	}
	if got := unsafe.Offsetof(MethodSlot{}.TFn); got != 16 {
		t.Errorf("MethodSlot.TFn offset = %d, want 16", got)
	}

	if got := unsafe.Sizeof(MethodSig{}); got != 12 {
		t.Errorf("sizeof(MethodSig) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(MethodSig{}.MType); got != 8 {
		t.Errorf("MethodSig.MType offset = %d, want 8", got)
	}
}

// TestTypeChildrenAlignment uses symbol names of irregular total length so the
// string table is unlikely to land on a 4-byte boundary on its own, verifying
// that stringTable padding keeps the zero-copy TypeChildren view correctly aligned.
func TestTypeChildrenAlignment(t *testing.T) {
	for _, pad := range []string{"a", "ab", "abc", "abcd", "abcde"} {
		b := NewBuilder()
		// a symbol whose name length varies, to shift the string table size
		b.Sym("x." + pad)
		parent := b.Sym("*pkg.Parent")
		c0 := b.Sym("pkg.C0")
		c1 := b.Sym("pkg.C1")
		c2 := b.Sym("pkg.C2")
		b.AddTypeChild(parent, c0)
		b.AddTypeChild(parent, c1)
		b.AddTypeChild(parent, c2)

		pm, err := b.Build()
		if err != nil {
			t.Fatalf("pad=%q build: %v", pad, err)
		}
		got := pm.typeChildren(parent)
		want := []LocalSymbol{c0, c1, c2}
		if len(got) != len(want) {
			t.Fatalf("pad=%q TypeChildren len = %d, want %d", pad, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("pad=%q child[%d] = %d, want %d", pad, i, got[i], want[i])
			}
		}
	}
}

// TestRoundTrip builds a small package summary, serializes it, then reads it
// back and verifies every query returns the expected values.
func TestRoundTrip(t *testing.T) {
	b := NewBuilder()

	// symbols
	main := b.Sym("main.main")
	helper := b.Sym("main.helper")
	allocZ := b.Sym("runtime.AllocZ")
	myType := b.Sym("*_llgo_main.MyStruct")
	myField := b.Sym("_llgo_main.Inner")
	myIface := b.Sym("_llgo_iface$Reader")
	mtype := b.Sym("_llgo_func$Read")
	ifn := b.Sym("(*MyStruct).Read$ifn")
	tfn := b.Sym("(*MyStruct).Read$tfn")

	// ordinary edges
	b.AddEdge(main, helper, EdgeOrdinary, 0)
	b.AddEdge(main, allocZ, EdgeOrdinary, 0)

	// interface conversion
	b.AddEdge(main, myType, EdgeUseIface, 0)

	// interface method call: Reader.Read is method index 0
	b.AddEdge(main, myIface, EdgeUseIfaceMethod, 0)

	// named method call
	b.AddNamedMethodEdge(helper, "ServeHTTP")

	// TypeChildren: *MyStruct contains Inner
	b.AddTypeChild(myType, myField)

	// MethodInfo for *MyStruct: slot 0 = Read
	b.AddMethodSlot(myType, "Read", mtype, ifn, tfn)

	// InterfaceInfo for Reader: method 0 = Read
	b.AddIfaceMethod(myIface, "Read", mtype)

	// reflect
	b.MarkReflect(helper)

	// build
	pm, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// ── verify Symbols ────────────────────────────────────────────────────────

	checkName := func(sym LocalSymbol, want string) {
		t.Helper()
		if got := pm.symbolName(sym); got != want {
			t.Errorf("SymbolName(%d) = %q, want %q", sym, got, want)
		}
	}
	checkName(main, "main.main")
	checkName(helper, "main.helper")
	checkName(allocZ, "runtime.AllocZ")
	checkName(myType, "*_llgo_main.MyStruct")

	// ── verify Edges ──────────────────────────────────────────────────────────

	mainEdges := pm.edges(main)
	if len(mainEdges) != 4 {
		t.Fatalf("Edges(main): got %d edges, want 4", len(mainEdges))
	}
	if e := mainEdges[0]; e.Kind != EdgeOrdinary || LocalSymbol(e.Target) != helper {
		t.Errorf("edge[0] = %+v, want {Target:%d Kind:Ordinary}", e, helper)
	}
	if e := mainEdges[1]; e.Kind != EdgeOrdinary || LocalSymbol(e.Target) != allocZ {
		t.Errorf("edge[1] = %+v, want {Target:%d Kind:Ordinary}", e, allocZ)
	}
	if e := mainEdges[2]; e.Kind != EdgeUseIface || LocalSymbol(e.Target) != myType {
		t.Errorf("edge[2] = %+v, want {Target:%d Kind:UseIface}", e, myType)
	}
	if e := mainEdges[3]; e.Kind != EdgeUseIfaceMethod || LocalSymbol(e.Target) != myIface || e.Extra != 0 {
		t.Errorf("edge[3] = %+v, want {Target:%d Kind:UseIfaceMethod Extra:0}", e, myIface)
	}

	helperEdges := pm.edges(helper)
	if len(helperEdges) != 1 {
		t.Fatalf("Edges(helper): got %d, want 1", len(helperEdges))
	}
	if e := helperEdges[0]; e.Kind != EdgeUseNamedMethod {
		t.Errorf("helper edge[0].Kind = %d, want UseNamedMethod", e.Kind)
	}
	// For UseNamedMethod, target=NameRef.Off and extra=NameRef.Len.
	gotName := pm.nameString(NameRef{Off: helperEdges[0].Target, Len: helperEdges[0].Extra})
	if gotName != "ServeHTTP" {
		t.Errorf("UseNamedMethod target name = %q, want \"ServeHTTP\"", gotName)
	}
	if got := pm.edges(allocZ); len(got) != 0 {
		t.Errorf("Edges(allocZ): got %d, want 0", len(got))
	}

	// ── verify TypeChildren ───────────────────────────────────────────────────

	children := pm.typeChildren(myType)
	if len(children) != 1 || children[0] != myField {
		t.Errorf("TypeChildren(myType) = %v, want [%d]", children, myField)
	}
	if pm.typeChildren(main) != nil {
		t.Errorf("TypeChildren(main) should be nil")
	}
	if pm.ntypeChild(myType) == 0 {
		t.Errorf("NTypeChild(myType) = 0, want >0")
	}
	if pm.ntypeChild(main) > 0 {
		t.Errorf("NTypeChild(main) > 0, want 0")
	}

	// ── verify MethodSlots ────────────────────────────────────────────────────

	slots := pm.methodSlots(myType)
	if len(slots) != 1 {
		t.Fatalf("MethodSlots(myType): got %d, want 1", len(slots))
	}
	slot := slots[0]
	if pm.nameString(slot.Name) != "Read" {
		t.Errorf("slot.Name = %q, want \"Read\"", pm.nameString(slot.Name))
	}
	if slot.MType != mtype || slot.IFn != ifn || slot.TFn != tfn {
		t.Errorf("slot = %+v, unexpected symbols", slot)
	}
	if len(pm.methodSlots(myType)) == 0 {
		t.Errorf("MethodSlots(myType) = empty, want non-empty")
	}

	// ── verify IfaceMethods ───────────────────────────────────────────────────

	sigs := pm.ifaceMethods(myIface)
	if len(sigs) != 1 {
		t.Fatalf("IfaceMethods(myIface): got %d, want 1", len(sigs))
	}
	if pm.nameString(sigs[0].Name) != "Read" {
		t.Errorf("iface method name = %q, want \"Read\"", pm.nameString(sigs[0].Name))
	}
	if pm.nifaceMethod(myIface) == 0 {
		t.Errorf("NIfaceMethod(myIface) = 0, want >0")
	}
	if pm.nifaceMethod(main) > 0 {
		t.Errorf("NIfaceMethod(main) > 0, want 0")
	}

	// ── verify ReflectBitmap ──────────────────────────────────────────────────

	if !pm.hasReflect(helper) {
		t.Errorf("HasReflect(helper) = false, want true")
	}
	if pm.hasReflect(main) {
		t.Errorf("HasReflect(main) = true, want false")
	}
}

// TestRoundTripFile writes the meta to disk and reads it back via ReadMeta.
func TestRoundTripFile(t *testing.T) {
	b := NewBuilder()
	fn := b.Sym("pkg.Fn")
	dep := b.Sym("runtime.X")
	b.AddEdge(fn, dep, EdgeOrdinary, 0)

	pm, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	path := t.TempDir() + "/test.meta"
	if err := os.WriteFile(path, pm.Bytes(), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	pm2, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	defer pm2.Close()

	if got := pm2.symbolName(fn); got != "pkg.Fn" {
		t.Errorf("SymbolName after file round-trip = %q, want \"pkg.Fn\"", got)
	}
	edges := pm2.edges(fn)
	if len(edges) != 1 || LocalSymbol(edges[0].Target) != dep {
		t.Errorf("Edges after file round-trip = %v", edges)
	}
}

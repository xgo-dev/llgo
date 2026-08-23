package meta

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestDemandFunctionNames(t *testing.T) {
	b := NewBuilder()
	main := b.Sym("pkg.main")
	reflectFn := b.Sym("pkg.reflectFn")
	typ := b.Sym("_llgo_pkg.T")
	b.AddIfaceUse(main, typ)
	b.MarkReflect(reflectFn)
	b.AddOrdinaryEdge(b.Sym("pkg.helper"), typ)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"pkg.main", "pkg.reflectFn"}
	if got := pm.DemandFunctionNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DemandFunctionNames() = %#v, want %#v", got, want)
	}
	if got := (*PackageMeta)(nil).DemandFunctionNames(); got != nil {
		t.Fatalf("nil DemandFunctionNames() = %#v, want nil", got)
	}
}

// TestWireLayout verifies the zero-copy structs match their on-disk byte layout:
// correct total size and field offsets. If these drift, unsafe reinterpretation
// of mmap bytes would silently corrupt — so we assert them explicitly.
func TestWireLayout(t *testing.T) {
	if got := unsafe.Sizeof(localFuncDemand{}); got != 12 {
		t.Errorf("sizeof(localFuncDemand) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(localFuncDemand{}.Kind); got != 0 {
		t.Errorf("localFuncDemand.Kind offset = %d, want 0", got)
	}
	if got := unsafe.Offsetof(localFuncDemand{}.Target); got != 4 {
		t.Errorf("localFuncDemand.Target offset = %d, want 4", got)
	}
	if got := unsafe.Offsetof(localFuncDemand{}.Extra); got != 8 {
		t.Errorf("localFuncDemand.Extra offset = %d, want 8", got)
	}

	if got := unsafe.Sizeof(localMethodSlot{}); got != 20 {
		t.Errorf("sizeof(localMethodSlot) = %d, want 20", got)
	}
	if got := unsafe.Offsetof(localMethodSlot{}.MType); got != 8 {
		t.Errorf("localMethodSlot.MType offset = %d, want 8", got)
	}
	if got := unsafe.Offsetof(localMethodSlot{}.TFn); got != 16 {
		t.Errorf("localMethodSlot.TFn offset = %d, want 16", got)
	}

	if got := unsafe.Sizeof(localMethodSig{}); got != 12 {
		t.Errorf("sizeof(localMethodSig) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(localMethodSig{}.MType); got != 8 {
		t.Errorf("localMethodSig.MType offset = %d, want 8", got)
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
		want := []Symbol{c0, c1, c2}
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
	b.AddOrdinaryEdge(main, helper)
	b.AddOrdinaryEdge(main, allocZ)

	// interface conversion
	b.AddIfaceUse(main, myType)

	// interface method call: Reader.Read is method index 0
	b.AddIfaceMethodUse(main, myIface, 0)

	// named method call
	b.AddNamedMethodUse(helper, "ServeHTTP")

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

	checkName := func(sym Symbol, want string) {
		t.Helper()
		if got := pm.symbolName(sym); got != want {
			t.Errorf("SymbolName(%d) = %q, want %q", sym, got, want)
		}
	}
	checkName(main, "main.main")
	checkName(helper, "main.helper")
	checkName(allocZ, "runtime.AllocZ")
	checkName(myType, "*_llgo_main.MyStruct")

	// ── verify OrdinaryEdges / FuncDemand ─────────────────────────────────────

	mainEdges := pm.ordinaryEdges(main)
	if len(mainEdges) != 2 {
		t.Fatalf("OrdinaryEdges(main): got %d edges, want 2", len(mainEdges))
	}
	if mainEdges[0] != helper {
		t.Errorf("ordinary[0] = %d, want helper=%d", mainEdges[0], helper)
	}
	if mainEdges[1] != allocZ {
		t.Errorf("ordinary[1] = %d, want allocZ=%d", mainEdges[1], allocZ)
	}

	mainDemands := pm.funcDemands(main)
	if len(mainDemands) != 2 {
		t.Fatalf("FuncDemand(main): got %d demands, want 2", len(mainDemands))
	}
	if d := mainDemands[0]; d.Kind != DemandUseIface || Symbol(d.Target) != myType {
		t.Errorf("demand[0] = %+v, want {Kind:UseIface Target:%d}", d, myType)
	}
	if d := mainDemands[1]; d.Kind != DemandIfaceMethod || Symbol(d.Target) != myIface || d.Extra != 0 {
		t.Errorf("demand[1] = %+v, want {Kind:IfaceMethod Target:%d Extra:0}", d, myIface)
	}

	helperDemands := pm.funcDemands(helper)
	if len(helperDemands) != 2 {
		t.Fatalf("FuncDemand(helper): got %d, want 2", len(helperDemands))
	}
	if d := helperDemands[0]; d.Kind != DemandNamedMethod {
		t.Errorf("helper demand[0].Kind = %d, want NamedMethod", d.Kind)
	}
	// For UseNamedMethod, target=nameRef.Off and extra=nameRef.Len.
	gotName := pm.nameString(nameRef{Off: helperDemands[0].Target, Len: helperDemands[0].Extra})
	if gotName != "ServeHTTP" {
		t.Errorf("UseNamedMethod target name = %q, want \"ServeHTTP\"", gotName)
	}
	if d := helperDemands[1]; d.Kind != DemandReflectMethod {
		t.Errorf("helper demand[1].Kind = %d, want ReflectMethod", d.Kind)
	}
	if got := pm.ordinaryEdges(allocZ); len(got) != 0 {
		t.Errorf("OrdinaryEdges(allocZ): got %d, want 0", len(got))
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

}

// TestRoundTripFile writes the meta to disk and reads it back via Open.
func TestRoundTripFile(t *testing.T) {
	b := NewBuilder()
	fn := b.Sym("pkg.Fn")
	dep := b.Sym("runtime.X")
	b.AddOrdinaryEdge(fn, dep)

	pm, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	path := t.TempDir() + "/test.meta"
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := pm.WriteTo(f); err != nil {
		f.Close()
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	pm2, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pm2.Close()

	if got := pm2.symbolName(fn); got != "pkg.Fn" {
		t.Errorf("SymbolName after file round-trip = %q, want \"pkg.Fn\"", got)
	}
	edges := pm2.ordinaryEdges(fn)
	if len(edges) != 1 || edges[0] != dep {
		t.Errorf("OrdinaryEdges after file round-trip = %v", edges)
	}
}

func TestOpenErrors(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		if _, err := Open(filepath.Join(t.TempDir(), "missing.meta")); err == nil {
			t.Fatal("Open succeeded for a missing file")
		}
	})

	t.Run("mmap", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.meta")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path); err == nil || !strings.Contains(err.Error(), "meta: mmap") {
			t.Fatalf("Open error = %v, want mmap error", err)
		}
	})

	tests := []struct {
		name string
		raw  func() []byte
		want string
	}{
		{
			name: "magic",
			raw: func() []byte {
				raw := make([]byte, headerSize)
				copy(raw, "NOPE")
				return raw
			},
			want: "meta: bad magic",
		},
		{
			name: "version",
			raw: func() []byte {
				raw := make([]byte, headerSize)
				copy(raw, magic)
				binary.LittleEndian.PutUint32(raw[4:8], version+1)
				return raw
			},
			want: "meta: unsupported version 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".meta")
			if err := os.WriteFile(path, tt.raw(), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(path); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Open error = %v, want %q", err, tt.want)
			}
		})
	}
}

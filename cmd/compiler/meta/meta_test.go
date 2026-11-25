package meta

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

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
	b.AddIfaceUse(fn, dep)

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
	if got := pm2.symbolName(fn); got != "pkg.Fn" {
		t.Errorf("SymbolName after file round-trip = %q, want \"pkg.Fn\"", got)
	}
	edges := pm2.ordinaryEdges(fn)
	if len(edges) != 1 || edges[0] != dep {
		t.Errorf("OrdinaryEdges after file round-trip = %v", edges)
	}
	demands := pm2.funcDemands(fn)
	if len(demands) != 1 || demands[0].Kind != DemandUseIface || Symbol(demands[0].Target) != dep {
		t.Errorf("FuncDemand after file round-trip = %v", demands)
	}
	if err := pm2.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := pm2.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestOpenErrors(t *testing.T) {
	t.Run("short in-memory header", func(t *testing.T) {
		if _, err := newPackageMeta(make([]byte, headerSize-1)); err == nil || !strings.Contains(err.Error(), "meta: file too small") {
			t.Fatalf("newPackageMeta error = %v, want short-file error", err)
		}
	})

	t.Run("oversized in-memory metadata", func(t *testing.T) {
		size := uint64(^uint32(0)) + 1
		if err := validateMetaSize(size); err == nil || !strings.Contains(err.Error(), "meta: file too large") {
			t.Fatalf("validateMetaSize error = %v, want large-file error", err)
		}
	})

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

	validRaw := validationMetaBytes(t)
	cloneValid := func() []byte { return append([]byte(nil), validRaw...) }
	tests := []struct {
		name string
		raw  func() []byte
		want string
	}{
		{
			name: "short header",
			raw: func() []byte {
				return make([]byte, headerSize-1)
			},
			want: "meta: mmap",
		},
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
		{
			name: "section offset past end",
			raw: func() []byte {
				raw := validEmptyMetaHeader(headerSize)
				binary.LittleEndian.PutUint32(raw[8+secIfaceInfo*4:], headerSize+4)
				return raw
			},
			want: "meta: invalid section 6 offset 40",
		},
		{
			name: "section offsets out of order",
			raw: func() []byte {
				raw := validEmptyMetaHeader(headerSize + 4)
				binary.LittleEndian.PutUint32(raw[8+secSymbols*4:], headerSize+4)
				binary.LittleEndian.PutUint32(raw[8+secOrdinaryEdges*4:], headerSize)
				return raw
			},
			want: "meta: invalid section 2 offset 36",
		},
		{
			name: "unaligned section offset",
			raw: func() []byte {
				raw := validEmptyMetaHeader(headerSize + 4)
				binary.LittleEndian.PutUint32(raw[8+secSymbols*4:], headerSize+1)
				return raw
			},
			want: "meta: invalid section 1 offset 37",
		},
		{
			name: "truncated symbols section",
			raw: func() []byte {
				return validEmptyMetaHeader(headerSize)
			},
			want: "meta: truncated symbols section",
		},
		{
			name: "symbol count exceeds section",
			raw: func() []byte {
				raw := cloneValid()
				symOff := metaSectionOffset(raw, secSymbols)
				nsyms := binary.LittleEndian.Uint32(raw[symOff:])
				binary.LittleEndian.PutUint32(raw[symOff:], nsyms+1)
				return raw
			},
			want: "meta: invalid symbols section size",
		},
		{
			name: "truncated csr offsets",
			raw: func() []byte {
				raw := cloneValid()
				ordinaryOff := metaSectionOffset(raw, secOrdinaryEdges)
				binary.LittleEndian.PutUint32(raw[8+secFuncDemand*4:], ordinaryOff+4)
				return raw
			},
			want: "meta: truncated OrdinaryEdges CSR header",
		},
		{
			name: "csr symbol count mismatch",
			raw: func() []byte {
				raw := cloneValid()
				ordinaryOff := metaSectionOffset(raw, secOrdinaryEdges)
				nsyms := binary.LittleEndian.Uint32(raw[ordinaryOff:])
				binary.LittleEndian.PutUint32(raw[ordinaryOff:], nsyms+1)
				return raw
			},
			want: "meta: OrdinaryEdges has",
		},
		{
			name: "descending csr offsets",
			raw: func() []byte {
				raw := cloneValid()
				ordinaryOff := metaSectionOffset(raw, secOrdinaryEdges)
				offsetsBase := ordinaryOff + 4
				binary.LittleEndian.PutUint32(raw[offsetsBase+4:], 1)
				binary.LittleEndian.PutUint32(raw[offsetsBase+8:], 0)
				return raw
			},
			want: "meta: invalid OrdinaryEdges CSR offset",
		},
		{
			name: "csr offset past data",
			raw: func() []byte {
				raw := cloneValid()
				ordinaryOff := metaSectionOffset(raw, secOrdinaryEdges)
				nsyms := binary.LittleEndian.Uint32(raw[ordinaryOff:])
				offsetsBase := ordinaryOff + 4
				last := offsetsBase + nsyms*4
				nrecords := binary.LittleEndian.Uint32(raw[last:])
				binary.LittleEndian.PutUint32(raw[last:], nrecords+1)
				return raw
			},
			want: "meta: invalid OrdinaryEdges CSR offset",
		},
		{
			name: "symbol name past string table",
			raw: func() []byte {
				raw := cloneValid()
				strOff := metaSectionOffset(raw, secStringTable)
				symOff := metaSectionOffset(raw, secSymbols)
				binary.LittleEndian.PutUint32(raw[symOff+4:], symOff-strOff)
				return raw
			},
			want: "meta: Symbols record 0 has invalid name range",
		},
		{
			name: "demand name past string table",
			raw: func() []byte {
				raw := cloneValid()
				strOff := metaSectionOffset(raw, secStringTable)
				symOff := metaSectionOffset(raw, secSymbols)
				dataOff := metaCSRDataOffset(raw, secFuncDemand)
				binary.LittleEndian.PutUint32(raw[dataOff+4:], symOff-strOff)
				return raw
			},
			want: "meta: FuncDemand record 0 has invalid name range",
		},
		{
			name: "method name past string table",
			raw: func() []byte {
				raw := cloneValid()
				strOff := metaSectionOffset(raw, secStringTable)
				symOff := metaSectionOffset(raw, secSymbols)
				dataOff := metaCSRDataOffset(raw, secMethodInfo)
				binary.LittleEndian.PutUint32(raw[dataOff:], symOff-strOff)
				return raw
			},
			want: "meta: MethodInfo record 0 has invalid name range",
		},
		{
			name: "interface name past string table",
			raw: func() []byte {
				raw := cloneValid()
				strOff := metaSectionOffset(raw, secStringTable)
				symOff := metaSectionOffset(raw, secSymbols)
				dataOff := metaCSRDataOffset(raw, secIfaceInfo)
				binary.LittleEndian.PutUint32(raw[dataOff:], symOff-strOff)
				return raw
			},
			want: "meta: InterfaceInfo record 0 has invalid name range",
		},
		{
			name: "ordinary edge symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secOrdinaryEdges):], metaSymbolCount(raw))
				return raw
			},
			want: "meta: OrdinaryEdges record 0 has invalid symbol",
		},
		{
			name: "interface-use demand symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				dataOff := metaCSRDataOffset(raw, secFuncDemand)
				binary.LittleEndian.PutUint32(raw[dataOff+12+4:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: FuncDemand record 1 has invalid target symbol",
		},
		{
			name: "interface-method demand symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				dataOff := metaCSRDataOffset(raw, secFuncDemand)
				binary.LittleEndian.PutUint32(raw[dataOff+2*12+4:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: FuncDemand record 2 has invalid target symbol",
		},
		{
			name: "type child symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secTypeChildren):], metaSymbolCount(raw))
				return raw
			},
			want: "meta: TypeChildren record 0 has invalid symbol",
		},
		{
			name: "method type symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secMethodInfo)+8:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: MethodInfo record 0 has invalid MType symbol",
		},
		{
			name: "interface function symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secMethodInfo)+12:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: MethodInfo record 0 has invalid IFn symbol",
		},
		{
			name: "type function symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secMethodInfo)+16:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: MethodInfo record 0 has invalid TFn symbol",
		},
		{
			name: "interface method type symbol past table",
			raw: func() []byte {
				raw := cloneValid()
				binary.LittleEndian.PutUint32(raw[metaCSRDataOffset(raw, secIfaceInfo)+8:], metaSymbolCount(raw))
				return raw
			},
			want: "meta: InterfaceInfo record 0 has invalid MType symbol",
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

func TestValidateCSRSectionErrors(t *testing.T) {
	tests := []struct {
		name  string
		raw   []byte
		nsyms uint32
		want  string
	}{
		{
			name: "misaligned data size",
			raw:  make([]byte, 9),
			want: "invalid Test data size",
		},
		{
			name: "nonzero first offset",
			raw: func() []byte {
				raw := make([]byte, 8)
				binary.LittleEndian.PutUint32(raw[4:], 1)
				return raw
			}(),
			want: "Test CSR first offset is 1, want 0",
		},
		{
			name: "terminal offset before data end",
			raw: func() []byte {
				raw := make([]byte, 16)
				binary.LittleEndian.PutUint32(raw, 1)
				return raw
			}(),
			nsyms: 1,
			want:  "Test CSR covers 0 records, section contains 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateCSRSection(tt.raw, "Test", 0, uint32(len(tt.raw)), tt.nsyms, 4)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateCSRSection error = %v, want %q", err, tt.want)
			}
		})
	}
}

func validationMetaBytes(t *testing.T) []byte {
	t.Helper()
	b := NewBuilder()
	src := b.Sym("pkg.src")
	dst := b.Sym("pkg.dst")
	iface := b.Sym("pkg.iface")
	mtype := b.Sym("pkg.mtype")
	ifn := b.Sym("pkg.ifn")
	tfn := b.Sym("pkg.tfn")
	b.AddOrdinaryEdge(src, dst)
	b.AddNamedMethodUse(src, "Method")
	b.AddIfaceUse(src, dst)
	b.AddIfaceMethodUse(src, iface, 0)
	b.AddTypeChild(src, dst)
	b.AddMethodSlot(src, "Method", mtype, ifn, tfn)
	b.AddIfaceMethod(iface, "Method", mtype)
	pm, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), pm.raw...)
}

func metaSectionOffset(raw []byte, section int) uint32 {
	return binary.LittleEndian.Uint32(raw[8+section*4:])
}

func metaCSRDataOffset(raw []byte, section int) uint32 {
	sectionOff := metaSectionOffset(raw, section)
	nsyms := binary.LittleEndian.Uint32(raw[sectionOff:])
	return sectionOff + 4 + (nsyms+1)*4
}

func metaSymbolCount(raw []byte) uint32 {
	return binary.LittleEndian.Uint32(raw[metaSectionOffset(raw, secSymbols):])
}

func validEmptyMetaHeader(size int) []byte {
	raw := make([]byte, size)
	copy(raw, magic)
	binary.LittleEndian.PutUint32(raw[4:8], version)
	for sec := range numSections {
		binary.LittleEndian.PutUint32(raw[8+sec*4:], headerSize)
	}
	return raw
}

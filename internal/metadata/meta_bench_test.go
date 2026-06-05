package metadata

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

// scale defines the size parameters for a test PackageMeta.
type scale struct {
	numSymbols   int // string table entries
	numTypes     int // types (with slots, children, used in iface)
	numFunctions int // functions
	numMethods   int // methods per type (avg)
}

var scales = map[string]scale{
	"Small":  {numSymbols: 200, numTypes: 30, numFunctions: 40, numMethods: 3},
	"Medium": {numSymbols: 1000, numTypes: 150, numFunctions: 200, numMethods: 4},
	"Large":  {numSymbols: 5000, numTypes: 500, numFunctions: 800, numMethods: 5},
	"XL":     {numSymbols: 20000, numTypes: 2000, numFunctions: 3000, numMethods: 5},
}

// genMeta generates a realistic PackageMeta for benchmarking.
func genMeta(s scale) *PackageMeta {
	rng := rand.New(rand.NewSource(42))

	// Generate string table entries
	strings := make([]string, s.numSymbols)
	strings[0] = "main.main"
	for i := 1; i < s.numSymbols; i++ {
		strings[i] = randomSymName(rng, i)
	}

	meta := NewPackageMeta(strings)

	typeStart := s.numFunctions + 1
	funcStart := 1

	// OrdinaryEdges: functions reference symbols
	for f := range s.numFunctions {
		src := funcStart + f
		ndsts := 2 + rng.Intn(8) // 2-9 references
		dsts := make([]Symbol, 0, ndsts)
		for range ndsts {
			dst := rng.Intn(s.numSymbols)
			if dst != src {
				dsts = append(dsts, dst)
			}
		}
		if len(dsts) > 0 {
			meta.OrdinaryEdges[src] = dsts
		}
	}
	// Also edges from types to their element types
	for t := range s.numTypes {
		typ := typeStart + t
		nchildren := 1 + rng.Intn(3)
		children := make([]Symbol, 0, nchildren)
		for range nchildren {
			child := typeStart + rng.Intn(s.numTypes)
			if child != typ {
				children = append(children, child)
			}
			// Also add ordinary edge from type to child
			if len(children) > 0 {
				meta.OrdinaryEdges[typ] = append(meta.OrdinaryEdges[typ], child)
			}
		}
		if len(children) > 0 {
			meta.TypeChildren[typ] = children
		}
	}

	// InterfaceInfo: some types are interfaces
	numIfaces := s.numTypes / 5
	for range numIfaces {
		iface := typeStart + rng.Intn(s.numTypes)
		if _, ok := meta.InterfaceInfo[iface]; ok {
			continue
		}
		nm := 1 + rng.Intn(s.numMethods)
		methods := make([]MethodSig, nm)
		for j := range nm {
			methods[j] = MethodSig{
				Name:  rng.Intn(s.numSymbols),
				MType: rng.Intn(s.numSymbols),
			}
		}
		meta.InterfaceInfo[iface] = methods
	}

	// UseIface: functions convert types to interfaces
	for f := range s.numFunctions {
		if rng.Intn(4) == 0 { // 25% of functions
			owner := funcStart + f
			ntypes := 1 + rng.Intn(3)
			types := make([]Symbol, ntypes)
			for j := range ntypes {
				types[j] = typeStart + rng.Intn(s.numTypes)
			}
			meta.UseIface[owner] = types
		}
	}

	// UseIfaceMethod: functions call interface methods
	for f := range s.numFunctions {
		if rng.Intn(5) == 0 { // 20% of functions
			owner := funcStart + f
			nd := 1 + rng.Intn(2)
			demands := make([]IfaceMethodDemand, nd)
			for j := range nd {
				demands[j] = IfaceMethodDemand{
					Target: typeStart + rng.Intn(s.numTypes),
					Sig: MethodSig{
						Name:  rng.Intn(s.numSymbols),
						MType: rng.Intn(s.numSymbols),
					},
				}
			}
			meta.UseIfaceMethod[owner] = demands
		}
	}

	// MethodInfo: every concrete type has methods
	for t := range s.numTypes {
		typ := typeStart + t
		if _, isIface := meta.InterfaceInfo[typ]; isIface {
			continue // skip interfaces
		}
		nm := 1 + rng.Intn(s.numMethods)
		slots := make([]MethodSlot, nm)
		for j := range nm {
			slots[j] = MethodSlot{
				Sig: MethodSig{
					Name:  rng.Intn(s.numSymbols),
					MType: rng.Intn(s.numSymbols),
				},
				IFn: rng.Intn(s.numSymbols),
				TFn: rng.Intn(s.numSymbols),
			}
		}
		meta.MethodInfo[typ] = slots
	}

	// UseNamedMethod
	for f := range s.numFunctions {
		if rng.Intn(8) == 0 { // ~12%
			owner := funcStart + f
			nnames := 1 + rng.Intn(2)
			names := make([]Symbol, nnames)
			for j := range nnames {
				names[j] = rng.Intn(s.numSymbols)
			}
			meta.UseNamedMethod[owner] = names
		}
	}

	// ReflectMethod
	for f := range s.numFunctions {
		if rng.Intn(10) == 0 { // 10%
			meta.ReflectMethod[funcStart+f] = struct{}{}
		}
	}

	return meta
}

func randomSymName(rng *rand.Rand, i int) string {
	pkgIdx := rng.Intn(5)
	switch rng.Intn(8) {
	case 0:
		return fmt.Sprintf("*_llgo_pkg%d.Type%d", pkgIdx, i)
	case 1:
		return fmt.Sprintf("_llgo_pkg%d.Type%d", pkgIdx, i)
	case 2:
		return fmt.Sprintf("(*pkg%d.Type%d).Method%d", pkgIdx, i, rng.Intn(10))
	case 3:
		return fmt.Sprintf("_llgo_iface$%d", i)
	case 4:
		return fmt.Sprintf("_llgo_func$%d", i)
	case 5:
		return fmt.Sprintf("pkg%d.Func%d", pkgIdx, i)
	case 6:
		return fmt.Sprintf("runtime.Method%d", i)
	default:
		return fmt.Sprintf("pkg%d.Var%d", pkgIdx, i)
	}
}

// ---- correctness test ----

func TestRoundTrip(t *testing.T) {
	for name, s := range scales {
		t.Run(name, func(t *testing.T) {
			orig := genMeta(s)
			var buf bytes.Buffer
			if err := orig.WriteMeta(&buf); err != nil {
				t.Fatalf("encode: %v", err)
			}
			decoded, err := ReadMeta(&buf)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if err := metaEqual(orig, decoded); err != nil {
				t.Fatalf("mismatch: %v", err)
			}
		})
	}
}

func metaEqual(a, b *PackageMeta) error {
	if len(a.stringTable) != len(b.stringTable) {
		return fmt.Errorf("stringTable len: %d vs %d", len(a.stringTable), len(b.stringTable))
	}
	for i := range a.stringTable {
		if a.stringTable[i] != b.stringTable[i] {
			return fmt.Errorf("stringTable[%d]: %q vs %q", i, a.stringTable[i], b.stringTable[i])
		}
	}
	if err := mapEqual("OrdinaryEdges", a.OrdinaryEdges, b.OrdinaryEdges); err != nil {
		return err
	}
	if err := mapSliceEqual("TypeChildren", a.TypeChildren, b.TypeChildren); err != nil {
		return err
	}
	if err := mapMethodSigEqual("InterfaceInfo", a.InterfaceInfo, b.InterfaceInfo); err != nil {
		return err
	}
	if err := mapSliceEqual("UseIface", a.UseIface, b.UseIface); err != nil {
		return err
	}
	if err := mapDemandEqual("UseIfaceMethod", a.UseIfaceMethod, b.UseIfaceMethod); err != nil {
		return err
	}
	if err := mapSlotEqual("MethodInfo", a.MethodInfo, b.MethodInfo); err != nil {
		return err
	}
	if err := mapSliceEqual("UseNamedMethod", a.UseNamedMethod, b.UseNamedMethod); err != nil {
		return err
	}
	if err := setEqual("ReflectMethod", a.ReflectMethod, b.ReflectMethod); err != nil {
		return err
	}
	return nil
}

func mapEqual(name string, a, b map[Symbol][]Symbol) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s len: %d vs %d", name, len(a), len(b))
	}
	for k, va := range a {
		vb := b[k]
		if len(va) != len(vb) {
			return fmt.Errorf("%s[%d] len: %d vs %d", name, k, len(va), len(vb))
		}
		for i := range va {
			if va[i] != vb[i] {
				return fmt.Errorf("%s[%d][%d]: %d vs %d", name, k, i, va[i], vb[i])
			}
		}
	}
	return nil
}

func mapSliceEqual[T comparable](name string, a, b map[Symbol][]T) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s len: %d vs %d", name, len(a), len(b))
	}
	for k, va := range a {
		vb := b[k]
		if len(va) != len(vb) {
			return fmt.Errorf("%s[%d] len: %d vs %d", name, k, len(va), len(vb))
		}
		for i := range va {
			if va[i] != vb[i] {
				return fmt.Errorf("%s[%d][%d]: %v vs %v", name, k, i, va[i], vb[i])
			}
		}
	}
	return nil
}

func mapMethodSigEqual(name string, a, b map[Symbol][]MethodSig) error {
	return mapSliceEqual(name, a, b)
}

func mapDemandEqual(name string, a, b map[Symbol][]IfaceMethodDemand) error {
	return mapSliceEqual(name, a, b)
}

func mapSlotEqual(name string, a, b map[Symbol][]MethodSlot) error {
	return mapSliceEqual(name, a, b)
}

func setEqual(name string, a, b map[Symbol]struct{}) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s len: %d vs %d", name, len(a), len(b))
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return fmt.Errorf("%s: %d in a but not b", name, k)
		}
	}
	return nil
}

// ---- benchmarks ----

func BenchmarkEncode(b *testing.B) {
	for name, s := range scales {
		b.Run(name, func(b *testing.B) {
			meta := genMeta(s)
			data := make([]byte, 0, estimateSize(meta))
			b.ReportMetric(float64(estimateSize(meta)), "est-filesize")
			b.ResetTimer()
			for b.Loop() {
				var buf bytes.Buffer
				buf.Grow(estimateSize(meta))
				_ = meta.WriteMeta(&buf)
				data = buf.Bytes()[:0]
			}
			_ = data
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	for name, s := range scales {
		b.Run(name, func(b *testing.B) {
			meta := genMeta(s)
			var buf bytes.Buffer
			if err := meta.WriteMeta(&buf); err != nil {
				b.Fatalf("encode: %v", err)
			}
			encoded := buf.Bytes()
			b.ReportMetric(float64(len(encoded)), "filesize-B")
			b.ResetTimer()
			for b.Loop() {
				_, _ = ReadMeta(bytes.NewReader(encoded))
			}
		})
	}
}

func estimateSize(meta *PackageMeta) int {
	n := 4 + 10 // magic + version ~14 bytes
	// string table
	n += 10 + len(meta.stringTable)*2
	for _, s := range meta.stringTable {
		n += len(s)
	}
	// each map: count + per-entry overhead ~ 5 bytes per entry, + 5 per symbol
	n += estimateMapSizeSlice(meta.OrdinaryEdges)
	n += estimateMapSizeSlice(meta.TypeChildren)
	n += estimateInterfaceInfoSize(meta.InterfaceInfo)
	n += estimateMapSizeSlice(meta.UseIface)
	n += estimateUseIfaceMethodSize(meta.UseIfaceMethod)
	n += estimateMethodInfoSize(meta.MethodInfo)
	n += estimateMapSizeSlice(meta.UseNamedMethod)
	n += estimateSetSize(meta.ReflectMethod)
	return n * 2 // generous overhead
}

func estimateMapSizeSlice(m map[Symbol][]Symbol) int {
	n := 5
	for k, v := range m {
		n += 5 + 5 + 5*len(v)
		_ = k
	}
	return n
}

func estimateInterfaceInfoSize(m map[Symbol][]MethodSig) int {
	n := 5
	for _, v := range m {
		n += 5 + 5 + 10*len(v)
	}
	return n
}

func estimateUseIfaceMethodSize(m map[Symbol][]IfaceMethodDemand) int {
	n := 5
	for _, v := range m {
		n += 5 + 5 + 15*len(v)
	}
	return n
}

func estimateMethodInfoSize(m map[Symbol][]MethodSlot) int {
	n := 5
	for _, v := range m {
		n += 5 + 5 + 20*len(v)
	}
	return n
}

func estimateSetSize(m map[Symbol]struct{}) int {
	n := 5
	for range m {
		n += 5
	}
	return n
}

// ---- v2 benchmarks ----

// memWriteSeeker is an in-memory io.WriteSeeker for benchmark fairness (no disk I/O).
type memWriteSeeker struct {
	buf []byte
	pos int
}

func (m *memWriteSeeker) Write(p []byte) (int, error) {
	end := m.pos + len(p)
	if end > len(m.buf) {
		m.buf = append(m.buf, make([]byte, end-len(m.buf))...)
	}
	n := copy(m.buf[m.pos:], p)
	m.pos += n
	return n, nil
}

func (m *memWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.pos = int(offset)
	case io.SeekCurrent:
		m.pos += int(offset)
	case io.SeekEnd:
		m.pos = len(m.buf) + int(offset)
	}
	return int64(m.pos), nil
}

func (m *memWriteSeeker) Bytes() []byte { return m.buf }

// estimateV2Size estimates the LLP2 file size.
func estimateV2Size(meta *PackageMeta) int {
	n := 88 // header
	n += 40 * 32 // section directory (max 32 sections)
	// string offsets: len+1 uint32s
	n += (len(meta.stringTable) + 1) * 4
	// string data
	for _, s := range meta.stringTable {
		n += len(s)
	}
	// each group section: groups (12 bytes each) + values
	n += len(meta.OrdinaryEdges) * 12
	n += countValuesSlice(meta.OrdinaryEdges) * 4
	n += len(meta.TypeChildren) * 12
	n += countValuesSlice(meta.TypeChildren) * 4
	n += len(meta.InterfaceInfo) * 12
	n += countInterfaceValues(meta.InterfaceInfo) * 8
	n += len(meta.UseIface) * 12
	n += countValuesSlice(meta.UseIface) * 4
	n += len(meta.UseIfaceMethod) * 12
	n += countDemandValues(meta.UseIfaceMethod) * 12
	n += len(meta.MethodInfo) * 12
	n += countSlotValues(meta.MethodInfo) * 16
	n += len(meta.UseNamedMethod) * 12
	n += countValuesSlice(meta.UseNamedMethod) * 4
	n += len(meta.ReflectMethod) * 4
	return n
}

func countValuesSlice(m map[Symbol][]Symbol) int { n := 0; for _, v := range m { n += len(v) }; return n }
func countInterfaceValues(m map[Symbol][]MethodSig) int { n := 0; for _, v := range m { n += len(v) }; return n }
func countDemandValues(m map[Symbol][]IfaceMethodDemand) int { n := 0; for _, v := range m { n += len(v) }; return n }
func countSlotValues(m map[Symbol][]MethodSlot) int { n := 0; for _, v := range m { n += len(v) }; return n }

func TestRoundTripV2(t *testing.T) {
	for name, s := range scales {
		t.Run(name, func(t *testing.T) {
			orig := genMeta(s)
			v2f, err := orig.ToV2Format()
			if err != nil {
				t.Fatalf("ToV2Format: %v", err)
			}
			var ws memWriteSeeker
			if err := v2f.WriteTo(&ws); err != nil {
				t.Fatalf("encode v2: %v", err)
			}
			view, err := ParseMetaV2(ws.Bytes())
			if err != nil {
				t.Fatalf("decode v2: %v", err)
			}
			if err := metaViewEqual(orig, view); err != nil {
				t.Fatalf("mismatch: %v", err)
			}
		})
	}
}

func metaViewEqual(orig *PackageMeta, view *PackageMetaView) error {
	// String table
	st := view.Strings()
	if st.Len() != len(orig.stringTable) {
		return fmt.Errorf("stringTable len: %d vs %d", st.Len(), len(orig.stringTable))
	}
	for i, s := range orig.stringTable {
		if st.Lookup(Symbol(i)) != s {
			return fmt.Errorf("stringTable[%d]: %q vs %q", i, s, st.Lookup(Symbol(i)))
		}
	}
	// OrdinaryEdges
	if err := checkGroupSymbol("OrdinaryEdges", orig.OrdinaryEdges,
		func(k Symbol) []Symbol { return view.OrdinaryEdges(k) }); err != nil {
		return err
	}
	// TypeChildren
	if err := checkGroupSymbol("TypeChildren", orig.TypeChildren,
		func(k Symbol) []Symbol { return view.TypeChildren(k) }); err != nil {
		return err
	}
	// InterfaceInfo
	for k, va := range orig.InterfaceInfo {
		vb := view.InterfaceMethods(k)
		if len(va) != len(vb) {
			return fmt.Errorf("InterfaceInfo[%d] len: %d vs %d", k, len(va), len(vb))
		}
		for i := range va {
			if Symbol(va[i].Name) != Symbol(vb[i].Name) || Symbol(va[i].MType) != Symbol(vb[i].MType) {
				return fmt.Errorf("InterfaceInfo[%d][%d]: {%d,%d} vs {%d,%d}",
					k, i, va[i].Name, va[i].MType, vb[i].Name, vb[i].MType)
			}
		}
	}
	// UseIface
	if err := checkGroupSymbol("UseIface", orig.UseIface,
		func(k Symbol) []Symbol { return view.UseIface(k) }); err != nil {
		return err
	}
	// UseIfaceMethod
	for k, va := range orig.UseIfaceMethod {
		vb := view.UseIfaceMethod(k)
		if len(va) != len(vb) {
			return fmt.Errorf("UseIfaceMethod[%d] len: %d vs %d", k, len(va), len(vb))
		}
		for i := range va {
			ea, eb := va[i], vb[i]
			if Symbol(ea.Target) != Symbol(eb.Target) || Symbol(ea.Sig.Name) != Symbol(eb.Name) || Symbol(ea.Sig.MType) != Symbol(eb.MType) {
				return fmt.Errorf("UseIfaceMethod[%d][%d]: {%d,%d,%d} vs {%d,%d,%d}",
					k, i, ea.Target, ea.Sig.Name, ea.Sig.MType, eb.Target, eb.Name, eb.MType)
			}
		}
	}
	// MethodInfo
	for k, va := range orig.MethodInfo {
		vb := view.MethodSlots(k)
		if len(va) != len(vb) {
			return fmt.Errorf("MethodInfo[%d] len: %d vs %d", k, len(va), len(vb))
		}
		for i := range va {
			ea, eb := va[i], vb[i]
			if Symbol(ea.Sig.Name) != Symbol(eb.Name) || Symbol(ea.Sig.MType) != Symbol(eb.MType) ||
				Symbol(ea.IFn) != Symbol(eb.IFn) || Symbol(ea.TFn) != Symbol(eb.TFn) {
				return fmt.Errorf("MethodInfo[%d][%d]: {%d,%d,%d,%d} vs {%d,%d,%d,%d}",
					k, i, ea.Sig.Name, ea.Sig.MType, ea.IFn, ea.TFn,
					eb.Name, eb.MType, eb.IFn, eb.TFn)
			}
		}
	}
	// UseNamedMethod
	if err := checkGroupSymbol("UseNamedMethod", orig.UseNamedMethod,
		func(k Symbol) []Symbol { return view.UseNamedMethod(k) }); err != nil {
		return err
	}
	// ReflectMethod
	for k := range orig.ReflectMethod {
		if !view.HasReflectMethod(k) {
			return fmt.Errorf("ReflectMethod: %d missing", k)
		}
	}
	return nil
}

func checkGroupSymbol(name string, m map[Symbol][]Symbol, lookup func(Symbol) []Symbol) error {
	if len(m) == 0 && lookup(0) == nil {
		return nil
	}
	for k, va := range m {
		vb := lookup(k)
		if len(va) != len(vb) {
			return fmt.Errorf("%s[%d] len: %d vs %d", name, k, len(va), len(vb))
		}
		for i := range va {
			if va[i] != vb[i] {
				return fmt.Errorf("%s[%d][%d]: %d vs %d", name, k, i, va[i], vb[i])
			}
		}
	}
	return nil
}

func BenchmarkEncodeV2(b *testing.B) {
	for name, s := range scales {
		b.Run(name, func(b *testing.B) {
			meta := genMeta(s)
			v2f, err := meta.ToV2Format()
			if err != nil {
				b.Fatalf("ToV2Format: %v", err)
			}
			b.ReportMetric(float64(estimateV2Size(meta)), "est-filesize")
			b.ResetTimer()
			for b.Loop() {
				var ws memWriteSeeker
				_ = v2f.WriteTo(&ws)
			}
		})
	}
}

func BenchmarkDecodeV2(b *testing.B) {
	for name, s := range scales {
		b.Run(name, func(b *testing.B) {
			meta := genMeta(s)
			v2f, err := meta.ToV2Format()
			if err != nil {
				b.Fatalf("ToV2Format: %v", err)
			}
			var ws memWriteSeeker
			if err := v2f.WriteTo(&ws); err != nil {
				b.Fatalf("encode v2: %v", err)
			}
			encoded := ws.Bytes()
			b.ReportMetric(float64(len(encoded)), "filesize-B")
			b.ResetTimer()
			for b.Loop() {
				_, _ = ParseMetaV2(encoded)
			}
		})
	}
}

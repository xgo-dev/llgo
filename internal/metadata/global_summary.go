package metadata

import (
	"fmt"
	"reflect"
	"sort"
)

// GlobalSummary is a whole-program metadata view in one global Symbol/Name space.
type GlobalSummary struct {
	stringTable []string

	symbolByText map[string]Symbol
	nameByText   map[string]Name

	ordinaryEdges  map[Symbol][]Symbol
	typeChildren   map[Symbol][]Symbol
	interfaceInfo  map[Symbol][]MethodSig
	useIface       map[Symbol][]Symbol
	useIfaceMethod map[Symbol][]IfaceMethodDemand
	methodInfo     map[Symbol][]MethodSlot
	useNamedMethod map[Symbol][]Name
	reflectMethod  map[Symbol]struct{}
}

// NewGlobalSummary merges package-local metadata into a whole-program view.
func NewGlobalSummary(pkgs []*PackageMeta) (*GlobalSummary, error) {
	b := newGlobalSummaryBuilder()
	for _, pm := range pkgs {
		if pm == nil {
			continue
		}
		r := newPackageRemapper(pm, b)

		pm.ForEachOrdinaryEdge(func(src Symbol, dsts []Symbol) {
			gsrc := r.symbol(src)
			for _, dst := range dsts {
				addUniqueSymbol(&b.summary.ordinaryEdges, gsrc, r.symbol(dst))
			}
		})
		pm.ForEachTypeChild(func(parent Symbol, children []Symbol) {
			gparent := r.symbol(parent)
			for _, child := range children {
				addUniqueSymbol(&b.summary.typeChildren, gparent, r.symbol(child))
			}
		})
		pm.ForEachInterface(func(iface Symbol, methods []MethodSig) {
			giface := r.symbol(iface)
			for _, method := range methods {
				addUniqueMethodSig(&b.summary.interfaceInfo, giface, r.methodSig(method))
			}
		})
		pm.ForEachUseIface(func(owner Symbol, types []Symbol) {
			gowner := r.symbol(owner)
			for _, typ := range types {
				addUniqueSymbol(&b.summary.useIface, gowner, r.symbol(typ))
			}
		})
		pm.ForEachUseIfaceMethod(func(owner Symbol, demands []IfaceMethodDemand) {
			gowner := r.symbol(owner)
			for _, demand := range demands {
				addUniqueIfaceMethodDemand(&b.summary.useIfaceMethod, gowner, IfaceMethodDemand{
					Target: r.symbol(demand.Target),
					Sig:    r.methodSig(demand.Sig),
				})
			}
		})
		var methodErr error
		pm.ForEachMethodInfo(func(typ Symbol, slots []MethodSlot) {
			if methodErr != nil {
				return
			}
			gtyp := r.symbol(typ)
			gslots := make([]MethodSlot, 0, len(slots))
			for _, slot := range slots {
				gslots = append(gslots, MethodSlot{
					Sig: r.methodSig(slot.Sig),
					IFn: r.symbol(slot.IFn),
					TFn: r.symbol(slot.TFn),
				})
			}
			methodErr = b.addMethodSlots(gtyp, gslots)
		})
		if methodErr != nil {
			return nil, methodErr
		}
		pm.ForEachUseNamedMethod(func(owner Symbol, names []Name) {
			gowner := r.symbol(owner)
			for _, name := range names {
				addUniqueName(&b.summary.useNamedMethod, gowner, r.name(name))
			}
		})
		pm.ForEachReflectMethod(func(owner Symbol) {
			b.summary.reflectMethod[r.symbol(owner)] = struct{}{}
		})
	}
	return b.build(), nil
}

// LookupSymbol returns a global Symbol for a module-level symbol name.
func (g *GlobalSummary) LookupSymbol(name string) (Symbol, bool) {
	if g == nil {
		return 0, false
	}
	sym, ok := g.symbolByText[name]
	return sym, ok
}

// SymbolName returns the text referenced by a global Symbol.
func (g *GlobalSummary) SymbolName(sym Symbol) string {
	if g == nil || int(sym) >= len(g.stringTable) {
		return ""
	}
	return g.stringTable[sym]
}

// Name returns the text referenced by a global Name.
func (g *GlobalSummary) Name(ref Name) string {
	if g == nil || int(ref) >= len(g.stringTable) {
		return ""
	}
	return g.stringTable[ref]
}

// Interfaces returns all interface type symbols known to the summary.
func (g *GlobalSummary) Interfaces() []Symbol {
	if g == nil {
		return nil
	}
	return sortedGlobalKeys(g.interfaceInfo, g.SymbolName)
}

// ConcreteTypes returns all concrete type symbols with method slots.
func (g *GlobalSummary) ConcreteTypes() []Symbol {
	if g == nil {
		return nil
	}
	return sortedGlobalKeys(g.methodInfo, g.SymbolName)
}

// OrdinaryEdges returns direct ordinary references from sym.
func (g *GlobalSummary) OrdinaryEdges(sym Symbol) []Symbol {
	if g == nil {
		return nil
	}
	return cloneSymbols(g.ordinaryEdges[sym])
}

// TypeChildren returns child type symbols for typ.
func (g *GlobalSummary) TypeChildren(typ Symbol) []Symbol {
	if g == nil {
		return nil
	}
	return cloneSymbols(g.typeChildren[typ])
}

// InterfaceMethods returns the method set for iface.
func (g *GlobalSummary) InterfaceMethods(iface Symbol) []MethodSig {
	if g == nil {
		return nil
	}
	return cloneMethodSigs(g.interfaceInfo[iface])
}

// UseIface returns concrete types that enter interface semantics from fn.
func (g *GlobalSummary) UseIface(fn Symbol) []Symbol {
	if g == nil {
		return nil
	}
	return cloneSymbols(g.useIface[fn])
}

// UseIfaceMethod returns interface method demands emitted by fn.
func (g *GlobalSummary) UseIfaceMethod(fn Symbol) []IfaceMethodDemand {
	if g == nil {
		return nil
	}
	return cloneIfaceMethodDemands(g.useIfaceMethod[fn])
}

// MethodSlots returns ABI method slots for typ.
func (g *GlobalSummary) MethodSlots(typ Symbol) []MethodSlot {
	if g == nil {
		return nil
	}
	return cloneMethodSlots(g.methodInfo[typ])
}

// UseNamedMethod returns constant MethodByName names emitted by fn.
func (g *GlobalSummary) UseNamedMethod(fn Symbol) []Name {
	if g == nil {
		return nil
	}
	return cloneNames(g.useNamedMethod[fn])
}

// HasReflectMethod reports whether fn triggers conservative reflection handling.
func (g *GlobalSummary) HasReflectMethod(fn Symbol) bool {
	if g == nil {
		return false
	}
	_, ok := g.reflectMethod[fn]
	return ok
}

type globalSummaryBuilder struct {
	summary  *GlobalSummary
	idByText map[string]uint32
}

func newGlobalSummaryBuilder() *globalSummaryBuilder {
	return &globalSummaryBuilder{
		summary: &GlobalSummary{
			symbolByText:   make(map[string]Symbol),
			nameByText:     make(map[string]Name),
			ordinaryEdges:  make(map[Symbol][]Symbol),
			typeChildren:   make(map[Symbol][]Symbol),
			interfaceInfo:  make(map[Symbol][]MethodSig),
			useIface:       make(map[Symbol][]Symbol),
			useIfaceMethod: make(map[Symbol][]IfaceMethodDemand),
			methodInfo:     make(map[Symbol][]MethodSlot),
			useNamedMethod: make(map[Symbol][]Name),
			reflectMethod:  make(map[Symbol]struct{}),
		},
		idByText: make(map[string]uint32),
	}
}

func (b *globalSummaryBuilder) internSymbol(text string) Symbol {
	id := b.internText(text)
	sym := Symbol(id)
	b.summary.symbolByText[text] = sym
	return sym
}

func (b *globalSummaryBuilder) internName(text string) Name {
	id := b.internText(text)
	name := Name(id)
	b.summary.nameByText[text] = name
	return name
}

func (b *globalSummaryBuilder) internText(text string) uint32 {
	if id, ok := b.idByText[text]; ok {
		return id
	}
	id := uint32(len(b.summary.stringTable))
	b.idByText[text] = id
	b.summary.stringTable = append(b.summary.stringTable, text)
	return id
}

func (b *globalSummaryBuilder) addMethodSlots(typ Symbol, slots []MethodSlot) error {
	if existing, ok := b.summary.methodInfo[typ]; ok {
		if reflect.DeepEqual(existing, slots) {
			return nil
		}
		return fmt.Errorf("conflicting MethodInfo for %s", b.summary.SymbolName(typ))
	}
	b.summary.methodInfo[typ] = cloneMethodSlots(slots)
	return nil
}

func (b *globalSummaryBuilder) build() *GlobalSummary {
	return b.summary
}

type packageRemapper struct {
	pm *PackageMeta
	b  *globalSummaryBuilder

	symbols map[Symbol]Symbol
	names   map[Name]Name
}

func newPackageRemapper(pm *PackageMeta, b *globalSummaryBuilder) *packageRemapper {
	return &packageRemapper{
		pm:      pm,
		b:       b,
		symbols: make(map[Symbol]Symbol),
		names:   make(map[Name]Name),
	}
}

func (r *packageRemapper) symbol(local Symbol) Symbol {
	if global, ok := r.symbols[local]; ok {
		return global
	}
	global := r.b.internSymbol(r.pm.SymbolName(local))
	r.symbols[local] = global
	return global
}

func (r *packageRemapper) name(local Name) Name {
	if global, ok := r.names[local]; ok {
		return global
	}
	global := r.b.internName(r.pm.Name(local))
	r.names[local] = global
	return global
}

func (r *packageRemapper) methodSig(local MethodSig) MethodSig {
	return MethodSig{
		Name:  r.name(local.Name),
		MType: r.symbol(local.MType),
	}
}

func addUniqueSymbol(m *map[Symbol][]Symbol, key, value Symbol) {
	values := (*m)[key]
	for _, existing := range values {
		if existing == value {
			return
		}
	}
	(*m)[key] = append(values, value)
}

func addUniqueName(m *map[Symbol][]Name, key Symbol, value Name) {
	values := (*m)[key]
	for _, existing := range values {
		if existing == value {
			return
		}
	}
	(*m)[key] = append(values, value)
}

func addUniqueMethodSig(m *map[Symbol][]MethodSig, key Symbol, value MethodSig) {
	values := (*m)[key]
	for _, existing := range values {
		if existing == value {
			return
		}
	}
	(*m)[key] = append(values, value)
}

func addUniqueIfaceMethodDemand(m *map[Symbol][]IfaceMethodDemand, key Symbol, value IfaceMethodDemand) {
	values := (*m)[key]
	for _, existing := range values {
		if existing == value {
			return
		}
	}
	(*m)[key] = append(values, value)
}

func sortedGlobalKeys[V any](m map[Symbol]V, name func(Symbol) string) []Symbol {
	keys := make([]Symbol, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return name(keys[i]) < name(keys[j]) })
	return keys
}

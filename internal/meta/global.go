package meta

// GlobalSummary is a whole-program metadata view over multiple PackageMetas,
// in one unified symbol/name space.
//
// Merge strategy:
//   - Symbols are interned into a global Symbol space; each package's local
//     symbols are mapped via locToGlb. Edges and TypeChildren are NOT rewritten
//     at merge time — they are translated lazily on query (the bulk of the data,
//     and the main cost we avoid up front).
//   - MethodInfo / InterfaceInfo / reflect are small, so they are translated
//     eagerly at merge time, interning method names into the global Name space.
//   - Duplicate symbols (e.g. linkonce type descriptors emitted by several
//     packages) are first-wins: the first package that owns facts for a symbol
//     becomes its owner; later duplicates are ignored.
type GlobalSummary struct {
	pkgs []*PackageMeta

	// symbol space
	symIntern  map[string]Symbol
	symStrings []string  // Symbol → text
	locToGlb   [][]Symbol // [pkgIdx][localSym] → global Symbol
	owner      []symLoc   // global Symbol → owning (pkg, local); pkg<0 if none

	// method-name space (distinct from symbols)
	nameIntern  map[string]Name
	nameStrings []string // Name → text

	// eagerly translated small sections
	methodInfo    map[Symbol][]GMethodSlot
	interfaceInfo map[Symbol][]GMethodSig
	reflect       map[Symbol]struct{}

	interfaces    []Symbol
	concreteTypes []Symbol
}

// Symbol is a whole-program symbol ID in GlobalSummary's unified namespace.
type Symbol uint32

// Name is a whole-program method-name ID, in a namespace distinct from Symbol.
type Name uint32

// GMethodSlot is a method slot in the global namespace.
type GMethodSlot struct {
	Name  Name
	MType Symbol
	IFn   Symbol
	TFn   Symbol
}

// GMethodSig is an interface method signature in the global namespace.
type GMethodSig struct {
	Name  Name
	MType Symbol
}

// IfaceMethodDemand is a reachable interface method call: a demand that some
// type's method matching Sig on interface Target be kept.
type IfaceMethodDemand struct {
	Target Symbol
	Sig    GMethodSig
}

// symLoc identifies a (package, local symbol) pair. pkg < 0 means "no owner".
type symLoc struct {
	pkg   int32
	local LocalSymbol
}

// NewGlobalSummary merges package-local metadata into a whole-program view.
func NewGlobalSummary(pkgs []*PackageMeta) (*GlobalSummary, error) {
	g := &GlobalSummary{
		pkgs:          pkgs,
		symIntern:     make(map[string]Symbol),
		nameIntern:    make(map[string]Name),
		locToGlb:      make([][]Symbol, len(pkgs)),
		methodInfo:    make(map[Symbol][]GMethodSlot),
		interfaceInfo: make(map[Symbol][]GMethodSig),
		reflect:       make(map[Symbol]struct{}),
	}

	// Phase 1: intern symbols, build locToGlb and owner (used for lazy
	// edges/children translation). Touches no edges.
	for pi, pm := range pkgs {
		if pm == nil {
			continue
		}
		n := pm.NSyms()
		tab := make([]Symbol, n)
		for li := LocalSymbol(0); li < LocalSymbol(n); li++ {
			gs := g.internSymbol(pm.SymbolName(li))
			tab[li] = gs
			if g.owner[gs].pkg < 0 && hasFacts(pm, li) {
				g.owner[gs] = symLoc{pkg: int32(pi), local: li}
			}
		}
		g.locToGlb[pi] = tab
	}

	// Phase 2: eagerly translate small sections (method/iface/reflect),
	// interning method names into the global Name space. First-wins on dups.
	for pi, pm := range pkgs {
		if pm == nil {
			continue
		}
		tab := g.locToGlb[pi]
		n := pm.NSyms()
		for li := LocalSymbol(0); li < LocalSymbol(n); li++ {
			gs := tab[li]
			if pm.IsConcreteType(li) {
				if _, done := g.methodInfo[gs]; !done {
					g.methodInfo[gs] = g.translateSlots(tab, pm, li)
					g.concreteTypes = append(g.concreteTypes, gs)
				}
			}
			if pm.IsInterface(li) {
				if _, done := g.interfaceInfo[gs]; !done {
					g.interfaceInfo[gs] = g.translateSigs(tab, pm, li)
					g.interfaces = append(g.interfaces, gs)
				}
			}
			if pm.HasReflect(li) {
				g.reflect[gs] = struct{}{}
			}
		}
	}
	return g, nil
}

// hasFacts reports whether li carries any facts in pm (i.e. is defined here,
// not merely referenced). Used to pick the owning package for lazy queries.
func hasFacts(pm *PackageMeta, li LocalSymbol) bool {
	return pm.HasEdges(li) ||
		pm.IsCompositeType(li) ||
		pm.IsConcreteType(li) ||
		pm.IsInterface(li) ||
		pm.HasReflect(li)
}

func (g *GlobalSummary) internSymbol(s string) Symbol {
	if id, ok := g.symIntern[s]; ok {
		return id
	}
	id := Symbol(len(g.symStrings))
	g.symIntern[s] = id
	g.symStrings = append(g.symStrings, s)
	g.owner = append(g.owner, symLoc{pkg: -1})
	return id
}

func (g *GlobalSummary) internName(s string) Name {
	if id, ok := g.nameIntern[s]; ok {
		return id
	}
	id := Name(len(g.nameStrings))
	g.nameIntern[s] = id
	g.nameStrings = append(g.nameStrings, s)
	return id
}

func (g *GlobalSummary) translateSlots(tab []Symbol, pm *PackageMeta, li LocalSymbol) []GMethodSlot {
	local := pm.MethodSlots(li)
	out := make([]GMethodSlot, len(local))
	for i, s := range local {
		out[i] = GMethodSlot{
			Name:  g.internName(pm.NameString(s.Name)),
			MType: tab[s.MType],
			IFn:   tab[s.IFn],
			TFn:   tab[s.TFn],
		}
	}
	return out
}

func (g *GlobalSummary) translateSigs(tab []Symbol, pm *PackageMeta, li LocalSymbol) []GMethodSig {
	local := pm.IfaceMethods(li)
	out := make([]GMethodSig, len(local))
	for i, s := range local {
		out[i] = GMethodSig{
			Name:  g.internName(pm.NameString(s.Name)),
			MType: tab[s.MType],
		}
	}
	return out
}

// ── symbol / name identity ────────────────────────────────────────────────────

// LookupSymbol returns the global Symbol for a module-level symbol name.
func (g *GlobalSummary) LookupSymbol(name string) (Symbol, bool) {
	id, ok := g.symIntern[name]
	return id, ok
}

// SymbolName returns the text of a global Symbol.
func (g *GlobalSummary) SymbolName(sym Symbol) string {
	if int(sym) < len(g.symStrings) {
		return g.symStrings[sym]
	}
	return ""
}

// Name returns the text of a global Name.
func (g *GlobalSummary) Name(n Name) string {
	if int(n) < len(g.nameStrings) {
		return g.nameStrings[n]
	}
	return ""
}

// ── enumeration ───────────────────────────────────────────────────────────────

// Interfaces returns all interface type symbols.
func (g *GlobalSummary) Interfaces() []Symbol { return g.interfaces }

// ConcreteTypes returns all concrete type symbols with method slots.
func (g *GlobalSummary) ConcreteTypes() []Symbol { return g.concreteTypes }

// ── eager small sections ──────────────────────────────────────────────────────

// MethodSlots returns the ABI method slots for concrete type typ. Read-only.
func (g *GlobalSummary) MethodSlots(typ Symbol) []GMethodSlot { return g.methodInfo[typ] }

// InterfaceMethods returns the method set for interface iface. Read-only.
func (g *GlobalSummary) InterfaceMethods(iface Symbol) []GMethodSig { return g.interfaceInfo[iface] }

// HasReflectMethod reports whether sym triggers conservative reflection handling.
func (g *GlobalSummary) HasReflectMethod(sym Symbol) bool {
	_, ok := g.reflect[sym]
	return ok
}

// ── lazy edge queries ─────────────────────────────────────────────────────────

// ownerEdges returns the owning package, its locToGlb table, and the raw local
// edges for sym, or nil if sym has no owner.
func (g *GlobalSummary) ownerEdges(sym Symbol) (*PackageMeta, []Symbol, []Edge) {
	if int(sym) >= len(g.owner) {
		return nil, nil, nil
	}
	loc := g.owner[sym]
	if loc.pkg < 0 {
		return nil, nil, nil
	}
	pm := g.pkgs[loc.pkg]
	return pm, g.locToGlb[loc.pkg], pm.Edges(loc.local)
}

// OrdinaryEdges returns plain reachability targets from sym (global Symbols).
func (g *GlobalSummary) OrdinaryEdges(sym Symbol) []Symbol {
	_, tab, edges := g.ownerEdges(sym)
	var out []Symbol
	for _, e := range edges {
		if e.Kind == EdgeOrdinary {
			out = append(out, tab[e.Target])
		}
	}
	return out
}

// UseIface returns concrete types converted to interfaces by sym.
func (g *GlobalSummary) UseIface(sym Symbol) []Symbol {
	_, tab, edges := g.ownerEdges(sym)
	var out []Symbol
	for _, e := range edges {
		if e.Kind == EdgeUseIface {
			out = append(out, tab[e.Target])
		}
	}
	return out
}

// UseIfaceMethod returns interface method demands emitted by sym.
func (g *GlobalSummary) UseIfaceMethod(sym Symbol) []IfaceMethodDemand {
	_, tab, edges := g.ownerEdges(sym)
	var out []IfaceMethodDemand
	for _, e := range edges {
		if e.Kind != EdgeUseIfaceMethod {
			continue
		}
		iface := tab[e.Target]
		sigs := g.interfaceInfo[iface]
		if int(e.Extra) < len(sigs) {
			out = append(out, IfaceMethodDemand{Target: iface, Sig: sigs[e.Extra]})
		}
	}
	return out
}

// UseNamedMethod returns constant MethodByName names referenced by sym.
func (g *GlobalSummary) UseNamedMethod(sym Symbol) []Name {
	pm, _, edges := g.ownerEdges(sym)
	if pm == nil {
		return nil
	}
	var out []Name
	for _, e := range edges {
		if e.Kind == EdgeUseNamedMethod {
			name := pm.NameString(NameRef{Off: e.Target, Len: e.Extra})
			out = append(out, g.internName(name))
		}
	}
	return out
}

// TypeChildren returns child type symbols for typ (global Symbols).
func (g *GlobalSummary) TypeChildren(typ Symbol) []Symbol {
	if int(typ) >= len(g.owner) {
		return nil
	}
	loc := g.owner[typ]
	if loc.pkg < 0 {
		return nil
	}
	pm := g.pkgs[loc.pkg]
	tab := g.locToGlb[loc.pkg]
	local := pm.TypeChildren(loc.local)
	if len(local) == 0 {
		return nil
	}
	out := make([]Symbol, len(local))
	for i, c := range local {
		out[i] = tab[c]
	}
	return out
}

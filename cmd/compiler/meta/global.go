package meta

// GlobalSummary is a whole-program metadata view over multiple PackageMetas,
// in one unified symbol/name space.
//
// Merge strategy:
//   - Symbols are interned into a global Symbol space; each package's local
//     symbols are mapped via locToGlb. Edges, FuncDemands, TypeChildren,
//     MethodSlots and IfaceMethods are NOT rewritten at merge time — they
//     are translated lazily on query. Only the strings are interned up front.
//   - Duplicate symbols (e.g. linkonce type descriptors emitted by several
//     packages) are assigned one owner by fact strength. Function-demand,
//     MethodInfo, InterfaceInfo and TypeChildren facts outrank ordinary edges,
//     so descriptor references do not hide the package that carries semantic
//     method/interface facts.
//
// GlobalSummary borrows its input PackageMetas. They must remain open and
// unchanged for the lifetime of the summary.
type GlobalSummary struct {
	pkgs []*PackageMeta

	// symbol space
	symIntern  map[string]Symbol
	symStrings []string   // Symbol → text
	locToGlb   [][]Symbol // [pkgIdx][localSym] → global Symbol
	owner      []symLoc   // global Symbol → owning (pkg, local); pkg<0 if none

	// method-name space (distinct from symbols)
	nameIntern  map[string]Name
	nameStrings []string // Name → text

	interfaces []Symbol
}

// symLoc identifies a (package, local symbol) pair. pkg < 0 means "no owner".
type symLoc struct {
	pkg   int32
	local Symbol
}

type ownerKind uint8

const (
	ownerNone ownerKind = iota
	ownerOrdinary
	ownerType
	ownerInterface
	ownerFunction
)

// ownerState exists only while NewGlobalSummary selects owners.
type ownerState struct {
	kind ownerKind
}

// NewGlobalSummary merges package-local metadata into a whole-program view.
//
// Phase 1 interns all symbol names and builds the locToGlb mapping, owner
// indices, and type-kind flags. No per-symbol data is translated — only
// string interning and CSR range checks happen here.
//
// MethodSlots / IfaceMethods / FuncDemands are translated lazily on each
// query. This avoids translating metadata that DCE never reaches.
func NewGlobalSummary(pkgs []*PackageMeta) (*GlobalSummary, error) {
	g := &GlobalSummary{
		pkgs:       pkgs,
		symIntern:  make(map[string]Symbol),
		nameIntern: make(map[string]Name),
		locToGlb:   make([][]Symbol, len(pkgs)),
	}
	var ownerStates []ownerState
	var interfaceSeen []bool

	// Phase 1: intern symbols, build locToGlb and owner, mark type kinds.
	// Touches no edges, translates no slot/sig data.
	for pi, pm := range pkgs {
		n := pm.nsyms
		tab := make([]Symbol, n)
		for li := Symbol(0); li < Symbol(n); li++ {
			gs, added := g.internSymbol(pm.symbolName(li))
			if added {
				ownerStates = append(ownerStates, ownerState{})
				interfaceSeen = append(interfaceSeen, false)
			}
			tab[li] = gs
			kind := g.considerOwner(gs, symLoc{pkg: int32(pi), local: li}, pm, li, &ownerStates[gs])

			// Collect each interface with method information once.
			if kind == ownerInterface && !interfaceSeen[gs] {
				interfaceSeen[gs] = true
				g.interfaces = append(g.interfaces, gs)
			}
		}
		g.locToGlb[pi] = tab
	}
	return g, nil
}

func packageOwnerKind(pm *PackageMeta, local Symbol) ownerKind {
	switch {
	case pm.hasFuncDemand(local):
		return ownerFunction
	case pm.nifaceMethod(local) > 0:
		return ownerInterface
	case pm.nmethodSlot(local) > 0 || pm.ntypeChild(local) > 0:
		return ownerType
	case pm.hasOrdinaryEdges(local):
		return ownerOrdinary
	default:
		return ownerNone
	}
}

// considerOwner merges one package-local owner candidate into the global view
// and returns the selected owner's semantic kind.
func (g *GlobalSummary) considerOwner(global Symbol, candidate symLoc, pm *PackageMeta, local Symbol, state *ownerState) ownerKind {
	if state.kind == ownerFunction {
		return state.kind
	}

	candidateKind := packageOwnerKind(pm, local)
	switch candidateKind {
	case ownerFunction:
		g.owner[global] = candidate
		state.kind = ownerFunction
	case ownerType, ownerInterface:
		if state.kind == ownerNone || state.kind == ownerOrdinary {
			g.owner[global] = candidate
			state.kind = candidateKind
		}
	case ownerOrdinary:
		if state.kind == ownerNone {
			g.owner[global] = candidate
			state.kind = ownerOrdinary
		}
	}
	return state.kind
}

func (g *GlobalSummary) internSymbol(s string) (Symbol, bool) {
	if id, ok := g.symIntern[s]; ok {
		return id, false
	}
	id := Symbol(len(g.symStrings))
	g.symIntern[s] = id
	g.symStrings = append(g.symStrings, s)
	g.owner = append(g.owner, symLoc{pkg: -1})
	return id, true
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

// ownerData returns the owning package and locToGlb table for sym.
func (g *GlobalSummary) ownerData(sym Symbol) (*PackageMeta, []Symbol, Symbol) {
	loc := g.owner[sym]
	return g.pkgs[loc.pkg], g.locToGlb[loc.pkg], loc.local
}

func (g *GlobalSummary) translateSlots(tab []Symbol, pm *PackageMeta, li Symbol) []MethodSlot {
	local := pm.methodSlots(li)
	out := make([]MethodSlot, len(local))
	for i, s := range local {
		out[i] = MethodSlot{
			Name:  g.internName(pm.nameString(s.Name)),
			MType: tab[s.MType],
			IFn:   tab[s.IFn],
			TFn:   tab[s.TFn],
		}
	}
	return out
}

func (g *GlobalSummary) translateSigs(tab []Symbol, pm *PackageMeta, li Symbol) []MethodSig {
	local := pm.ifaceMethods(li)
	out := make([]MethodSig, len(local))
	for i, s := range local {
		out[i] = MethodSig{
			Name:  g.internName(pm.nameString(s.Name)),
			MType: tab[s.MType],
		}
	}
	return out
}

func (g *GlobalSummary) translateFuncDemands(tab []Symbol, pm *PackageMeta, li Symbol) []FuncDemand {
	local := pm.funcDemands(li)
	var out []FuncDemand
	for _, d := range local {
		switch d.Kind {
		case DemandUseIface:
			out = append(out, FuncDemand{Kind: d.Kind, Target: tab[d.Target]})
		case DemandIfaceMethod:
			iface := tab[d.Target]
			sigs := g.IfaceMethods(iface)
			if int(d.Extra) < len(sigs) {
				out = append(out, FuncDemand{Kind: d.Kind, Target: iface, Sig: sigs[d.Extra]})
			}
		case DemandNamedMethod:
			name := pm.nameString(nameRef{Off: d.Target, Len: d.Extra})
			out = append(out, FuncDemand{Kind: d.Kind, MethodName: g.internName(name)})
		case DemandReflectMethod:
			out = append(out, FuncDemand{Kind: d.Kind})
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

// HasFacts reports whether sym has a package owner carrying metadata facts.
// A symbol may be present only because another fact references it, such as a
// method slot's function symbol.
func (g *GlobalSummary) HasFacts(sym Symbol) bool {
	return g.owner[sym].pkg >= 0
}

// SymbolName returns the text of a global Symbol.
func (g *GlobalSummary) SymbolName(sym Symbol) string {
	return g.symStrings[sym]
}

// Name returns the text of a global Name.
func (g *GlobalSummary) Name(n Name) string {
	return g.nameStrings[n]
}

// ── enumeration ───────────────────────────────────────────────────────────────

// Ifaces returns all interface type symbols with method information.
// The returned slice is owned by GlobalSummary and must not be modified.
func (g *GlobalSummary) Ifaces() []Symbol { return g.interfaces }

// ── lazy per-type queries ─────────────────────────────────────────────────────

// MethodSlots returns the ABI method slots for concrete type typ.
func (g *GlobalSummary) MethodSlots(typ Symbol) []MethodSlot {
	pm, tab, li := g.ownerData(typ)
	return g.translateSlots(tab, pm, li)
}

// IfaceMethods returns the method set for interface iface.
func (g *GlobalSummary) IfaceMethods(iface Symbol) []MethodSig {
	pm, tab, li := g.ownerData(iface)
	return g.translateSigs(tab, pm, li)
}

// ── lazy edge queries ─────────────────────────────────────────────────────────

// OrdinaryEdges returns plain reachability targets from sym (global Symbols).
func (g *GlobalSummary) OrdinaryEdges(sym Symbol) []Symbol {
	pm, tab, li := g.ownerData(sym)
	edges := pm.ordinaryEdges(li)
	var out []Symbol
	for _, dst := range edges {
		out = append(out, tab[dst])
	}
	return out
}

// FuncDemands returns the method/interface/reflection demands emitted by sym.
// Records are translated to the global symbol and name spaces on demand.
func (g *GlobalSummary) FuncDemands(sym Symbol) []FuncDemand {
	pm, tab, li := g.ownerData(sym)
	return g.translateFuncDemands(tab, pm, li)
}

// TypeChildren returns child type symbols for typ (global Symbols).
func (g *GlobalSummary) TypeChildren(typ Symbol) []Symbol {
	pm, tab, li := g.ownerData(typ)
	local := pm.typeChildren(li)
	if len(local) == 0 {
		return nil
	}
	out := make([]Symbol, len(local))
	for i, c := range local {
		out[i] = tab[c]
	}
	return out
}

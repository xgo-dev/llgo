// Package metadata defines package summary facts and their LLPS binary format.
package metadata

// Symbol is a module-level named entity participating in reachability.
type Symbol uint32

// Name is a plain name reference used for semantic matching.
type Name uint32

// MethodSig describes a method by short name and function type symbol.
type MethodSig struct {
	Name  Name
	MType Symbol
}

// MethodSlot describes one entry in a concrete type's ABI method table.
type MethodSlot struct {
	Sig MethodSig
	IFn Symbol
	TFn Symbol
}

// IfaceMethodDemand records one reachable interface method call.
type IfaceMethodDemand struct {
	Target Symbol
	Sig    MethodSig
}

// PackageMeta holds all single-package facts needed by whole-program analysis.
type PackageMeta struct {
	stringTable []string

	ordinaryEdges  map[Symbol][]Symbol
	typeChildren   map[Symbol][]Symbol
	interfaceInfo  map[Symbol][]MethodSig
	useIface       map[Symbol][]Symbol
	useIfaceMethod map[Symbol][]IfaceMethodDemand
	methodInfo     map[Symbol][]MethodSlot
	useNamedMethod map[Symbol][]Name
	reflectMethod  map[Symbol]struct{}
}

// NewPackageMeta creates an empty PackageMeta with initialized maps.
func NewPackageMeta(stringTable []string) *PackageMeta {
	table := append([]string(nil), stringTable...)
	return &PackageMeta{
		stringTable:    table,
		ordinaryEdges:  make(map[Symbol][]Symbol),
		typeChildren:   make(map[Symbol][]Symbol),
		interfaceInfo:  make(map[Symbol][]MethodSig),
		useIface:       make(map[Symbol][]Symbol),
		useIfaceMethod: make(map[Symbol][]IfaceMethodDemand),
		methodInfo:     make(map[Symbol][]MethodSlot),
		useNamedMethod: make(map[Symbol][]Name),
		reflectMethod:  make(map[Symbol]struct{}),
	}
}

// SymbolName returns the string referenced by a Symbol.
func (pm *PackageMeta) SymbolName(sym Symbol) string {
	if int(sym) >= len(pm.stringTable) {
		return ""
	}
	return pm.stringTable[sym]
}

// Name returns the string referenced by a Name.
func (pm *PackageMeta) Name(ref Name) string {
	if int(ref) >= len(pm.stringTable) {
		return ""
	}
	return pm.stringTable[ref]
}

// ForEachOrdinaryEdge visits each ordinary reachability edge group.
func (pm *PackageMeta) ForEachOrdinaryEdge(fn func(src Symbol, dsts []Symbol)) {
	for src, dsts := range pm.ordinaryEdges {
		fn(src, dsts)
	}
}

// ForEachTypeChild visits each type-child edge group.
func (pm *PackageMeta) ForEachTypeChild(fn func(parent Symbol, children []Symbol)) {
	for parent, children := range pm.typeChildren {
		fn(parent, children)
	}
}

// ForEachInterface visits each interface method set.
func (pm *PackageMeta) ForEachInterface(fn func(iface Symbol, methods []MethodSig)) {
	for iface, methods := range pm.interfaceInfo {
		fn(iface, methods)
	}
}

// ForEachUseIface visits each function's concrete types used as interfaces.
func (pm *PackageMeta) ForEachUseIface(fn func(owner Symbol, types []Symbol)) {
	for owner, types := range pm.useIface {
		fn(owner, types)
	}
}

// ForEachUseIfaceMethod visits each function's interface method demands.
func (pm *PackageMeta) ForEachUseIfaceMethod(fn func(owner Symbol, demands []IfaceMethodDemand)) {
	for owner, demands := range pm.useIfaceMethod {
		fn(owner, demands)
	}
}

// ForEachMethodInfo visits each concrete type's method slots.
func (pm *PackageMeta) ForEachMethodInfo(fn func(typ Symbol, slots []MethodSlot)) {
	for typ, slots := range pm.methodInfo {
		fn(typ, slots)
	}
}

// ForEachUseNamedMethod visits each function's constant MethodByName names.
func (pm *PackageMeta) ForEachUseNamedMethod(fn func(owner Symbol, names []Name)) {
	for owner, names := range pm.useNamedMethod {
		fn(owner, names)
	}
}

// ForEachReflectMethod visits each function that needs conservative reflection handling.
func (pm *PackageMeta) ForEachReflectMethod(fn func(owner Symbol)) {
	for owner := range pm.reflectMethod {
		fn(owner)
	}
}

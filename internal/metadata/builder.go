package metadata

// Builder accumulates per-package metadata facts and produces a PackageMeta.
// It is created by the build layer and passed into cl/ssa for filling.
type Builder struct {
	pm      *PackageMeta
	strToID map[string]Symbol
}

// NewBuilder creates a Builder ready to accept facts.
func NewBuilder() *Builder {
	return &Builder{
		pm:      NewPackageMeta(nil),
		strToID: make(map[string]Symbol),
	}
}

// String registers a string and returns its Symbol ID.
// Repeated calls with the same string return the same ID.
func (b *Builder) String(s string) Symbol {
	if id, ok := b.strToID[s]; ok {
		return id
	}
	id := Symbol(len(b.strToID))
	b.strToID[s] = id
	return id
}

// AddEdge records an ordinary edge src → dst.
func (b *Builder) AddEdge(src, dst Symbol) {
	b.pm.OrdinaryEdges[src] = append(b.pm.OrdinaryEdges[src], dst)
}

// AddTypeChild records that parent type references child type.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	b.pm.TypeChildren[parent] = append(b.pm.TypeChildren[parent], child)
}

// AddIfaceEntry records the method set for an interface type.
func (b *Builder) AddIfaceEntry(iface Symbol, methods []MethodSig) {
	b.pm.InterfaceInfo[iface] = append(b.pm.InterfaceInfo[iface], methods...)
}

// AddUseIface records types that enter the interface domain when owner is reachable.
func (b *Builder) AddUseIface(owner Symbol, types []Symbol) {
	b.pm.UseIface[owner] = append(b.pm.UseIface[owner], types...)
}

// AddUseIfaceMethod records interface method demands when owner is reachable.
func (b *Builder) AddUseIfaceMethod(owner Symbol, demands []IfaceMethodDemand) {
	b.pm.UseIfaceMethod[owner] = append(b.pm.UseIfaceMethod[owner], demands...)
}

// AddMethodInfo records the method table slots for a concrete type.
func (b *Builder) AddMethodInfo(typeID Symbol, slots []MethodSlot) {
	b.pm.MethodInfo[typeID] = append(b.pm.MethodInfo[typeID], slots...)
}

// AddUseNamedMethod records method names demanded via MethodByName when owner is reachable.
func (b *Builder) AddUseNamedMethod(owner Symbol, names []Symbol) {
	b.pm.UseNamedMethod[owner] = append(b.pm.UseNamedMethod[owner], names...)
}

// AddReflectMethod records that owner forces conservative reflection handling.
func (b *Builder) AddReflectMethod(owner Symbol) {
	b.pm.ReflectMethod[owner] = struct{}{}
}

// Build finalizes the builder and returns an immutable PackageMeta.
// The Builder must not be used after Build is called.
func (b *Builder) Build() *PackageMeta {
	// Build string table from strToID (sorted by Symbol ID)
	table := make([]string, len(b.strToID))
	for s, id := range b.strToID {
		table[id] = s
	}
	b.pm.stringTable = table
	return b.pm
}

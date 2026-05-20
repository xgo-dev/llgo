package metadata

// Builder accumulates per-package metadata facts and builds a PackageMeta.
type Builder struct {
	pm      *PackageMeta
	strToID map[string]uint32
}

// NewBuilder creates an empty metadata builder.
func NewBuilder() *Builder {
	return &Builder{
		pm:      NewPackageMeta(nil),
		strToID: make(map[string]uint32),
	}
}

// Symbol registers a module-level named symbol.
func (b *Builder) Symbol(s string) Symbol {
	return Symbol(b.intern(s))
}

// Name registers a plain semantic name.
func (b *Builder) Name(s string) Name {
	return Name(b.intern(s))
}

func (b *Builder) intern(s string) uint32 {
	if id, ok := b.strToID[s]; ok {
		return id
	}
	id := uint32(len(b.strToID))
	b.strToID[s] = id
	return id
}

// AddEdge records an ordinary reachability edge src -> dst.
func (b *Builder) AddEdge(src, dst Symbol) {
	b.pm.ordinaryEdges[src] = append(b.pm.ordinaryEdges[src], dst)
}

// AddTypeChild records that parent type references child type.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	b.pm.typeChildren[parent] = append(b.pm.typeChildren[parent], child)
}

// AddIfaceEntry records the method set of an interface type.
func (b *Builder) AddIfaceEntry(iface Symbol, methods []MethodSig) {
	b.pm.interfaceInfo[iface] = append(b.pm.interfaceInfo[iface], methods...)
}

// AddUseIface records types converted to interface when owner is reachable.
func (b *Builder) AddUseIface(owner Symbol, types []Symbol) {
	b.pm.useIface[owner] = append(b.pm.useIface[owner], types...)
}

// AddUseIfaceMethod records interface method calls when owner is reachable.
func (b *Builder) AddUseIfaceMethod(owner Symbol, demands []IfaceMethodDemand) {
	b.pm.useIfaceMethod[owner] = append(b.pm.useIfaceMethod[owner], demands...)
}

// AddMethodInfo records concrete type method table slots.
func (b *Builder) AddMethodInfo(typeID Symbol, slots []MethodSlot) {
	b.pm.methodInfo[typeID] = append(b.pm.methodInfo[typeID], slots...)
}

// AddUseNamedMethod records constant MethodByName method names.
func (b *Builder) AddUseNamedMethod(owner Symbol, names []Name) {
	b.pm.useNamedMethod[owner] = append(b.pm.useNamedMethod[owner], names...)
}

// AddReflectMethod records that owner triggers conservative reflection handling.
func (b *Builder) AddReflectMethod(owner Symbol) {
	b.pm.reflectMethod[owner] = struct{}{}
}

// Build finalizes the builder and returns the package metadata.
func (b *Builder) Build() *PackageMeta {
	table := make([]string, len(b.strToID))
	for s, id := range b.strToID {
		table[id] = s
	}
	b.pm.stringTable = table
	return b.pm
}

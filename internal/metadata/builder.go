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
	b.pm.ordinaryEdges[src] = appendSymbolUnique(b.pm.ordinaryEdges[src], dst)
}

// AddTypeChild records that parent type references child type.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	b.pm.typeChildren[parent] = appendSymbolUnique(b.pm.typeChildren[parent], child)
}

// AddIfaceEntry records the method set of an interface type.
func (b *Builder) AddIfaceEntry(iface Symbol, methods []MethodSig) {
	for _, method := range methods {
		b.pm.interfaceInfo[iface] = appendMethodSigUnique(b.pm.interfaceInfo[iface], method)
	}
}

// AddUseIface records types converted to interface when owner is reachable.
func (b *Builder) AddUseIface(owner Symbol, types []Symbol) {
	for _, typ := range types {
		b.pm.useIface[owner] = appendSymbolUnique(b.pm.useIface[owner], typ)
	}
}

// AddUseIfaceMethod records interface method calls when owner is reachable.
func (b *Builder) AddUseIfaceMethod(owner Symbol, demands []IfaceMethodDemand) {
	for _, demand := range demands {
		b.pm.useIfaceMethod[owner] = appendIfaceMethodDemandUnique(b.pm.useIfaceMethod[owner], demand)
	}
}

// AddMethodInfo records concrete type method table slots.
func (b *Builder) AddMethodInfo(typeID Symbol, slots []MethodSlot) {
	for _, slot := range slots {
		b.pm.methodInfo[typeID] = appendMethodSlotUnique(b.pm.methodInfo[typeID], slot)
	}
}

// AddUseNamedMethod records constant MethodByName method names.
func (b *Builder) AddUseNamedMethod(owner Symbol, names []Name) {
	for _, name := range names {
		b.pm.useNamedMethod[owner] = appendNameUnique(b.pm.useNamedMethod[owner], name)
	}
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

func appendSymbolUnique(items []Symbol, item Symbol) []Symbol {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func appendNameUnique(items []Name, item Name) []Name {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func appendMethodSigUnique(items []MethodSig, item MethodSig) []MethodSig {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func appendIfaceMethodDemandUnique(items []IfaceMethodDemand, item IfaceMethodDemand) []IfaceMethodDemand {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func appendMethodSlotUnique(items []MethodSlot, item MethodSlot) []MethodSlot {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

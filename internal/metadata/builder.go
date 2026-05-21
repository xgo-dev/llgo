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
	addUniqueSymbol(&b.pm.ordinaryEdges, src, dst)
}

// AddTypeChild records that parent type references child type.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	addUniqueSymbol(&b.pm.typeChildren, parent, child)
}

// AddIfaceEntry records the method set of an interface type.
func (b *Builder) AddIfaceEntry(iface Symbol, methods []MethodSig) {
	for _, method := range methods {
		addUniqueMethodSig(&b.pm.interfaceInfo, iface, method)
	}
}

// AddUseIface records types converted to interface when owner is reachable.
func (b *Builder) AddUseIface(owner Symbol, types []Symbol) {
	for _, typ := range types {
		addUniqueSymbol(&b.pm.useIface, owner, typ)
	}
}

// AddUseIfaceMethod records interface method calls when owner is reachable.
func (b *Builder) AddUseIfaceMethod(owner Symbol, demands []IfaceMethodDemand) {
	for _, demand := range demands {
		addUniqueIfaceMethodDemand(&b.pm.useIfaceMethod, owner, demand)
	}
}

// AddMethodInfo records concrete type method table slots.
func (b *Builder) AddMethodInfo(typeID Symbol, slots []MethodSlot) {
	if len(slots) == 0 {
		return
	}
	b.pm.methodInfo[typeID] = append(b.pm.methodInfo[typeID], slots...)
}

// AddUseNamedMethod records constant MethodByName method names.
func (b *Builder) AddUseNamedMethod(owner Symbol, names []Name) {
	for _, name := range names {
		addUniqueName(&b.pm.useNamedMethod, owner, name)
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

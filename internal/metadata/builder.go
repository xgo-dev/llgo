package metadata

// Builder accumulates per-package metadata facts and builds a PackageMeta.
type Builder struct {
	pm                 *PackageMeta
	strToID            map[string]uint32
	seenEdge           map[[2]Symbol]struct{}
	seenTypeChild      map[[2]Symbol]struct{}
	seenInterfaceInfo  map[interfaceInfoKey]struct{}
	seenUseIface       map[[2]Symbol]struct{}
	seenUseIfaceMethod map[useIfaceMethodKey]struct{}
	seenUseNamedMethod map[useNamedMethodKey]struct{}
}

// NewBuilder creates an empty metadata builder.
func NewBuilder() *Builder {
	return &Builder{
		pm:                 NewPackageMeta(nil),
		strToID:            make(map[string]uint32),
		seenEdge:           make(map[[2]Symbol]struct{}),
		seenTypeChild:      make(map[[2]Symbol]struct{}),
		seenInterfaceInfo:  make(map[interfaceInfoKey]struct{}),
		seenUseIface:       make(map[[2]Symbol]struct{}),
		seenUseIfaceMethod: make(map[useIfaceMethodKey]struct{}),
		seenUseNamedMethod: make(map[useNamedMethodKey]struct{}),
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
	key := [2]Symbol{src, dst}
	if _, ok := b.seenEdge[key]; ok {
		return
	}
	b.seenEdge[key] = struct{}{}
	b.pm.ordinaryEdges[src] = append(b.pm.ordinaryEdges[src], dst)
}

// AddTypeChild records that parent type references child type.
func (b *Builder) AddTypeChild(parent, child Symbol) {
	key := [2]Symbol{parent, child}
	if _, ok := b.seenTypeChild[key]; ok {
		return
	}
	b.seenTypeChild[key] = struct{}{}
	b.pm.typeChildren[parent] = append(b.pm.typeChildren[parent], child)
}

// AddIfaceEntry records the method set of an interface type.
func (b *Builder) AddIfaceEntry(iface Symbol, methods []MethodSig) {
	for _, method := range methods {
		key := interfaceInfoKey{Iface: iface, Sig: method}
		if _, ok := b.seenInterfaceInfo[key]; ok {
			continue
		}
		b.seenInterfaceInfo[key] = struct{}{}
		b.pm.interfaceInfo[iface] = append(b.pm.interfaceInfo[iface], method)
	}
}

// AddUseIface records types converted to interface when owner is reachable.
func (b *Builder) AddUseIface(owner Symbol, types []Symbol) {
	for _, typ := range types {
		key := [2]Symbol{owner, typ}
		if _, ok := b.seenUseIface[key]; ok {
			continue
		}
		b.seenUseIface[key] = struct{}{}
		b.pm.useIface[owner] = append(b.pm.useIface[owner], typ)
	}
}

// AddUseIfaceMethod records interface method calls when owner is reachable.
func (b *Builder) AddUseIfaceMethod(owner Symbol, demands []IfaceMethodDemand) {
	for _, demand := range demands {
		key := useIfaceMethodKey{Owner: owner, Demand: demand}
		if _, ok := b.seenUseIfaceMethod[key]; ok {
			continue
		}
		b.seenUseIfaceMethod[key] = struct{}{}
		b.pm.useIfaceMethod[owner] = append(b.pm.useIfaceMethod[owner], demand)
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
		key := useNamedMethodKey{Owner: owner, Name: name}
		if _, ok := b.seenUseNamedMethod[key]; ok {
			continue
		}
		b.seenUseNamedMethod[key] = struct{}{}
		b.pm.useNamedMethod[owner] = append(b.pm.useNamedMethod[owner], name)
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

type interfaceInfoKey struct {
	Iface Symbol
	Sig   MethodSig
}

type useIfaceMethodKey struct {
	Owner  Symbol
	Demand IfaceMethodDemand
}

type useNamedMethodKey struct {
	Owner Symbol
	Name  Name
}

// Package metadata defines the package summary data model and its LLPS binary
// serialization format for whole-program method reachability analysis.
package metadata

// Symbol is an integer ID referencing a string in the unified string table.
type Symbol = int

// MethodSig describes a method signature: name + function type symbol.
type MethodSig struct {
	Name  Symbol // method name string table ID
	MType Symbol // method type symbol string table ID
}

// MethodSlot describes one slot in a concrete type's method table.
// Slots are stored in order matching Go runtime abi.Method table.
type MethodSlot struct {
	Sig MethodSig
	IFn Symbol // interface method entry symbol
	TFn Symbol // concrete type method symbol
}

// IfaceMethodDemand records an interface method call: (interface, method sig).
type IfaceMethodDemand struct {
	Target Symbol // target interface symbol
	Sig    MethodSig
}

// PackageMeta holds all per-package analysis facts for whole-program method
// reachability analysis. All strings are stored once in stringTable and
// referenced by integer Symbol IDs throughout the other fields.
type PackageMeta struct {
	stringTable []string

	OrdinaryEdges  map[Symbol][]Symbol            // top-level symbol -> direct references
	TypeChildren   map[Symbol][]Symbol            // parent type -> child types
	InterfaceInfo  map[Symbol][]MethodSig         // interface type -> method signatures
	UseIface       map[Symbol][]Symbol            // function -> types converted to interface
	UseIfaceMethod map[Symbol][]IfaceMethodDemand // function -> interface method calls
	MethodInfo     map[Symbol][]MethodSlot        // concrete type -> method slots
	UseNamedMethod map[Symbol][]Symbol            // function -> method names from MethodByName
	ReflectMethod  map[Symbol]struct{}            // functions with conservative reflection
}

// StringTable returns the string table (read-only).
func (pm *PackageMeta) StringTable() []string { return pm.stringTable }

// NewPackageMeta creates an empty PackageMeta with initialized maps.
func NewPackageMeta(stringTable []string) *PackageMeta {
	return &PackageMeta{
		stringTable:    stringTable,
		OrdinaryEdges:  make(map[Symbol][]Symbol),
		TypeChildren:   make(map[Symbol][]Symbol),
		InterfaceInfo:  make(map[Symbol][]MethodSig),
		UseIface:       make(map[Symbol][]Symbol),
		UseIfaceMethod: make(map[Symbol][]IfaceMethodDemand),
		MethodInfo:     make(map[Symbol][]MethodSlot),
		UseNamedMethod: make(map[Symbol][]Symbol),
		ReflectMethod:  make(map[Symbol]struct{}),
	}
}

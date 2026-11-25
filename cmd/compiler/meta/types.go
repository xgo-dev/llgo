// Package meta defines the binary format and in-memory view for LLGo package
// summary cache files (.meta). The format is designed for zero-copy access via
// mmap: the file layout is the memory layout.
package meta

// Symbol is a symbol ID whose namespace is determined by the Builder,
// PackageMeta, or GlobalSummary that owns it. Builder and PackageMeta symbols
// are package-local; GlobalSummary symbols belong to its unified namespace.
type Symbol uint32

// Name is a whole-program method-name ID, in a namespace distinct from Symbol.
type Name uint32

// DemandKind identifies a function-level method/interface/reflection demand.
type DemandKind uint32

// FuncDemand kinds used in the FuncDemand section and global analysis API.
const (
	DemandUseIface      DemandKind = 1
	DemandIfaceMethod   DemandKind = 2
	DemandNamedMethod   DemandKind = 3
	DemandReflectMethod DemandKind = 4
)

// MethodSlot is a method slot in the global namespace.
type MethodSlot struct {
	Name  Name
	MType Symbol
	IFn   Symbol
	TFn   Symbol
}

// MethodSig is an interface method signature in the global namespace.
type MethodSig struct {
	Name  Name
	MType Symbol
}

// FuncDemand is a function-level method/interface/reflection demand in the
// global namespace. Valid fields depend on Kind:
//
//   - DemandUseIface: Target is the concrete type converted to an interface.
//   - DemandIfaceMethod: Target is the interface and Sig is the demanded method.
//   - DemandNamedMethod: MethodName is the constant MethodByName argument.
//   - DemandReflectMethod: no additional fields are set.
type FuncDemand struct {
	Kind       DemandKind
	Target     Symbol
	Sig        MethodSig
	MethodName Name
}

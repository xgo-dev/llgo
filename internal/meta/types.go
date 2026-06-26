// Package meta defines the binary format and in-memory view for LLGo package
// summary cache files (.meta). The format is designed for zero-copy access via
// mmap: the file layout is the memory layout.
package meta

// LocalSymbol is a package-local symbol ID, equal to its index in the Symbols
// section. Valid within one PackageMeta only; use GlobalSummary for cross-package
// references.
type LocalSymbol uint32

// NameRef references a name string by its byte range in the string table.
// It is used for method short names, which are matched by value across packages
// — names are not module-level symbols and live in their own namespace.
type NameRef struct {
	Off uint32
	Len uint32
}

// Edge kinds used in the Edges section.
const (
	// EdgeOrdinary is a plain symbol reference (call, type use, global var, etc.).
	EdgeOrdinary uint8 = 0
	// EdgeUseIface marks that the source converts Target type to an interface.
	EdgeUseIface uint8 = 1
	// EdgeUseIfaceMethod marks a call to method Extra of interface Target.
	EdgeUseIfaceMethod uint8 = 2
	// EdgeUseNamedMethod marks a constant MethodByName call; Target is a
	// stringTable byte offset (not a LocalSymbol).
	EdgeUseNamedMethod uint8 = 3
)

// Magic is the 4-byte file signature.
const Magic = "LLPM"

// Version is the current binary format version.
const Version = 1

// Section index constants for Header.SectionOffsets.
const (
	SecStringTable  = 0
	SecSymbols      = 1
	SecEdges        = 2
	SecTypeChildren = 3
	SecMethodInfo   = 4
	SecIfaceInfo    = 5
	SecReflect      = 6
	numSections     = 7
)

// headerSize = magic(4) + version(4) + sectionOffsets(numSections×4)
const headerSize = 4 + 4 + numSections*4

package abi

// CompactGCRootMap marks a root map whose metadata word encodes the number
// of direct roots, followed by addresses of pointer fields of one width.
const (
	CompactGCRootMap uint32 = 1 << 31
	WideGCRootMap    uint32 = 1 << 30
	GCRootDirectMask        = WideGCRootMap - 1
)

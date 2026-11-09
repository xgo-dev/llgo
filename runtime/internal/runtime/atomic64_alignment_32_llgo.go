//go:build llgo && go1.23 && !baremetal && (386 || arm || mips || mipsle)

package runtime

import "structs"

// Counter storage needs eight-byte alignment, independent of the target's
// natural uint64 alignment. The value and atomic operations remain our own.
type atomic64Alignment = structs.Align8

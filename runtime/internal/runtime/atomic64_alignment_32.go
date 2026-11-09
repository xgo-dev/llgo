//go:build !baremetal && (386 || arm || mips || mipsle) && (!llgo || !go1.23)

package runtime

import "sync/atomic"

// Compatibility for other Go compilers and toolchains predating structs.
// LLGo uses its explicit alignment marker when the structs package is available.
type atomic64Alignment = [0]atomic.Uint64

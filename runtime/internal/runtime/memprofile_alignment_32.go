//go:build !baremetal && (386 || arm || mips || mipsle)

package runtime

import "sync/atomic"

// A zero-length atomic array preserves alignment without adding storage.
type memProfileAlignment = [0]atomic.Uint64

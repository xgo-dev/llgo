//go:build (darwin || linux) && !go1.23

package runtime

import (
	llrt "github.com/goplus/llgo/runtime/internal/runtime"
	_ "unsafe"
)

//go:linkname pprof_memProfileInternal runtime.pprof_memProfileInternal
func pprof_memProfileInternal(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	suppressed := llrt.MemProfileSuppress()
	defer llrt.MemProfileRestoreSuppressed(suppressed)

	return MemProfile(p, inuseZero)
}

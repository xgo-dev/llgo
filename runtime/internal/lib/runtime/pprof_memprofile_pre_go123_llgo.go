//go:build (darwin || linux) && !go1.23

package runtime

import _ "unsafe"

//go:linkname pprof_memProfileInternal runtime.pprof_memProfileInternal
func pprof_memProfileInternal(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	return MemProfile(p, inuseZero)
}

//go:build (darwin || linux || (llgo && wasm)) && go1.25

package runtime

import _ "unsafe"

//go:linkname sync_runtime_SemacquireWaitGroup sync.runtime_SemacquireWaitGroup
func sync_runtime_SemacquireWaitGroup(addr *uint32, _ bool) {
	syncWaitGroupAcquire(addr)
}

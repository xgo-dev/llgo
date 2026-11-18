//go:build !llgo || baremetal || (wasm && !llgo.wasm.debugger)

package runtime

var TracebackCreator func([]uintptr) int

func registerTraceback(gp, parent *g) {}
func attachTraceback(gp *g)           {}
func unregisterTraceback(gp *g)       {}
func releaseFaultSnapshot()           {}

func TracebackWaiting(bool) {}

// Non-native builds do not register native creator-PC snapshots.
func TracebackParentPCs() (uint64, []uintptr) { return 0, nil }

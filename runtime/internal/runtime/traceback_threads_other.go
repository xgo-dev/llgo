//go:build !llgo || baremetal || wasm

package runtime

var TracebackCreator func() uintptr

func registerTraceback(gp, parent *g) {}
func attachTraceback(gp *g)           {}
func unregisterTraceback(gp *g)       {}
func releaseFaultSnapshot()           {}

func TracebackWaiting(bool) {}

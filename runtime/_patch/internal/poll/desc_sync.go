//go:build llgo && (wasip1 || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded))

package poll

//llgo:skip (*pollDesc).prepare (*pollDesc).pollable

// WASI and bare-metal descriptors use synchronous I/O because their
// runtime poll hooks reject registration.
func (*pollDesc) prepare(int, bool) error { return nil }

func (*pollDesc) pollable() bool { return false }

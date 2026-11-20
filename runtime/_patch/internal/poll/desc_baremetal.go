//go:build llgo && linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded

package poll

//llgo:skip (*pollDesc).prepare (*pollDesc).pollable

// The bare-metal runtime rejects poll registration. Every descriptor uses
// synchronous I/O, including descriptors opened through os.NewFile.
func (*pollDesc) prepare(int, bool) error { return nil }

func (*pollDesc) pollable() bool { return false }

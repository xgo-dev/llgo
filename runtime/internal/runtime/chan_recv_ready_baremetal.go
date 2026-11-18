//go:build baremetal

package runtime

// Keep the locked receive path on baremetal. Even word-sized atomic loads
// and stores can need unavailable libatomic helpers on small RISC-V targets.
func (*Chan) recvMayBeReady() bool { return true }

func (*Chan) publishRecvReady()  {}
func (*Chan) publishRecvClosed() {}

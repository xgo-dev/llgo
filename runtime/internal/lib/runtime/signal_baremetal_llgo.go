//go:build baremetal

package runtime

// Baremetal targets do not provide OS signal delivery.

func signal_enable(sig uint32) {
	_ = sig
}

func signal_disable(sig uint32) {
	_ = sig
}

func signal_ignore(sig uint32) {
	_ = sig
}

func signal_ignored(sig uint32) bool {
	_ = sig
	return false
}

func signal_recv() uint32 {
	return 0
}

func signalWaitUntilIdle() {}

//go:build !wasm

package runtime

func capturePanicCallerFrames(any) {}

func panicCallerSnapshotAvailable() bool {
	return getg().panicPCs.n != 0
}

func clearPanicCallerSnapshot() {
	p := &getg().panicPCs
	if p.fault != 0 {
		releaseFaultSnapshot()
	}
	p.n = 0
	p.longPCs = nil
	p.fault = 0
}

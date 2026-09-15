//go:build !wasm

package runtime

func capturePanicCallerFrames(any) {}

func panicCallerSnapshotAvailable() bool {
	return getg().panicPCs.n != 0
}

func clearPanicCallerSnapshot() {}

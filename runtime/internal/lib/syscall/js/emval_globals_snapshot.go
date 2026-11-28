//go:build llgo && js && wasm && llgo.wasm.fs_snapshot && !llgo.wasm.workers

package js

//go:wasmimport env llgo_fs_snapshot_global
func fsSnapshotGlobal(slot uint32) float64

func snapshotGlobal(slot uint32) Value {
	return valueFromEmval(uintptr(fsSnapshotGlobal(slot)))
}

func initEmvalGlobals() {
	if emvalGlobalsReady {
		return
	}
	// Use the same properties, including the constructors seen by the
	// filesystem initializer, even if the host replaced a global since then.
	valueGlobal = snapshotGlobal(0)
	objectConstructor = snapshotGlobal(1)
	stringConstructor = snapshotGlobal(2)
	arrayConstructor = snapshotGlobal(3)
	functionConstructor = snapshotGlobal(4)
	valueNaN = snapshotGlobal(5)
	valueZero = snapshotGlobal(6)
	uint8Array = snapshotGlobal(7)
	uint8ClampedArray = snapshotGlobal(8)
	emvalGlobalsReady = true
}

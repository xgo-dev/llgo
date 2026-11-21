//go:build llgo && js && wasm && llgo.wasm.fs_snapshot && !llgo.wasm.workers

package syscall

import (
	"sync"
	"syscall/js"
	_ "unsafe"
)

//go:wasmimport env llgo_fs_snapshot_capture
func llgoFSSnapshotCapture(slot uint32)

//go:wasmimport env llgo_fs_snapshot_constants
func llgoFSSnapshotConstants(slot uint32) uint32

//go:wasmimport env llgo_fs_snapshot_flag
func llgoFSSnapshotFlag(flag uint32) float64

//go:wasmimport env llgo_fs_snapshot_error
func llgoFSSnapshotError() uint32

//go:wasmimport env llgo_fs_snapshot_ref
func llgoFSSnapshotRef(slot uint32) float64

//go:linkname llgoFSValueFromEmval syscall/js.valueFromEmval
func llgoFSValueFromEmval(handle uintptr) js.Value

type llgoFSValue uint32
type llgoFSConstants struct{}

var llgoFSSlots [4]struct {
	once  sync.Once
	value js.Value
}

func llgoCaptureFSValue(slot uint32) llgoFSValue {
	llgoFSSnapshotCapture(slot)
	return llgoFSValue(slot)
}

func llgoCaptureFSConstants(fs llgoFSValue) llgoFSConstants {
	if code := llgoFSSnapshotConstants(uint32(fs)); code != 0 {
		llgoFSSnapshotPanic(code)
	}
	return llgoFSConstants{}
}

func (llgoFSConstants) intFlag(flag uint32) int {
	value := llgoFSSnapshotFlag(flag)
	if value != value {
		if code := llgoFSSnapshotError(); code != 0 {
			llgoFSSnapshotPanic(code)
		}
	}
	return int(value)
}

func llgoFSSnapshotPanic(code uint32) {
	if code == 32 {
		panic("bad type")
	}
	method := "Value.Int"
	if code >= 16 {
		method = "Value.Get"
	}
	panic(&js.ValueError{Method: method, Type: js.Type(code & 7)})
}

func (v llgoFSValue) value() js.Value {
	slot := &llgoFSSlots[v]
	slot.once.Do(func() {
		slot.value = llgoFSValueFromEmval(uintptr(llgoFSSnapshotRef(uint32(v))))
	})
	return slot.value
}

func (v llgoFSValue) Get(name string) js.Value { return v.value().Get(name) }
func (v llgoFSValue) Call(name string, args ...any) js.Value {
	return v.value().Call(name, args...)
}
func (v llgoFSValue) New(args ...any) js.Value { return v.value().New(args...) }

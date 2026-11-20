//go:build llgo && wasm

package wasmexportabi

//llgo:cdecl
type reflectSlot struct {
	Data *byte
	Tag  uint32
}

type reflectTextRecord struct {
	Slot reflectSlot
	Len  int64
}

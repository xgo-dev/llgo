//go:build llgo && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package llgoext

import (
	"reflect"
	"testing"
	_ "unsafe"
)

//llgo:cdecl
type reflectedWasmNativeUintptr func(uintptr) uintptr

func reflectedWasmIncrement(n uintptr) uintptr { return n + 1 }

//go:linkname reflectedWasmNativeIncrement C.llgo_reflect_wasm_increment
func reflectedWasmNativeIncrement(n uintptr) uintptr

func TestReflectWasmNativeUintptrABI(t *testing.T) {
	high := uint64(1) << 40
	input := uintptr(41)
	native := reflectedWasmNativeUintptr(reflectedWasmNativeIncrement)
	got := reflect.ValueOf(native).Call([]reflect.Value{reflect.ValueOf(input)})
	if len(got) != 1 || uintptr(got[0].Uint()) != native(input) {
		t.Fatalf("reflected native result = %v, want %#x", got, native(input))
	}
	input = uintptr(high) | 41
	outcome := func(call func() uintptr) (result uintptr, panicked bool) {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		result = call()
		return
	}
	direct, directPanic := outcome(func() uintptr { return native(input) })
	reflected, reflectedPanic := outcome(func() uintptr {
		return uintptr(reflect.ValueOf(native).Call([]reflect.Value{reflect.ValueOf(input)})[0].Uint())
	})
	if directPanic != reflectedPanic || (!directPanic && direct != reflected) {
		t.Fatalf("native direct = (%#x, panic=%v), reflection = (%#x, panic=%v)", direct, directPanic, reflected, reflectedPanic)
	}
	goResult := reflect.ValueOf(reflectedWasmIncrement).Call([]reflect.Value{reflect.ValueOf(input)})
	if uintptr(goResult[0].Uint()) != input+1 {
		t.Fatalf("reflected Go result = %#x, want %#x", goResult[0].Uint(), input+1)
	}
}

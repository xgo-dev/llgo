//go:build baremetal && llgo.baremetal.stdio && !wasm

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// This profile has no libffi provider. Keep metadata available without
// importing the descriptors and their initialization code.
const useWasmReflectBridges = false

func resetWasmFuncBridge(*abi.FuncType) {}

func providerNativeSource(*abi.FuncType, *closure) *Value { return nil }

func callProvider(*abi.FuncType, unsafe.Pointer, unsafe.Pointer, *abi.Type, *unsafe.Pointer, bool, []Value) []Value {
	panic("reflect: dynamic calls require a function call provider")
}

func makeProviderFunc(*funcType, func([]Value) []Value, unsafe.Pointer, *Value) Value {
	panic("reflect: dynamic calls require a function call provider")
}

package build

import (
	"fmt"
	"strings"

	ssa "github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llvm"
)

// This section holds storage that cannot contain heap pointers, including
// writable pointer-free Go globals classified by SSA. Its C identifier gives
// wasm-ld synthesized __start_/__stop_ boundaries.
const wasmGCNoScanSection = ssa.WasmGCNoScanSection

func prepareWasmGCGlobals(mod llvm.Module, roots bool, namespace string, typeTests bool) {
	if !roots || !strings.HasPrefix(mod.Target(), "wasm") {
		return
	}
	index := 0
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		index++
		if global.IsGlobalConstant() && !global.IsThreadLocal() && global.Section() == "" && !global.Initializer().IsNil() && (!typeTests || relocationFreeGCData(global.Initializer())) {
			global.SetSection(wasmGCNoScanSection)
		}
		if global.Section() == wasmGCNoScanSection && global.Comdat().C == nil {
			// A shared segment boundary must not merge unrelated data into one
			// input chunk: wasm-ld still needs to discard each unused global.
			group := mod.Comdat(fmt.Sprintf("__llgo_noscan$%s$%d", namespace, index))
			group.SetSelectionKind(llvm.AnyComdatSelectionKind)
			global.SetComdat(group)
		}
	}
}

// When type tests are enabled, keep symbol-bearing constants in their ordinary
// sections. In particular,
// static itabs acquire type-test metadata during LTO, and LowerTypeTests owns
// their layout and rejects explicit sections. Numeric tables need no symbols.
func relocationFreeGCData(value llvm.Value) bool {
	if value.IsNil() || !value.IsAGlobalValue().IsNil() {
		return false
	}
	for i := 0; i < value.OperandsCount(); i++ {
		if !relocationFreeGCData(value.Operand(i)) {
			return false
		}
	}
	return true
}

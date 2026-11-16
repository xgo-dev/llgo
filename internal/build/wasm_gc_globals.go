package build

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// LLVM constants cannot acquire pointers to dynamically allocated objects.
// Keep them outside the conservative global-root scan: packed numeric tables
// (including pclntab hashes) otherwise look like Memory32 heap addresses.
// A C identifier gives wasm-ld synthesized __start_/__stop_ boundaries.
const wasmGCNoScanSection = ssa.WasmGCNoScanSection

func prepareWasmGCGlobals(mod llvm.Module, roots bool, namespace string) {
	if !roots || !strings.HasPrefix(mod.Target(), "wasm") {
		return
	}
	index := 0
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		index++
		if global.IsGlobalConstant() && !global.IsThreadLocal() && global.Section() == "" && !global.Initializer().IsNil() {
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

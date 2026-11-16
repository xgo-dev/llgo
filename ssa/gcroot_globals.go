package ssa

import (
	"go/types"
)

// WasmGCNoScanSection holds static storage that cannot contain Go heap roots.
// wasm-ld supplies its __start_/__stop_ boundaries to the linear collector.
const WasmGCNoScanSection = "llgo_gc_noscan"

func (g Global) excludePointerFreeGlobal() {
	prog := g.prog
	if !prog.GCRootsEnabled() || prog.Target().GOARCH != "wasm" || g.impl.IsThreadLocal() || g.impl.Section() != "" {
		return
	}
	// Keep zero-initialized storage in .bss rather than embedding large zero
	// arrays in the module. It retains the existing conservative scan.
	if g.impl.Initializer().IsNull() {
		return
	}
	typ := prog.Elem(g.Type).raw.Type
	if typ == nil {
		return
	}
	if _, tuple := typ.(*types.Tuple); tuple {
		return
	}
	// Use the Go type before storage lowering: a Memory32 Go pointer may
	// occupy an i64 slot, so the final LLVM storage type is insufficient.
	if !prog.abi.HasPointers(typ) {
		g.impl.SetSection(WasmGCNoScanSection)
	}
}

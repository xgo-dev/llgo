package build

import "github.com/xgo-dev/llvm"

// localizeWasmStackAddresses runs after LLVM's main optimization pipeline.
// SjLj lowering can turn entry-block slot addresses into catch-pad PHIs, then
// spill every address before every throwing call. Compute constant alloca
// offsets at the stores instead: the frame and all root publications remain
// unchanged, without keeping a separate address live across setjmp.
func localizeWasmStackAddresses(goarch string, mod llvm.Module) int {
	if goarch != "wasm" {
		return 0
	}
	setjmp := mod.NamedFunction("setjmp")
	if setjmp.IsNil() {
		return 0
	}
	functions := make(map[llvm.Value]bool)
	for use := setjmp.FirstUse(); !use.IsNil(); use = use.NextUse() {
		call := use.User().IsACallInst()
		if !call.IsNil() && call.CalledValue() == setjmp {
			functions[call.InstructionParent().Parent()] = true
		}
	}
	b := mod.Context().NewBuilder()
	defer b.Dispose()
	moved := 0
	for fn := range functions {
		entry := fn.FirstBasicBlock()
		for address := entry.FirstInstruction(); !address.IsNil(); {
			next := llvm.NextInstruction(address)
			if !address.IsAGetElementPtrInst().IsNil() && !address.Operand(0).IsAAllocaInst().IsNil() {
				indices := make([]llvm.Value, address.OperandsCount()-1)
				constant := true
				for i := range indices {
					indices[i] = address.Operand(i + 1)
					constant = constant && !indices[i].IsAConstantInt().IsNil()
				}
				if constant {
					var stores []llvm.Value
					for use := address.FirstUse(); !use.IsNil(); use = use.NextUse() {
						store := use.User().IsAStoreInst()
						if !store.IsNil() && store.Operand(1) == address {
							stores = append(stores, store)
						}
					}
					for _, store := range stores {
						b.SetInsertPointBefore(store)
						local := b.CreateGEP(address.GEPSourceElementType(), address.Operand(0), indices, "")
						store.SetOperand(1, local)
						moved++
					}
					if address.FirstUse().IsNil() {
						address.EraseFromParentAsInstruction()
					}
				}
			}
			address = next
		}
	}
	return moved
}

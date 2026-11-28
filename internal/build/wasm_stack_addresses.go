package build

import (
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// localizeWasmStackAddresses runs after LLVM's main optimization pipeline.
// SjLj lowering can turn entry-block slot addresses into catch-pad PHIs, then
// spill every address before every throwing call. Compute constant alloca
// offsets at the stores instead: the frame and all root publications remain
// unchanged, without keeping a separate address live across setjmp.
func localizeWasmStackAddresses(goarch string, mod llvm.Module) int {
	if goarch != "wasm" {
		return 0
	}
	functions := wasmSetjmpFunctions(mod.NamedFunction("setjmp"))
	if len(functions) == 0 {
		return 0
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

func wasmSetjmpFunctions(setjmp llvm.Value) map[llvm.Value]bool {
	if setjmp.IsNil() {
		return nil
	}
	functions := make(map[llvm.Value]bool)
	for use := setjmp.FirstUse(); !use.IsNil(); use = use.NextUse() {
		call := use.User().IsACallInst()
		if !call.IsNil() && call.CalledValue() == setjmp {
			functions[call.InstructionParent().Parent()] = true
		}
	}
	return functions
}

// Emscripten's JavaScript SjLj lowering adds a setjmp-table check to calls
// that can longjmp. Inlining into a frame with setjmp can turn a single
// check into many. Keep the call boundaries for the full-LTO size profile.
func limitEmscriptenSetjmpInlining(ctx *context, mod llvm.Module) {
	if ctx.crossCompile.WasmProvider != crosscompile.WasmProviderEmscripten ||
		ctx.buildConf.OptLevel != optlevel.Oz || ctx.buildConf.ltoMode() != lto.Full ||
		ctx.buildConf.DebugArtifactMode != DebugArtifactNone {
		return
	}
	setjmp := mod.NamedFunction("setjmp")
	functions := wasmSetjmpFunctions(setjmp)
	if len(functions) == 0 || !usesEmscriptenJSSjLj(ctx) {
		return
	}
	const functionAttrs = -1 // LLVMAttributeFunctionIndex
	noInline := mod.Context().CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0)
	alwaysInline := llvm.AttributeKindID("alwaysinline")
	for fn := range functions {
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.IsACallInst().IsNil() {
					continue
				}
				callee := inst.CalledValue()
				if callee == setjmp || strings.HasPrefix(callee.Name(), "llvm.") ||
					!inst.GetCallSiteEnumAttribute(functionAttrs, alwaysInline).IsNil() ||
					(!callee.IsAFunction().IsNil() && !callee.GetEnumFunctionAttribute(alwaysInline).IsNil()) {
					continue
				}
				inst.AddCallSiteAttribute(functionAttrs, noInline)
			}
		}
	}
}

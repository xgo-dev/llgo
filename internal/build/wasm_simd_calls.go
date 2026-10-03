package build

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

// lowerEmscriptenSIMDCalls keeps vectors inside Wasm when Emscripten's JS SjLj
// lowering wraps a potentially throwing call. JavaScript cannot carry v128.
// Only calls in setjmp functions need a memory bridge; ordinary vector calls
// retain their vector ABI. Run after optimization so inlined calls are covered.
// Remaining vector-call bodies cannot be inlined later into a setjmp function.
func lowerEmscriptenSIMDCalls(provider string, mod llvm.Module) int {
	if provider != "emscripten" {
		return 0
	}
	setjmp := mod.NamedFunction("setjmp")
	functions := make(map[llvm.Value]bool)
	if !setjmp.IsNil() {
		for use := setjmp.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User().IsACallInst()
			if !call.IsNil() && call.CalledValue() == setjmp {
				functions[call.InstructionParent().Parent()] = true
			}
		}
	}
	var calls []llvm.Value
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.IsACallInst().IsNil() || strings.HasPrefix(inst.CalledValue().Name(), "llvm.") {
					continue
				}
				typ := inst.CalledFunctionType()
				vector := typ.ReturnType().TypeKind() == llvm.VectorTypeKind
				for i := 0; i < inst.OperandsCount()-1; i++ {
					vector = vector || inst.Operand(i).Type().TypeKind() == llvm.VectorTypeKind
				}
				if vector {
					if functions[fn] {
						calls = append(calls, inst)
					} else {
						// A later backend/LTO inliner must not transplant these
						// unbridged calls into a function containing setjmp.
						// The body has already received the selected optimization.
						fn.RemoveEnumFunctionAttribute(llvm.AttributeKindID("alwaysinline"))
						fn.AddFunctionAttr(mod.Context().CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
					}
				}
			}
		}
	}
	for i, call := range calls {
		bridgeEmscriptenSIMDCall(mod, call, i)
	}
	return len(calls)
}

func bridgeEmscriptenSIMDCall(mod llvm.Module, call llvm.Value, id int) {
	ctx := mod.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	oldType := call.CalledFunctionType()
	retType := oldType.ReturnType()
	vectorResult := retType.TypeKind() == llvm.VectorTypeKind
	pointer := llvm.PointerType(ctx.Int8Type(), 0)
	types := []llvm.Type{pointer}
	args := []llvm.Value{call.CalledValue()}
	// New stack slots contain only vector bits, never Go pointers.
	allocate := func(typ llvm.Type) llvm.Value {
		b.SetInsertPointBefore(call.InstructionParent().Parent().FirstBasicBlock().FirstInstruction())
		return b.CreateAlloca(typ, "simd.sjlj.slot")
	}
	var resultSlot llvm.Value
	if vectorResult {
		resultSlot = allocate(retType)
		types = append(types, pointer)
		args = append(args, resultSlot)
		retType = ctx.VoidType()
	}
	paramOffset := len(types)
	// Include any already-promoted variadic operands as fixed bridge arguments.
	// LLGo emits no operand bundles; the last call operand is the callee.
	oldParams := make([]llvm.Type, call.OperandsCount()-1)
	for i := range oldParams {
		oldParams[i] = call.Operand(i).Type()
	}
	for i, typ := range oldParams {
		arg := call.Operand(i)
		if typ.TypeKind() == llvm.VectorTypeKind {
			slot := allocate(typ)
			b.SetInsertPointBefore(call)
			b.CreateStore(arg, slot)
			arg, typ = slot, pointer
		}
		types, args = append(types, typ), append(args, arg)
	}
	// Each bridge retains the original call and its complete ABI attributes.
	// noinline/optnone keep late optimization from exposing a v128 JS call again.
	bridgeType := llvm.FunctionType(retType, types, false)
	bridge := llvm.AddFunction(mod, fmt.Sprintf("__llgo_simd_sjlj.%d", id), bridgeType)
	bridge.SetLinkage(llvm.InternalLinkage)
	for _, name := range []string{"noinline", "optnone"} {
		bridge.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID(name), 0))
	}
	bridge.AddTargetDependentFunctionAttr("target-features", "+simd128")
	b.SetInsertPointBefore(call)
	replacement := b.CreateCall(bridgeType, bridge, args, "")
	replacement.InstructionSetDebugLoc(call.InstructionDebugLoc())
	value := replacement
	if vectorResult {
		value = b.CreateLoad(oldType.ReturnType(), resultSlot, "")
	}
	call.ReplaceAllUsesWith(value)
	block := ctx.AddBasicBlock(bridge, "entry")
	b.SetInsertPointAtEnd(block)
	for i, typ := range oldParams {
		arg := bridge.Param(paramOffset + i)
		if typ.TypeKind() == llvm.VectorTypeKind {
			arg = b.CreateLoad(typ, arg, "")
		}
		call.SetOperand(i, arg)
	}
	call.SetOperand(call.OperandsCount()-1, bridge.Param(0))
	call.RemoveFromParentAsInstruction()
	b.Insert(call)
	call.SetTailCall(false)
	if vectorResult {
		b.CreateStore(call, bridge.Param(1))
	}
	if retType.TypeKind() == llvm.VoidTypeKind {
		b.CreateRetVoid()
	} else {
		b.CreateRet(call)
	}
}

package build

import (
	"fmt"
	"os"
	"strings"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/xtool/safesplit"
	"github.com/xgo-dev/llvm"
)

func usesEmscriptenJSSjLj(ctx *context) bool {
	provider := ctx.crossCompile.WasmProvider
	if provider != crosscompile.WasmProviderEmscripten && provider != crosscompile.WasmProviderGoJS {
		return false
	}
	if ctx.crossCompile.WasmProfile == crosscompile.WasmProfileJ64 {
		// Memory64 IR uses matching clang with explicit JS SjLj flags, rather
		// than emcc. EMCC_CFLAGS does not select that compiler's lowering.
		return true
	}
	emccFlags := ctx.commands.lookup("EMCC_CFLAGS")
	if ctx.commands.environ == nil {
		emccFlags = os.Getenv("EMCC_CFLAGS")
	}
	flags := safesplit.SplitPkgConfigFlags(emccFlags)
	compiler := ctx.irCompiler()
	linker := ctx.linker()
	// emcc appends EMCC_CFLAGS after its command-line arguments. Require
	// native SjLj at both codegen and link time, including LTO codegen.
	return !emscriptenNativeSjLj(append(compiler.CompileArguments(), flags...)) ||
		!emscriptenNativeSjLj(append(linker.LinkArguments(), flags...))
}

func emscriptenNativeSjLj(args []string) bool {
	wasmEH := false
	wasmEHSetting := ""
	longjmp := "1"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "@") {
			// Unknown response-file contents can override any earlier option.
			return false
		}
		if arg == "-fwasm-exceptions" {
			wasmEH = true
		}
		if arg == "-s" && i+1 < len(args) {
			i++
			arg += args[i]
		}
		if value, ok := strings.CutPrefix(arg, "-sSUPPORT_LONGJMP="); ok {
			longjmp = strings.Trim(value, "\"'")
		}
		if value, ok := strings.CutPrefix(arg, "-sWASM_EXCEPTIONS="); ok {
			wasmEHSetting = value
		}
	}
	// Emscripten applies explicit -s settings after parsing driver flags.
	if wasmEHSetting != "" {
		wasmEH = wasmEHSetting == "1"
	}
	return longjmp == "wasm" || longjmp == "1" && wasmEH
}

// lowerEmscriptenSIMDCalls keeps vectors inside Wasm when Emscripten's JS SjLj
// lowering wraps a potentially throwing call. JavaScript cannot carry v128.
// Only calls in setjmp functions need a memory bridge; ordinary vector calls
// retain their vector ABI. Run after optimization so inlined calls are covered.
// Remaining vector-call bodies cannot be inlined later into a setjmp function.
func lowerEmscriptenSIMDCalls(ctx *context, mod llvm.Module) int {
	if !usesEmscriptenJSSjLj(ctx) {
		return 0
	}
	setjmp := mod.NamedFunction("setjmp")
	var calls []llvm.Value
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		var vectorCalls []llvm.Value
		hasSetjmp := false
		for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.IsACallInst().IsNil() || strings.HasPrefix(inst.CalledValue().Name(), "llvm.") {
					continue
				}
				if inst.CalledValue() == setjmp {
					hasSetjmp = true
				}
				typ := inst.CalledFunctionType()
				vector := typ.ReturnType().TypeKind() == llvm.VectorTypeKind
				for i := 0; i < inst.OperandsCount()-1; i++ {
					vector = vector || inst.Operand(i).Type().TypeKind() == llvm.VectorTypeKind
				}
				if vector {
					vectorCalls = append(vectorCalls, inst)
				}
			}
		}
		if hasSetjmp {
			calls = append(calls, vectorCalls...)
		} else if len(vectorCalls) != 0 {
			// A later backend/LTO inliner must not transplant these
			// unbridged calls into a function containing setjmp.
			// The body has already received the selected optimization.
			fn.RemoveEnumFunctionAttribute(llvm.AttributeKindID("alwaysinline"))
			fn.AddFunctionAttr(mod.Context().CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
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
	// noinline/optnone keep late inlining from exposing a v128 JS call again.
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
			// LLVM 22's O3 argument promotion ignores optnone and can turn
			// this pointer back into a v128 parameter. A volatile load keeps
			// the memory ABI required by the JavaScript SjLj wrapper.
			arg.SetVolatile(true)
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

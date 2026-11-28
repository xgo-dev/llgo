package llvmfmv

import (
	intllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// The x86-64 pass selects the platform ABI from the module's physical triple.
type x86ABI uint8

const (
	x86SysV x86ABI = iota
	x86Win64
)

const x86VectorCall = llvm.CallConv(80)

func x86ABIFor(mod llvm.Module) x86ABI {
	if intllvm.IsWindowsTarget(mod.Target()) {
		return x86Win64
	}
	return x86SysV
}

func (abi x86ABI) queryReceiver(param llvm.Type) bool {
	// An empty Go receiver can survive as {} or be lowered to i32 on Win64.
	return (param.TypeKind() == llvm.StructTypeKind && param.StructElementTypesCount() == 0) ||
		(abi == x86Win64 && param.TypeKind() == llvm.IntegerTypeKind && param.IntTypeWidth() == 32)
}

// SetX86VectorCallConvention gives a compiler-generated vector instruction
// function the platform register convention, including XMM arguments on Win64.
func SetX86VectorCallConvention(mod llvm.Module, fn llvm.Value) {
	x86ABIFor(mod).applyWideCallingConvention(fn)
}

// Keep wide definitions, declarations and calls through function values in sync.
func (abi x86ABI) applyWideCallingConvention(v llvm.Value) {
	if abi == x86Win64 {
		if v.IsAFunction().IsNil() {
			v.SetInstructionCallConv(x86VectorCall)
		} else {
			v.SetFunctionCallConv(x86VectorCall)
		}
	}
}

func (abi x86ABI) forwardingKind(fn llvm.Value) llvm.TailCallKind {
	// Win64 vectorcall passes overflow vectors by reference to copies in
	// this frame. A tail transfer would release them before the callee runs.
	if abi == x86Win64 && fn.ParamsCount() > 6 {
		for _, arg := range fn.GlobalValueType().ParamTypes()[6:] {
			if arg.TypeKind() == llvm.VectorTypeKind {
				return llvm.TailCallKindNoTail
			}
		}
	}
	return llvm.TailCallKindMustTail
}

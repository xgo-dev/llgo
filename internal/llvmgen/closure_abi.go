package llvmgen

import (
	"strings"

	intllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
	"github.com/xgo-dev/llvm"
)

// closureEnvABI describes only the physical transport of an environment
// parameter. The environment is not part of a Go or go/types signature.
type closureEnvABI uint8

const (
	// closureEnvExplicit is the typed fallback used by WebAssembly and targets
	// for which no hidden parameter transport has been validated.
	closureEnvExplicit closureEnvABI = iota
	closureEnvNest
	closureEnvSwiftSelf
)

func closureEnvABIForTarget(triple string) closureEnvABI {
	triple = strings.ToLower(triple)
	arch, _, _ := strings.Cut(triple, "-")
	// Select the long-term machine ABI by physical target even when LLGo does
	// not yet support the target OS. FFI final-hop support may follow later
	// without changing compiled closure entries.
	switch {
	case arch == "arm64", arch == "arm64_32", arch == "aarch64", arch == "aarch64_be":
		// Keep a stable runtime ABI across LLVM versions on platforms which
		// reserve X18. swiftself uses the callee-saved X20 register and is also
		// usable by the libffi bridge without rebuilding libffi.
		if aarch64UsesSwiftSelf(triple) {
			return closureEnvSwiftSelf
		}
		return closureEnvNest
	case strings.HasPrefix(arch, "arm"), strings.HasPrefix(arch, "thumb"):
		// LLVM lowers swiftself through the platform's dedicated self register.
		// This keeps the ordinary C arguments in their normal ABI locations.
		return closureEnvSwiftSelf
	case arch == "x86_64", arch == "amd64",
		arch == "x86", arch == "386",
		arch == "i386", arch == "i486", arch == "i586", arch == "i686":
		return closureEnvNest
	case arch == "riscv32", arch == "riscv64":
		// LLVM and libffi both lower the RISC-V static chain through t2 (x7),
		// allowing the direct ffi_call_go path. The ESP32-C3 suite exercises
		// this transport on riscv32.
		return closureEnvNest
	default:
		return closureEnvExplicit
	}
}

func aarch64UsesSwiftSelf(triple string) bool {
	// Apple, Android, and Windows reserve X18, so LLGo uses LLVM's
	// swiftself/X20 transport there.
	return intllvm.IsDarwinTarget(triple) ||
		strings.Contains(triple, "android") ||
		strings.Contains(triple, "windows") ||
		strings.Contains(triple, "win32") ||
		strings.Contains(triple, "mingw")
}

func (p *Target) closureEnvABI() closureEnvABI {
	triple := p.LLVMTarget
	if triple == "" {
		triple = p.Spec().Triple
	}
	return closureEnvABIForTarget(triple)
}

// ClosureEnvBuildTag selects the runtime half of the same physical ABI used
// by the backend. It must be added after a named target has resolved its real
// LLVM triple; GOARCH may only be a package-selection compatibility value.
func (p *Target) ClosureEnvBuildTag() string {
	switch p.closureEnvABI() {
	case closureEnvNest:
		return "llgo_closure_env_nest"
	case closureEnvSwiftSelf:
		return "llgo_closure_env_swiftself"
	default:
		return "llgo_closure_env_explicit"
	}
}

func (p Program) closureEnvABI() closureEnvABI {
	return p.Target().closureEnvABI()
}

func (p Program) closureEnvAttribute() llvm.Attribute {
	var name string
	switch p.closureEnvABI() {
	case closureEnvNest:
		name = "nest"
	case closureEnvSwiftSelf:
		name = "swiftself"
	default:
		return llvm.Attribute{}
	}
	return p.ctx.CreateEnumAttribute(llvm.AttributeKindID(name), 0)
}

func (p Program) markClosureEnvFunction(fn llvm.Value, physicalIndex int) {
	attr := p.closureEnvAttribute()
	if attr.IsNil() {
		return
	}
	fn.AddAttributeAtIndex(physicalIndex+1, attr)
}

func (p Program) markClosureEnvCall(call llvm.Value, physicalIndex int) {
	attr := p.closureEnvAttribute()
	if attr.IsNil() {
		return
	}
	call.AddCallSiteAttribute(physicalIndex+1, attr)
}

// hideClosureCodeIdentity keeps LLVM from devirtualizing a native funcval call
// across the intentionally different IR prototypes of env and no-env entries.
// The empty tied-register asm is a machine-code no-op: it returns the same code
// pointer, but LLVM can no longer reinterpret a known no-env body as though the
// hidden environment were an ordinary first argument.
func (b Builder) hideClosureCodeIdentity(fn Expr) Expr {
	ftype := llvm.FunctionType(fn.Type.ll, []llvm.Type{fn.Type.ll}, false)
	asm := llvm.InlineAsm(ftype, "", "=r,0", false, false, llvm.InlineAsmDialectATT, false)
	return Expr{
		b.impl.CreateCall(ftype, asm, []llvm.Value{fn.impl}, "__llgo_funcval_code"),
		fn.Type,
	}
}

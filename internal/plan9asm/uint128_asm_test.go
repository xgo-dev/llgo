package plan9asm

import (
	"testing"

	"github.com/xgo-dev/llvm"
)

// modernc.org/libc's 128-bit overflow wrapper passes two Uint128 values
// through an ABI0 frame before calling its Go implementation.
// Y__ is the assembly wrapper; X__ declares its Go call target.
// This test checks signature lowering, not ABI0 call execution (plan9asm#44).
func TestTranslateUint128ABI0Wrapper(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/libc", `package libc
type Uint128 struct { Lo, Hi uint64 }
func Y__builtin_mul_overflowUint128(t *byte, a, b Uint128, res uintptr) int32
func X__builtin_mul_overflowUint128(t *byte, a, b Uint128, res uintptr) int32
`)
	asm := []byte(`TEXT ·Y__builtin_mul_overflowUint128(SB),$56-52
GO_ARGS
NO_LOCAL_POINTERS
MOVQ t+0(FP), AX
MOVQ AX, 0(SP)
MOVQ a_Lo+8(FP), AX
MOVQ AX, 8(SP)
MOVQ a_Hi+16(FP), AX
MOVQ AX, 16(SP)
MOVQ b_Lo+24(FP), AX
MOVQ AX, 24(SP)
MOVQ b_Hi+32(FP), AX
MOVQ AX, 32(SP)
MOVQ res+40(FP), AX
MOVQ AX, 40(SP)
CALL ·X__builtin_mul_overflowUint128(SB)
MOVL 48(SP), AX
MOVL AX, _3+48(FP)
RET
`)
	tr, err := TranslateSourceModuleForPkg(pkg, "abi0_linux_amd64.s", asm, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid LLVM module: %v", err)
	}
	wrapper := tr.Module.NamedFunction("example.com/libc.Y__builtin_mul_overflowUint128")
	if wrapper.IsNil() || wrapper.ParamsCount() != 4 {
		t.Fatal("missing wrapper or incorrect parameter count")
	}
	for i, name := range []string{"a", "b"} {
		arg := wrapper.Param(i + 1)
		if arg.Type().TypeKind() != llvm.StructTypeKind {
			t.Fatalf("%s is not a struct: %s", name, arg.Type())
		}
		fields := arg.Type().StructElementTypes()
		i64 := tr.Module.Context().Int64Type()
		if len(fields) != 2 || fields[0] != i64 || fields[1] != i64 {
			t.Fatalf("%s does not contain two i64 fields: %s", name, arg.Type())
		}
		var extracted [2]bool
		for block := wrapper.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if inst.InstructionOpcode() != llvm.ExtractValue || inst.Operand(0) != arg {
					continue
				}
				indices := inst.Indices()
				if len(indices) == 1 && indices[0] < 2 {
					extracted[indices[0]] = true
				}
			}
		}
		if extracted != [2]bool{true, true} {
			t.Fatalf("%s fields extracted = %v, want both Lo and Hi", name, extracted)
		}
	}
}

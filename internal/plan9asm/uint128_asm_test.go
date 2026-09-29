package plan9asm

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// modernc.org/libc's 128-bit overflow wrapper passes two Uint128 values
// through an ABI0 frame before calling its Go implementation.
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
	ir := tr.Module.String()
	if !strings.Contains(ir, "extractvalue { i64, i64 } %arg1, 0") || !strings.Contains(ir, "extractvalue { i64, i64 } %arg2, 1") {
		t.Fatalf("missing 128-bit argument fields in LLVM IR:\n%s", ir)
	}
}

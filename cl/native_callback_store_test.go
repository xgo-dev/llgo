package cl

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestNativeCallbackStoreWidth(t *testing.T) {
	const source = `package callback
import _ "unsafe"
//go:linkname Callback C.native_callback
var Callback func(int32) int32
func Reset() { Callback = nil }
func Capture() func() {
	saved := Callback
	return func() { Callback = saved }
}
`
	ssaPkg, _, files := buildGoSSAPkg(t, source)
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-w64-windows-gnu"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-w64-windows-gnu"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.LLVMTarget, func(t *testing.T) {
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			slot := mod.NamedGlobal("native_callback")
			if slot.IsNil() || slot.GlobalValueType().TypeKind() != llvm.PointerTypeKind {
				t.Fatal("native callback must occupy one pointer")
			}
			for _, name := range []string{"callback.Reset", "callback.Capture$1"} {
				fn := mustNamedFunction(t, mod, name)
				stores := 0
				for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
					for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
						if inst.InstructionOpcode() == llvm.Store && inst.Operand(1) == slot {
							stores++
							if inst.Operand(0).Type().TypeKind() != llvm.PointerTypeKind {
								t.Fatalf("callback store overwrites adjacent native storage:\n%s", fn.String())
							}
						}
					}
				}
				if stores != 1 {
					t.Fatalf("expected one callback store in %s, got %d", name, stores)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}

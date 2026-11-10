package plan9asm

import (
	"strings"
	"testing"

	llvm "github.com/xgo-dev/llvm"
)

const nativeTailFixture = `TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-16
	MOVD target+0(FP), R27
	MOVD context+8(FP), R0
	MOVD R29, 16(R0)
	MOVD RSP, R19
	MOVD R19, 24(R0)
	MOVD R30, 32(R0)
	JMP (R27)
`

func TestNativeTailTransfer(t *testing.T) {
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()
	llvm.InitializeAllAsmParsers()
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target, context uintptr)\n")
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(nativeTailFixture), goos, "arm64")
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			ir := tr.Module.String()
			for _, want := range []string{"native.tail:", "ldr x27, [sp, #8]", "ldr x0, [sp, #16]", "str x29, [x0, #16]", "str x30, [x0, #32]", "br x27", "bl Lnative_func_0${:uid}", "~{x19}", "~{v8}", "~{lr}", "ret void", "noinline"} {
				if !strings.Contains(ir, want) {
					t.Fatalf("missing %q:\n%s", want, ir)
				}
			}
			target, err := llvm.GetTargetFromTriple(tr.Module.Target())
			if err != nil {
				t.Fatal(err)
			}
			machine := target.CreateTargetMachine(tr.Module.Target(), "generic", "", llvm.CodeGenLevelDefault, llvm.RelocPIC, llvm.CodeModelDefault)
			defer machine.Dispose()
			object, err := machine.EmitToMemoryBuffer(tr.Module, llvm.ObjectFile)
			if err != nil {
				t.Fatal(err)
			}
			object.Dispose()
		})
	}
}

func TestNativeTailTransferRejectsUnsupportedABI(t *testing.T) {
	for _, tc := range []struct{ name, decl, source string }{
		{"result", "func Enter(target uintptr) uintptr", "TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-16\nMOVD target+0(FP), R27\nJMP (R27)\n"},
		{"frame", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "$0-16", "$16-16", 1)},
		{"narrow", "func Enter(target uintptr, context uint32)", nativeTailFixture},
		{"flags", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "NOSPLIT|NOFRAME", "NOSPLIT", 1)},
		{"unknown flag", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "NOSPLIT|NOFRAME", "NOSPLIT|NOFRAME|DUPOK", 1)},
		{"pseudo SP", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "RSP", "SP", 1)},
		{"late frame load", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tMOVD RSP, R19\n\tMOVD context+8(FP), R0\n\tMOVD R29,", 1)},
		{"nonterminal transfer", "func Enter(target, context uintptr)", nativeTailFixture + "\tRET\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := mustTestPackage(t, "example.com/tail", "package tail\n"+tc.decl+"\n")
			tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(tc.source), "linux", "arm64")
			if err == nil {
				tr.Module.Dispose()
				t.Fatal("unsupported native tail ABI accepted")
			}
		})
	}
}

func TestNativeTailTransferControlFlow(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target, context uintptr)\n")
	source := strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tCBZ R0, transfer\n\tMOVD R29,", 1)
	source = strings.Replace(source, "\tJMP (R27)", "transfer:\n\tJMP (R27)", 1)
	tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

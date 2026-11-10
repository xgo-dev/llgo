package plan9asm

import (
	"fmt"
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
			if strings.Contains(ir, "@GC_do_blocking") {
				t.Fatal("collector-free translation acquired a BDWGC dependency")
			}
			if got := strings.Contains(ir, "~{x18}"); got != (goos == "linux") {
				t.Fatalf("x18 clobber = %t on %s", got, goos)
			}
			for _, want := range []string{"native.tail:", "ldr x27, [sp, #8]", "ldr x0, [sp, #16]", "str x29, [x0, #16]", "str x30, [x0, #32]", "br x27", "bl Lnative_func_0${:uid}", "~{x19}", "~{v8}", "~{lr}", "ret void", "noinline"} {
				if !strings.Contains(ir, want) {
					t.Fatalf("missing %q:\n%s", want, ir)
				}
			}
			emitNativeTailObject(t, tr.Module)
		})
	}
}

func TestNativeTailTransferBDWGC(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target *byte, context uintptr)\nvar entryAddr uintptr\n")
	source := nativeTailFixture + "DATA ·entryAddr(SB)/8, $·Enter(SB)\nGLOBL ·entryAddr(SB), NOPTR, $8\n"
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "tail_arm64.s", []byte(source), goos, "arm64", TranslateOptions{NativeBDWGC: true})
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			entry := tr.Module.NamedFunction("example.com/tail.Enter")
			if ir := entry.String(); !strings.Contains(ir, "@GC_do_blocking") || strings.Contains(ir, "asm sideeffect") {
				t.Fatalf("public entry bypasses collector transition:\n%s", ir)
			}
			global := tr.Module.NamedGlobal("example.com/tail.entryAddr")
			if global.IsNil() || global.Initializer().IsNil() || strings.Contains(global.Initializer().String(), "$native") {
				t.Fatalf("assembly DATA bypasses public collector transition:\n%s", tr.Module.String())
			}
			emitNativeTailObject(t, tr.Module)
		})
	}
}

func TestNativeTailTransferBDWGCArgumentAttributes(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target *byte)\n")
	source := "TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-8\nMOVD target+0(FP), R27\nJMP (R27)\n"
	tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	native := tr.Module.NamedFunction("example.com/tail.Enter")
	kind := llvm.AttributeKindID("nonnull")
	native.AddAttributeAtIndex(1, tr.Module.Context().CreateEnumAttribute(kind, 0))
	wrapNativeTailBDWGC(tr.Module, native)
	entry := tr.Module.NamedFunction("example.com/tail.Enter")
	if entry.GetEnumAttributeAtIndex(1, kind).IsNil() {
		t.Fatal("public collector wrapper dropped its argument guarantee")
	}
	callback := tr.Module.NamedFunction("example.com/tail.Enter$blocked")
	if ir := callback.String(); !strings.Contains(ir, `(ptr nonnull `) {
		t.Fatalf("native forwarding call dropped its argument guarantee:\n%s", ir)
	}
	if !entry.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() || native.GetEnumFunctionAttribute(llvm.AttributeKindID("noinline")).IsNil() {
		t.Fatal("physical noinline policy was not kept at the native boundary")
	}
	if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	emitNativeTailObject(t, tr.Module)
}

func TestNativeTailTransferRejectsUnsupportedABI(t *testing.T) {
	for _, tc := range []struct{ name, decl, source string }{
		{"result", "func Enter(target uintptr) uintptr", "TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-16\nMOVD target+0(FP), R27\nJMP (R27)\n"},
		{"frame", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "$0-16", "$16-16", 1)},
		{"narrow", "func Enter(target uintptr, context uint32)", nativeTailFixture},
		{"flags", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "NOSPLIT|NOFRAME", "NOSPLIT", 1)},
		{"implicit flags", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "NOSPLIT|NOFRAME, ", "", 1)},
		{"unknown flag", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "NOSPLIT|NOFRAME", "NOSPLIT|NOFRAME|DUPOK", 1)},
		{"pseudo SP", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "RSP", "SP", 1)},
		{"reserved body register", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tMOVD R18, R5\n\tMOVD R29,", 1)},
		{"late frame load", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tMOVD RSP, R19\n\tMOVD context+8(FP), R0\n\tMOVD R29,", 1)},
		{"nonterminal transfer", "func Enter(target, context uintptr)", nativeTailFixture + "\tRET\n"},
		{"multiple transfers", "func Enter(target, context uintptr)", strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tJMP (R27)\n\tMOVD R29,", 1)},
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

func TestNativeTailTransferArgumentLimits(t *testing.T) {
	for _, argc := range []int{0, 1, 8, 9} {
		t.Run(fmt.Sprint(argc), func(t *testing.T) {
			var names []string
			for i := 0; i < argc; i++ {
				names = append(names, fmt.Sprintf("arg%d", i))
			}
			params := ""
			load := "MOVD $4096, R27"
			if argc != 0 {
				params = strings.Join(names, ", ") + " uintptr"
				load = "MOVD arg0+0(FP), R27"
			}
			pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter("+params+")\n")
			source := fmt.Sprintf("TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-%d\n%s\nJMP (R27)\n", argc*8, load)
			tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
			if argc > 8 {
				if err == nil {
					tr.Module.Dispose()
					t.Fatal("stack-passed arguments accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			emitNativeTailObject(t, tr.Module)
		})
	}
}

func TestNativeTailTransferHonorsGoBinderFilter(t *testing.T) {
	pkg := mustTestPackage(t, "syscall", "package syscall\nfunc rawVforkSyscall(target, context uintptr)\nfunc Kept()\n")
	source := strings.Replace(nativeTailFixture, "·Enter", "·rawVforkSyscall", 1)
	source += "TEXT ·Kept(SB), NOSPLIT, $0-0\nRET\n"
	tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	if fn := tr.Module.NamedFunction("syscall.rawVforkSyscall"); !fn.IsNil() && !fn.IsDeclaration() {
		t.Fatal("runtime-provided entry acquired a second assembly body")
	}
}

func TestNativeTailTransferControlFlow(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target, context uintptr)\n")
	for _, label := range []string{"transfer___tail_uid__", "Lnative_func_0", "Ltail_done"} {
		t.Run(label, func(t *testing.T) {
			source := strings.Replace(nativeTailFixture, "\tMOVD R29,", "\tCBZ R0, "+label+"\n\tMOVD R29,", 1)
			source = strings.Replace(source, "\tJMP (R27)", label+":\n\tJMP (R27)", 1)
			tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if ir := tr.Module.String(); strings.Contains(ir, "___tail_uid__") || !strings.Contains(ir, "Lnative_0_label_0${:uid}") {
				t.Fatalf("source label was not normalized and uniquified:\n%s", ir)
			}
			emitNativeTailObject(t, tr.Module)
		})
	}
}

func emitNativeTailObject(t *testing.T, mod llvm.Module) {
	t.Helper()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmPrinters()
	llvm.InitializeAllAsmParsers()
	target, err := llvm.GetTargetFromTriple(mod.Target())
	if err != nil {
		t.Fatal(err)
	}
	machine := target.CreateTargetMachine(mod.Target(), "generic", "", llvm.CodeGenLevelDefault, llvm.RelocPIC, llvm.CodeModelDefault)
	defer machine.Dispose()
	object, err := machine.EmitToMemoryBuffer(mod, llvm.ObjectFile)
	if err != nil {
		t.Fatal(err)
	}
	object.Dispose()
}

func TestNativeTailTransferRejectsSpecialTargetRegisters(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Enter(target, context uintptr)\n")
	for _, reg := range []string{"R18", "R29", "R30"} {
		t.Run(reg, func(t *testing.T) {
			source := strings.Replace(nativeTailFixture, "JMP (R27)", "JMP ("+reg+")", 1)
			tr, err := TranslateSourceModuleForPkg(pkg, "tail_arm64.s", []byte(source), "linux", "arm64")
			if err == nil {
				tr.Module.Dispose()
				t.Fatal("special target register accepted")
			}
			if !strings.Contains(err.Error(), "unsupported tail register") {
				t.Fatal(err)
			}
		})
	}
}

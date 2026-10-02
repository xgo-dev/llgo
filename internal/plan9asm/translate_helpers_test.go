package plan9asm

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llpackages "github.com/xgo-dev/llgo/internal/packages"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

func TestTranslateGOAMD64CPUDetectionLevel(t *testing.T) {
	pkg := mustTestPackage(t, "internal/cpu", "package cpu\nfunc getGOAMD64level() int32\n")
	asm := []byte(`TEXT ·getGOAMD64level(SB),NOSPLIT,$0-4
#ifdef GOAMD64_v4
 MOVL $4, ret+0(FP)
#else
#ifdef GOAMD64_v3
 MOVL $3, ret+0(FP)
#else
#ifdef GOAMD64_v2
 MOVL $2, ret+0(FP)
#else
 MOVL $1, ret+0(FP)
#endif
#endif
#endif
 RET
`)
	for _, level := range []string{"", "v1", "v2", "v3", "v4"} {
		t.Run(level, func(t *testing.T) {
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "cpu_x86.s", asm, "linux", "amd64", TranslateOptions{GOAMD64: level})
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			want := byte('1')
			if level != "" {
				want = level[1]
			}
			ir := tr.Module.NamedFunction("internal/cpu.getGOAMD64level").String()
			if !strings.Contains(ir, fmt.Sprintf("ret i32 %c", want)) {
				t.Fatalf("wrong CPU baseline for %q:\n%s", level, ir)
			}
		})
	}
}

func mustTestPackage(t *testing.T, pkgPath, src string) *llpackages.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "in.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse file: %v", err)
	}
	info := &types.Info{
		Defs: map[*ast.Ident]types.Object{},
		Uses: map[*ast.Ident]types.Object{},
	}
	conf := types.Config{Importer: importer.Default()}
	tpkg, err := conf.Check(pkgPath, fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("type-check package: %v", err)
	}
	return &llpackages.Package{
		ID:        pkgPath,
		Name:      f.Name.Name,
		PkgPath:   pkgPath,
		Fset:      fset,
		Syntax:    []*ast.File{f},
		Types:     tpkg,
		TypesInfo: info,
		Imports:   map[string]*llpackages.Package{},
	}
}

func TestTranslateWrappersAndErrors(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/foo", `package foo
func Foo()
`)
	asmPath := filepath.Join(t.TempDir(), "foo_amd64.s")
	asm := []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\n\tRET\n")

	if _, err := TranslateFileForPkgWithOptions(nil, asmPath, "linux", "amd64", nil, TranslateOptions{}); err == nil {
		t.Fatal("TranslateFileForPkgWithOptions(nil) should fail")
	}
	if _, err := TranslateSourceModuleForPkgWithOptions(&llpackages.Package{}, asmPath, asm, "linux", "amd64", TranslateOptions{}); err == nil {
		t.Fatal("TranslateSourceModuleForPkgWithOptions(empty pkg) should fail")
	}
	if _, err := TranslateSourceModuleForPkgWithOptions(&llpackages.Package{PkgPath: "example.com/bad"}, asmPath, asm, "linux", "amd64", TranslateOptions{}); err == nil {
		t.Fatal("TranslateSourceModuleForPkgWithOptions(missing types) should fail")
	}

	overlay := map[string][]byte{asmPath: asm}
	fileTr, err := TranslateFileForPkgWithOptions(pkg, asmPath, "linux", "amd64", overlay, TranslateOptions{AnnotateSource: true})
	if err != nil {
		t.Fatalf("TranslateFileForPkgWithOptions: %v", err)
	}
	if fileTr.LLVMIR == "" {
		t.Fatal("TranslateFileForPkgWithOptions returned empty LLVM IR")
	}
	if got := len(fileTr.Functions); got != 1 {
		t.Fatalf("function count = %d, want 1", got)
	}
	if got := fileTr.Functions[0].ResolvedSymbol; got != "example.com/foo.Foo" {
		t.Fatalf("resolved symbol = %q, want %q", got, "example.com/foo.Foo")
	}
	if _, ok := fileTr.Signatures["example.com/foo.Foo"]; !ok {
		t.Fatal("missing signature for example.com/foo.Foo")
	}

	fileTr2, err := TranslateSourceForPkg(pkg, asmPath, asm, "linux", "amd64")
	if err != nil {
		t.Fatalf("TranslateSourceForPkg: %v", err)
	}
	if len(fileTr2.Functions) != 1 {
		t.Fatalf("TranslateSourceForPkg function count = %d, want 1", len(fileTr2.Functions))
	}

	modTr, err := TranslateSourceModuleForPkg(pkg, asmPath, asm, "linux", "amd64")
	if err != nil {
		t.Fatalf("TranslateSourceModuleForPkg: %v", err)
	}
	defer modTr.Module.Dispose()
	if got := len(modTr.Functions); got != 1 {
		t.Fatalf("TranslateSourceModuleForPkg function count = %d, want 1", got)
	}
}

func TestTranslateGOARMTargetTriple(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/arm", `package arm
func Foo()
`)
	asmPath := filepath.Join(t.TempDir(), "foo_arm.s")
	asm := []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\n\tRET\n")
	tr, err := TranslateSourceModuleForPkgWithOptions(pkg, asmPath, asm, "linux", "arm", TranslateOptions{GOARM: "6,softfloat"})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	if got, want := tr.Module.Target(), "armv6-unknown-linux-gnueabi"; got != want {
		t.Fatalf("module target = %q, want %q", got, want)
	}
}

func TestTranslateX87Mode(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/x87", `package x87
func Round(value float64) int64
`)
	asm := []byte(`TEXT ·Round(SB),NOSPLIT,$0-16
	FMOVD value+0(FP), F0
	FRNDINT
	FMOVVP F0, ret+8(FP)
	RET
`)

	tests := []struct {
		name       string
		mode       extplan9asm.X87Mode
		want       string
		unwanted   string
		wantErrSub string
	}{
		{name: "default hardware", mode: extplan9asm.X87Auto, want: `asm sideeffect "fldl $1; frndint; fstpl $0"`, unwanted: "call double @llvm.floor.f64"},
		{name: "software fallback", mode: extplan9asm.X87Software, want: "call double @llvm.floor.f64", unwanted: "frndint"},
		{name: "invalid", mode: extplan9asm.X87Mode(255), wantErrSub: "invalid x87 mode"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, "round_386.s", asm, "windows", "386", TranslateOptions{X87Mode: test.mode})
			if test.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrSub) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Module.Dispose()
			ir := tr.Module.String()
			if !strings.Contains(ir, test.want) {
				t.Fatalf("IR missing %q:\n%s", test.want, ir)
			}
			if strings.Contains(ir, test.unwanted) {
				t.Fatalf("IR unexpectedly contains %q:\n%s", test.unwanted, ir)
			}
		})
	}
}

func TestTranslateMainPackageUsesCompilerSymbolPath(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/cmd", `package main
var value uint64
`)
	asm := []byte("DATA ·value(SB)/8, $42\nGLOBL ·value(SB),8,$8\n")
	tr, err := TranslateSourceModuleForPkg(pkg, "main_arm64.s", asm, "windows", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()

	if tr.Module.NamedGlobal("main.value").IsNil() {
		t.Fatalf("translated main package is missing main.value:\n%s", tr.Module.String())
	}
	if !tr.Module.NamedGlobal("example.com/cmd.value").IsNil() {
		t.Fatalf("translated main package retained module-qualified symbol:\n%s", tr.Module.String())
	}
}

func TestTranslateHelperFunctions(t *testing.T) {
	if got := StripABISuffix("runtime·cmpstring<ABIInternal>"); got != "runtime·cmpstring" {
		t.Fatalf("StripABISuffix runtime = %q", got)
	}
	if got := StripABISuffix("cmpbody<>"); got != "cmpbody" {
		t.Fatalf("StripABISuffix local helper = %q", got)
	}

	resolvePkg := ResolveSymFunc("example.com/foo")
	if got := resolvePkg("·Foo<ABIInternal>"); got != "example.com/foo.Foo" {
		t.Fatalf("ResolveSymFunc local = %q", got)
	}
	if got := resolvePkg("runtime·bar"); got != "runtime.bar" {
		t.Fatalf("ResolveSymFunc remote = %q", got)
	}
	if got := resolvePkg("plain"); got != "example.com/foo.plain" {
		t.Fatalf("ResolveSymFunc plain = %q", got)
	}

	resolveBytealg := ResolveSymFunc("internal/bytealg")
	if got := resolveBytealg("runtime·cmpstring<ABIInternal>"); got != "runtime.cmpstring" {
		t.Fatalf("ResolveSymFunc bytealg runtime = %q", got)
	}
	if got := resolveBytealg("cmpbody<>"); got != "internal/bytealg.cmpbody" {
		t.Fatalf("ResolveSymFunc bytealg helper = %q", got)
	}
	if got := resolveBytealg("·Count"); got != "internal/bytealg.Count" {
		t.Fatalf("ResolveSymFunc bytealg local = %q", got)
	}
	resolveDarwinSyscall := resolveSymFuncForTarget("syscall", "darwin", "arm64")
	if got := resolveDarwinSyscall("·RawSyscall"); got != "syscall.RawSyscall" {
		t.Fatalf("resolveSymFuncForTarget darwin RawSyscall = %q", got)
	}
	if got := resolveDarwinSyscall("·RawSyscall6"); got != "syscall.RawSyscall6" {
		t.Fatalf("resolveSymFuncForTarget darwin RawSyscall6 = %q", got)
	}

	if shouldKeepResolvedFunc("syscall", "linux", "amd64", "syscall.rawVforkSyscall") {
		t.Fatal("linux rawVforkSyscall should be filtered")
	}
	if !shouldKeepResolvedFunc("syscall", "linux", "amd64", "syscall.Syscall") {
		t.Fatal("normal syscall symbol should be kept")
	}
	if !shouldKeepResolvedFunc("syscall", "darwin", "arm64", "syscall.RawSyscall") {
		t.Fatal("darwin RawSyscall should be kept")
	}
	if !shouldKeepResolvedFunc("syscall", "darwin", "amd64", "syscall.RawSyscall6") {
		t.Fatal("darwin RawSyscall6 should be kept")
	}

	if got := FilterFuncs("syscall", "linux", "amd64", nil, resolvePkg); got != nil {
		t.Fatalf("FilterFuncs(nil) = %#v, want nil", got)
	}
	funcs := []extplan9asm.Func{{Sym: "·rawVforkSyscall"}, {Sym: "·Keep"}}
	filtered := FilterFuncs("syscall", "linux", "amd64", funcs, ResolveSymFunc("syscall"))
	if len(filtered) != 1 || filtered[0].Sym != "·Keep" {
		t.Fatalf("FilterFuncs linux = %#v, want only Keep", filtered)
	}
	darwinFuncs := []extplan9asm.Func{{Sym: "·RawSyscall"}, {Sym: "·RawSyscall6"}, {Sym: "·Syscall"}}
	darwinFiltered := FilterFuncs("syscall", "darwin", "arm64", darwinFuncs, resolveDarwinSyscall)
	if len(darwinFiltered) != len(darwinFuncs) {
		t.Fatalf("FilterFuncs darwin = %#v, want all kept", darwinFiltered)
	}

	tmpDir := t.TempDir()
	commentFile := filepath.Join(tmpDir, "comment.s")
	textFile := filepath.Join(tmpDir, "text.s")
	if err := os.WriteFile(commentFile, []byte("// comment only\n"), 0o644); err != nil {
		t.Fatalf("write comment file: %v", err)
	}
	if err := os.WriteFile(textFile, []byte("TEXT ·Foo(SB),NOSPLIT,$0-0\nRET\n"), 0o644); err != nil {
		t.Fatalf("write text file: %v", err)
	}
	if b, err := ReadFileWithOverlay(map[string][]byte{textFile: []byte("overlay")}, textFile); err != nil || string(b) != "overlay" {
		t.Fatalf("ReadFileWithOverlay overlay = %q, %v", string(b), err)
	}
	if b, err := ReadFileWithOverlay(nil, commentFile); err != nil || string(b) != "// comment only\n" {
		t.Fatalf("ReadFileWithOverlay file = %q, %v", string(b), err)
	}
	if ok, err := HasAnyTextAsm(nil, []string{commentFile}); err != nil || ok {
		t.Fatalf("HasAnyTextAsm(no text) = %v, %v", ok, err)
	}
	if ok, err := HasAnyTextAsm(nil, []string{commentFile, textFile}); err != nil || !ok {
		t.Fatalf("HasAnyTextAsm(with text) = %v, %v", ok, err)
	}
	if _, err := HasAnyTextAsm(nil, []string{filepath.Join(tmpDir, "missing.s")}); err == nil {
		t.Fatal("HasAnyTextAsm(missing file) should fail")
	}
}

func TestExtraAsmSigsAndDeclMap(t *testing.T) {
	generic := extraAsmSigsAndDeclMap("other/pkg", "amd64")
	if got, want := len(generic), 1; got != want {
		t.Fatalf("generic manual sig count = %d, want %d: %#v", got, want, generic)
	}
	memmove, ok := generic["runtime.memmove"]
	if !ok {
		t.Fatal("missing generic memmove signature")
	}
	if memmove.Name != "memmove" {
		t.Fatalf("runtime.memmove declaration = %q, want memmove", memmove.Name)
	}

	i386 := extraAsmSigsAndDeclMap("internal/bytealg", "386")
	for _, name := range []string{
		"internal/bytealg.cmpbody",
		"internal/bytealg.memeqbody",
	} {
		if _, ok := i386[name]; !ok {
			t.Fatalf("missing 386 manual sig %s", name)
		}
	}

	arm := extraAsmSigsAndDeclMap("internal/bytealg", "arm")
	for _, name := range []string{
		"internal/bytealg.cmpbody",
		"internal/bytealg.memeqbody",
		"internal/bytealg.countbytebody",
		"internal/bytealg.indexbytebody",
	} {
		if _, ok := arm[name]; !ok {
			t.Fatalf("missing arm manual sig %s", name)
		}
	}

	arm64 := extraAsmSigsAndDeclMap("internal/bytealg", "arm64")
	for _, name := range []string{
		"internal/bytealg.cmpbody",
		"internal/bytealg.memeqbody",
		"internal/bytealg.countbytebody",
		"internal/bytealg.indexbody",
		"internal/bytealg.indexbytebody",
	} {
		if _, ok := arm64[name]; !ok {
			t.Fatalf("missing arm64 manual sig %s", name)
		}
	}

	amd64 := extraAsmSigsAndDeclMap("internal/bytealg", "amd64")
	for _, name := range []string{
		"internal/bytealg.cmpbody",
		"internal/bytealg.countbody",
		"internal/bytealg.indexbody",
		"internal/bytealg.indexbytebody",
		"internal/bytealg.memeqbody",
	} {
		if _, ok := amd64[name]; !ok {
			t.Fatalf("missing amd64 manual sig %s", name)
		}
	}

	scan := extraAsmSigsAndDeclMap("internal/runtime/gc/scan", "amd64")
	if got, want := len(scan), 27; got != want {
		t.Fatalf("scan manual sig count = %d, want %d", got, want)
	}
	for _, name := range []string{
		"internal/runtime/gc/scan.expandAVX512_1",
		"internal/runtime/gc/scan.expandAVX512_32",
		"internal/runtime/gc/scan.expandAVX512_64",
	} {
		sig, ok := scan[name]
		if !ok {
			t.Fatalf("missing scan manual sig %s", name)
		}
		if len(sig.Args) != 1 || sig.Args[0] != extplan9asm.Ptr {
			t.Fatalf("%s args = %#v, want [Ptr]", name, sig.Args)
		}
		if sig.Ret != extplan9asm.Void {
			t.Fatalf("%s ret = %#v, want Void", name, sig.Ret)
		}
		if len(sig.ArgRegs) != 1 || sig.ArgRegs[0] != extplan9asm.AX {
			t.Fatalf("%s arg regs = %#v, want [AX]", name, sig.ArgRegs)
		}
	}
}

package plan9asm

import (
	"context"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	llvm "github.com/xgo-dev/llvm"
	asm "github.com/xgo-dev/plan9asm"
)

func TestPreprocessAndSDKVersionGuards(t *testing.T) {
	dir := t.TempDir()
	outsideDir := t.TempDir()
	outsideHeader := filepath.Join(outsideDir, "secret.h")
	if err := os.WriteFile(outsideHeader, []byte("#define SECRET 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkHeader := filepath.Join(dir, "escape.h")
	if err := os.Symlink(outsideHeader, linkHeader); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	pkg := mustTestPackage(t, "example.com/pre", "package pre\nfunc F()\n")
	pkg.Dir = dir

	// Symlink escaping allowed roots must fail.
	asmSrc := []byte("#include \"escape.h\"\nTEXT ·F(SB),$0-0\nRET\n")
	if _, err := PreprocessSourceForPkg(pkg, "f.s", asmSrc, "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()}); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected symlink escape error, got %v", err)
	}

	// Self-referencing symlink triggers EvalSymlinks non-ErrNotExist error.
	loopHeader := filepath.Join(dir, "loop.h")
	if err := os.Symlink(loopHeader, loopHeader); err == nil {
		if _, err := PreprocessSourceForPkg(pkg, "f.s", []byte("#include \"loop.h\"\nTEXT ·F(SB),$0-0\nRET\n"), "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()}); err == nil {
			t.Fatal("expected symlink loop error")
		}
	}

	// HasAnyTextAsmForPkg true path and error path.
	goodAsm := filepath.Join(dir, "ok.s")
	if err := os.WriteFile(goodAsm, []byte("TEXT ·F(SB),$0-0\nRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	has, err := HasAnyTextAsmForPkg(pkg, []string{goodAsm}, "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()})
	if err != nil || !has {
		t.Fatalf("HasAnyTextAsmForPkg = %v, %v, want true, nil", has, err)
	}
	if _, err := HasAnyTextAsmForPkg(pkg, []string{filepath.Join(dir, "missing.s")}, "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()}); err == nil {
		t.Fatal("expected error from HasAnyTextAsmForPkg for missing file")
	}

	// readAssemblyFile on directory fails (not regular file).
	if _, err := readAssemblyFile(nil, dir); err == nil {
		t.Fatal("expected error reading directory as assembly file")
	}

	// assemblySDKVersion with zbootstrap.go containing non-GenDecl (func) and non-ValueSpec (type) before const version.
	fakeSDK := t.TempDir()
	zbDir := filepath.Join(fakeSDK, "src", "internal", "buildcfg")
	if err := os.MkdirAll(zbDir, 0o700); err != nil {
		t.Fatal(err)
	}
	zbSrc := "package buildcfg\ntype helper int\nfunc init() {}\nconst version = \"go1.27.0\"\n"
	if err := os.WriteFile(filepath.Join(zbDir, "zbootstrap.go"), []byte(zbSrc), 0o600); err != nil {
		t.Fatal(err)
	}
	if ver, err := assemblySDKVersion(fakeSDK); err != nil || ver != "go1.27.0" {
		t.Fatalf("assemblySDKVersion(zbootstrap) = %q, %v", ver, err)
	}

	// assemblyExperimentDefines with non-VAR decl and non-ValueSpec in pkgspecial.go.
	asmMainDir := filepath.Join(fakeSDK, "src", "cmd", "asm")
	objabiDir := filepath.Join(fakeSDK, "src", "cmd", "internal", "objabi")
	_ = os.MkdirAll(asmMainDir, 0o700)
	_ = os.MkdirAll(objabiDir, 0o700)
	_ = os.WriteFile(filepath.Join(asmMainDir, "main.go"), []byte(`package main
func f() {
	if objabi.LookupPkgSpecial(ctxt.Pkgpath).AllowAsmABI {
		_ = "GOEXPERIMENT_"
	}
}
`), 0o600)
	_ = os.WriteFile(filepath.Join(objabiDir, "pkgspecial.go"), []byte(`package objabi
type dummy int
func init() {}
var allowAsmABIPkgs = []string{"runtime"}
`), 0o600)
	defs, err := assemblyExperimentDefines(fakeSDK, "runtime", []string{"goexperiment.simd"})
	if err != nil || len(defs) != 1 || defs[0] != "GOEXPERIMENT_simd" {
		t.Fatalf("assemblyExperimentDefines = %v, %v", defs, err)
	}
}

func TestFrameMetadataAndBindAMD64ExtraGuards(t *testing.T) {
	// frameMetadataChecksum rejects missing h1: prefix.
	if frameMetadataChecksum("sha256:abc") {
		t.Fatal("expected false for missing h1: prefix")
	}
	// prepareDownloadedFrameMetadata fails when temporary root does not exist.
	if err := prepareDownloadedFrameMetadata(context.Background(), "go", filepath.Join(t.TempDir(), "no-such-dir"), nil, "example.com/m", "v1.0.0"); err == nil {
		t.Fatal("expected error for missing temporary directory")
	}

	// bindAMD64Frame hitting invalid JSON from go env (Unix shell script) and SDK path inside GOROOT/src.
	pkg, _ := amd64SDKFrameFixture(t)
	resolve := ResolveSymFunc(pkg.PkgPath)
	goPkg := asm.GoPackage{Path: pkg.PkgPath, Types: pkg.Types, Syntax: pkg.Syntax}
	physSrc := []byte("TEXT ·Copy(SB),4,$16-24\nMOVQ $1, 0(SP)\nRET\n")

	if runtime.GOOS != "windows" {
		fakeRoot := t.TempDir()
		binDir := filepath.Join(fakeRoot, "bin")
		_ = os.MkdirAll(binDir, 0o700)
		fakeGo := filepath.Join(binDir, "go")
		_ = os.WriteFile(fakeGo, []byte("#!/bin/sh\necho 'not-valid-json'\n"), 0o700)
		if _, _, err := bindAMD64Frame(pkg, physSrc, "copy_amd64.s", "linux", "amd64", TranslateOptions{
			GOROOT:      fakeRoot,
			Environment: []string{"PATH=" + os.Getenv("PATH")},
		}, goPkg, "", resolve); err == nil {
			t.Fatal("expected JSON unmarshal error from fake go env")
		}
	}

	// SDK directory inside runtime.GOROOT()/src exercises CaptureGoAMD64SDKFrameSource branch (amd64_frame.go:144).
	sdkDir := filepath.Join(runtime.GOROOT(), "src", "internal", "bytealg")
	if entries, err := os.ReadDir(sdkDir); err == nil {
		var asmFile string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), "_amd64.s") {
				asmFile = e.Name()
				break
			}
		}
		if asmFile != "" {
			sdkPkg := &packages.Package{
				ID:      "internal/bytealg",
				PkgPath: "internal/bytealg",
				Dir:     sdkDir,
				Types:   types.NewPackage("internal/bytealg", "bytealg"),
			}
			_, finish, _ := bindAMD64Frame(sdkPkg, physSrc, asmFile, "linux", "amd64", TranslateOptions{
				GOROOT: runtime.GOROOT(),
			}, asm.GoPackage{Path: "internal/bytealg", Types: sdkPkg.Types}, "x86_64-unknown-linux-gnu", ResolveSymFunc("internal/bytealg"))
			if finish != nil {
				_ = finish()
			}
		}
	}

	// declareNativeTailEntries skips non-matching TEXT symbols in the same file (native_tail.go:77).
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("skip_non_tail")
	defer mod.Dispose()
	tr := &asm.GoModuleTranslation{Module: mod, Signatures: map[string]asm.FuncSig{}}
	srcTwo := "TEXT ·Other(SB), NOSPLIT, $0-0\nRET\n" + nativeTailFixture
	sig := asm.FuncSig{Name: "example.com/tail.Enter", Args: []asm.LLVMType{asm.Ptr, asm.I64}, Ret: asm.Void}
	if err := declareNativeTailEntries(tr, map[string]asm.FuncSig{sig.Name: sig}, []byte(srcTwo), ResolveSymFunc("example.com/tail")); err != nil {
		t.Fatalf("declareNativeTailEntries with extra TEXT failed: %v", err)
	}

	// TranslateSourceModuleForPkgWithOptions where a Go-ABI caller invokes a physical tail target
	// exercises manualSig -> physicalTails[resolved] (translate.go:157-158).
	callerPkg := mustTestPackage(t, "example.com/tail", "package tail\nfunc Caller(a, b uintptr)\nfunc Enter(target, context uintptr)\n")
	callerSrc := "TEXT ·Caller(SB), NOSPLIT, $16-16\nMOVD a+0(FP), R0\nMOVD b+8(FP), R1\nMOVD R0, 8(RSP)\nMOVD R1, 16(RSP)\nCALL ·Enter(SB)\nRET\n" + nativeTailFixture
	if tr2, err := TranslateSourceModuleForPkg(callerPkg, "tail_arm64.s", []byte(callerSrc), "linux", "arm64"); err == nil {
		tr2.Module.Dispose()
	}

	// Conflicting DATA definition for the same symbol triggers declareNativeTailEntries error inside TranslateSourceModuleForPkg (translate.go:235-237).
	conflictSrc := nativeTailFixture + "DATA ·Enter(SB)/8, $1\nGLOBL ·Enter(SB), $8\n"
	if tr3, err := TranslateSourceModuleForPkg(callerPkg, "tail_arm64.s", []byte(conflictSrc), "linux", "arm64"); err == nil {
		tr3.Module.Dispose()
		t.Fatal("expected declareNativeTailEntries conflict error against DATA ·Enter")
	}
}

func TestPreprocessBudgetAndReadErrorGuards(t *testing.T) {
	// Generated go_asm.h exceeding remaining maxAssemblyInputBytes triggers charge(data) error.
	dir := t.TempDir()
	pkg := mustTestPackage(t, "example.com/budget", "package budget\ntype S struct{ A, B, C, D uint64 }\nfunc F()\n")
	pkg.Dir = dir
	src := []byte("#include \"h1.h\"\n#include \"h2.h\"\n#include \"h3.h\"\n#include \"h4.h\"\n#include \"go_asm.h\"\nTEXT ·F(SB),$0-0\nRET\n")
	// Leave only 10 bytes of the 16 MiB input budget so h1..h4 succeed and generated go_asm.h triggers charge(data).
	chunk := make([]byte, maxAssemblySourceBytes)
	for i := range chunk {
		chunk[i] = ' '
	}
	chunk[len(chunk)-1] = '\n'
	tailLen := maxAssemblySourceBytes - len(src) - 10
	tailChunk := append([]byte(nil), chunk[:tailLen]...)
	tailChunk[len(tailChunk)-1] = '\n'
	overlay := map[string][]byte{
		filepath.Join(dir, "h1.h"): chunk,
		filepath.Join(dir, "h2.h"): chunk,
		filepath.Join(dir, "h3.h"): chunk,
		filepath.Join(dir, "h4.h"): tailChunk,
	}
	if _, err := PreprocessSourceForPkg(pkg, "f.s", src, "linux", "amd64", TranslateOptions{
		GOROOT:  runtime.GOROOT(),
		Overlay: overlay,
	}); err == nil || !strings.Contains(err.Error(), "input byte bound") {
		t.Fatalf("PreprocessSourceForPkg(go_asm.h over budget) err = %v, want input byte bound error", err)
	}

	// readAssemblyFile fails on os.Open after os.Stat succeeds for an unreadable regular file.
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		noRead := filepath.Join(dir, "unreadable.s")
		if err := os.WriteFile(noRead, []byte("TEXT ·F(SB),$0-0\nRET\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(noRead, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(noRead, 0o600) })
		if _, err := readAssemblyFile(nil, noRead); err == nil {
			t.Fatal("expected readAssemblyFile error on unreadable 0000 file")
		}
	}
}

func TestAMD64FrameTranslateAndNativeTailErrorGuards(t *testing.T) {
	physSrc := []byte("TEXT ·Copy(SB),4,$16-24\nMOVQ $1, 0(SP)\nRET\n")
	pkg, _ := amd64SDKFrameFixture(t)
	resolve := ResolveSymFunc(pkg.PkgPath)
	goPkg := asm.GoPackage{Path: pkg.PkgPath, Types: pkg.Types, Syntax: pkg.Syntax}

	// bindAMD64Frame empty filename -> assemblyPackageFile error (amd64_frame.go:47-48).
	if _, _, err := bindAMD64Frame(pkg, physSrc, "", "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()}, goPkg, "", resolve); err == nil {
		t.Fatal("expected assemblyPackageFile error for empty filename")
	}

	// bindAMD64Frame nonexistent pkg.Dir -> physicalFramePath error (amd64_frame.go:51-52).
	missingDirPkg := *pkg
	missingDirPkg.Dir = filepath.Join(t.TempDir(), "does-not-exist")
	if _, _, err := bindAMD64Frame(&missingDirPkg, physSrc, "copy_amd64.s", "linux", "amd64", TranslateOptions{GOROOT: runtime.GOROOT()}, goPkg, "", resolve); err == nil {
		t.Fatal("expected physicalFramePath error for missing pkg.Dir")
	}

	// bindAMD64Frame invalid temp dir -> os.MkdirTemp error (amd64_frame.go:121-123).
	t.Run("bindAMD64Frame MkdirTemp error", func(t *testing.T) {
		badTmp := filepath.Join(t.TempDir(), "missing-tmp")
		t.Setenv("TMPDIR", badTmp)
		t.Setenv("TMP", badTmp)
		t.Setenv("TEMP", badTmp)
		absCache := t.TempDir()
		env := []string{
			"GOPATH=" + absCache,
			"GOMODCACHE=" + absCache,
			"GOCACHE=" + absCache,
		}
		if _, _, err := bindAMD64Frame(pkg, physSrc, "copy_amd64.s", "linux", "amd64", TranslateOptions{
			GOROOT:      runtime.GOROOT(),
			Environment: env,
		}, goPkg, "", resolve); err == nil {
			t.Fatal("expected os.MkdirTemp error in bindAMD64Frame")
		}
	})

	// bindAMD64Frame with a variadic Go function triggers DeriveGoFuncABI0Signature error (amd64_frame.go:166-167).
	t.Run("bindAMD64Frame variadic signature error", func(t *testing.T) {
		dir := t.TempDir()
		goMod := "module example.com/varframe\n\ngo 1.24\n"
		goSrc := "package varframe\n\nfunc Var(args ...uintptr)\n"
		asmSrc := "TEXT ·Var(SB), 4, $16-24\nMOVQ $1, 0(SP)\nRET\n"
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600); err != nil {
			t.Fatal(err)
		}
		goPath := filepath.Join(dir, "var.go")
		if err := os.WriteFile(goPath, []byte(goSrc), 0o600); err != nil {
			t.Fatal(err)
		}
		asmPath := filepath.Join(dir, "var_amd64.s")
		if err := os.WriteFile(asmPath, []byte(asmSrc), 0o600); err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		syntax, err := parser.ParseFile(fset, goPath, goSrc, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Defs: make(map[*ast.Ident]types.Object), Uses: make(map[*ast.Ident]types.Object)}
		sizes := types.SizesFor("gc", "amd64")
		typed, err := (&types.Config{Sizes: sizes, Importer: importer.Default()}).Check("example.com/varframe", fset, []*ast.File{syntax}, info)
		if err != nil {
			t.Fatal(err)
		}
		varPkg := &packages.Package{
			ID: typed.Path(), PkgPath: typed.Path(), Name: typed.Name(), Dir: dir,
			Fset: fset, Types: typed, TypesInfo: info, TypesSizes: sizes,
			Syntax: []*ast.File{syntax}, GoFiles: []string{goPath}, CompiledGoFiles: []string{goPath},
			OtherFiles: []string{asmPath},
		}
		_, finish, err := bindAMD64Frame(varPkg, []byte(asmSrc), "var_amd64.s", "linux", "amd64",
			TranslateOptions{GOROOT: runtime.GOROOT()},
			asm.GoPackage{Path: typed.Path(), Types: typed, Syntax: varPkg.Syntax},
			"x86_64-unknown-linux-gnu", ResolveSymFunc(typed.Path()))
		if finish != nil {
			_ = finish()
		}
		if err == nil || !strings.Contains(err.Error(), "variadic") {
			t.Fatalf("bindAMD64Frame(variadic) err = %v, want variadic error", err)
		}
	})

	// TranslateSourceModuleForPkg with PkgPath="runtime" (no Imports["runtime"]) and ARM64 memmove
	// triggers manualErr != nil while TranslateGoModule succeeded (tr != nil -> disposes tr.Module, translate.go:196-197).
	rtPkg := mustTestPackage(t, "runtime", "package runtime\nimport \"unsafe\"\nfunc memmove(to, from unsafe.Pointer, n uintptr)\n")
	if tr, err := TranslateSourceModuleForPkg(rtPkg, "memmove_arm64.s", []byte("TEXT ·memmove(SB), 4, $0-24\nRET\n"), "linux", "arm64"); err == nil {
		tr.Module.Dispose()
		t.Fatal("expected manualErr for runtime.memmove without runtime import")
	}

	// nativeTailSignatures with a variadic ARM64 function triggers DeriveGoFuncABI0Signature error (native_tail.go:54-55).
	varTailPkg := mustTestPackage(t, "example.com/vartail", "package vartail\nfunc Enter(args ...uintptr)\n")
	if _, _, err := nativeTailSignatures(varTailPkg, []byte("TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-24\nJMP (R0)\n"), ResolveSymFunc("example.com/vartail"), "linux", "arm64"); err == nil || !strings.Contains(err.Error(), "variadic") {
		t.Fatalf("nativeTailSignatures(variadic) err = %v, want variadic error", err)
	}
}

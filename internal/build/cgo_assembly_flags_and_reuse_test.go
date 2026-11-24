package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestCgoAssemblyInputsAndFingerprintShareCgoEnvFlags(t *testing.T) {
	ctx, pkg := assemblyLanguageTestFixture(t, "example.com/cgoflags", true, false)
	asm := assemblyLanguageTestWrite(t, pkg.Dir, "const_asm.S", cgoConstTestAssembly())
	pkg.OtherFiles = []string{asm}

	// Default (no CGO_CPPFLAGS): compilerArgs omits -DTEST_CONST.
	defaultInputs, err := cgoAssemblyInputs(ctx, pkg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultInputs) != 1 {
		t.Fatalf("default cgo assembly inputs = %#v, want 1", defaultInputs)
	}
	defaultManifest := newManifestBuilder()
	ctx.collectEnvInputs(defaultManifest)
	if err := ctx.collectPackageInputs(defaultManifest, &aPackage{Package: pkg}); err != nil {
		t.Fatal(err)
	}
	defaultFP := defaultManifest.Fingerprint()

	// With CGO_CPPFLAGS and CGO_CFLAGS: both cgoAssemblyInputs and cache
	// fingerprint observe the exact same effective flags and produce a new key.
	ctxOverride, pkgOverride := assemblyLanguageTestFixture(t, "example.com/cgoflags", true, false)
	asmOverride := assemblyLanguageTestWrite(t, pkgOverride.Dir, "const_asm.S", cgoConstTestAssembly())
	pkgOverride.OtherFiles = []string{asmOverride}
	ctxOverride.commands.environ = withEnv(ctxOverride.commands.environ,
		`CGO_CPPFLAGS=-DTEST_CONST=30 "-DTEST_LABEL=with space"`,
		`CGO_CFLAGS=-DTEST_EXTRA=7`,
	)
	overrideInputs, err := cgoAssemblyInputs(ctxOverride, pkgOverride, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(overrideInputs) != 1 {
		t.Fatalf("override cgo assembly inputs = %#v, want 1", overrideInputs)
	}
	wantPrefix := []string{"-DTEST_CONST=30", "-DTEST_LABEL=with space", "-DTEST_EXTRA=7"}
	if len(overrideInputs[0].compilerArgs) < len(wantPrefix) ||
		!reflect.DeepEqual(overrideInputs[0].compilerArgs[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("override compilerArgs = %#v, want prefix %#v", overrideInputs[0].compilerArgs, wantPrefix)
	}
	overrideManifest := newManifestBuilder()
	ctxOverride.collectEnvInputs(overrideManifest)
	if err := ctxOverride.collectPackageInputs(overrideManifest, &aPackage{Package: pkgOverride}); err != nil {
		t.Fatal(err)
	}
	if overrideManifest.Fingerprint() == defaultFP {
		t.Fatal("changing CGO_CPPFLAGS did not change package cache fingerprint")
	}

	// Malformed quoting in CGO_CPPFLAGS must fail closed instead of silently
	// dropping flags.
	ctxBad, pkgBad := assemblyLanguageTestFixture(t, "example.com/cgoflags", true, false)
	pkgBad.OtherFiles = []string{assemblyLanguageTestWrite(t, pkgBad.Dir, "const_asm.S", cgoConstTestAssembly())}
	ctxBad.commands.environ = withEnv(ctxBad.commands.environ, `CGO_CPPFLAGS="-DTEST_CONST=30`)
	if _, err := cgoAssemblyInputs(ctxBad, pkgBad, nil); err == nil || !strings.Contains(err.Error(), "CGO_CPPFLAGS") {
		t.Fatalf("malformed CGO_CPPFLAGS error = %v, want parse error", err)
	}
}

func TestCgoAssemblyCGO_CPPFLAGSBuildRunAndCacheInvalidation(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native cgo assembly execution test targets darwin/linux")
	}
	if runtime.GOARCH != "arm64" && runtime.GOARCH != "amd64" {
		t.Skip("native cgo assembly fixture covers arm64/amd64")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not found")
	}

	dir := t.TempDir()
	cacheDir := t.TempDir()
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()
	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("go.mod", "module example.com/cgocppflags\n\ngo 1.24\n")
	writeFile("const_asm.S", cgoConstTestAssembly())
	writeFile("main.go", `package main

/*
extern int asm_const(void);
*/
import "C"
import "fmt"

func main() {
	fmt.Println(int(C.asm_const()))
}
`)

	runGo := func(extraEnv ...string) string {
		t.Helper()
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = dir
		cmd.Env = withEnv(os.Environ(), append([]string{"CGO_ENABLED=1", "GOWORK=off"}, extraEnv...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go run (%v) failed: %v\n%s", extraEnv, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	runLLGo := func(name string, extraEnv ...string) string {
		t.Helper()
		t.Setenv("LLGO_BUILD_CACHE", "on")
		t.Setenv("CGO_ENABLED", "1")
		for _, kv := range []string{"CGO_CPPFLAGS", "CGO_CFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS"} {
			t.Setenv(kv, "")
		}
		for _, kv := range extraEnv {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				t.Fatalf("invalid env %q", kv)
			}
			t.Setenv(k, v)
		}
		outBin := filepath.Join(t.TempDir(), name)
		origDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(origDir) }()

		_, err = Do([]string{"."}, &Config{
			Mode:    ModeBuild,
			OutFile: outBin,
			Goos:    runtime.GOOS,
			Goarch:  runtime.GOARCH,
		})
		if err != nil {
			t.Fatalf("llgo build (%v) failed: %v", extraEnv, err)
		}
		out, err := exec.Command(outBin).CombinedOutput()
		if err != nil {
			t.Fatalf("run %s (%v) failed: %v\n%s", outBin, extraEnv, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	// 1. Default: both stock Go and LLGo return 10.
	if gotGo, gotLLGo := runGo("CGO_CPPFLAGS="), runLLGo("default"); gotGo != "10" || gotLLGo != "10" {
		t.Fatalf("default output: go=%q llgo=%q, want both \"10\"", gotGo, gotLLGo)
	}
	// 2. CGO_CPPFLAGS=-DTEST_CONST=30 on the SAME warm cache: both return 30.
	if gotGo, gotLLGo := runGo("CGO_CPPFLAGS=-DTEST_CONST=30"), runLLGo("cpp30", "CGO_CPPFLAGS=-DTEST_CONST=30"); gotGo != "30" || gotLLGo != "30" {
		t.Fatalf("CGO_CPPFLAGS=-DTEST_CONST=30 output: go=%q llgo=%q, want both \"30\"", gotGo, gotLLGo)
	}
	// 3. Returning to default hits/rebuilds cleanly to 10.
	if gotLLGo := runLLGo("default-again"); gotLLGo != "10" {
		t.Fatalf("return-to-default output: llgo=%q, want \"10\"", gotLLGo)
	}
}

func TestBuildReusesNativeAssemblyObjectAndPlan9Preprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell shim test targets Unix")
	}
	clangPath, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang not found")
	}

	t.Run("NativeAssemblyObjectReusedBetweenFingerprintAndCompile", func(t *testing.T) {
		dir := t.TempDir()
		logPath := filepath.Join(dir, "clang-calls.log")
		shim := filepath.Join(dir, "clang-shim.sh")
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\nexec %q \"$@\"\n", logPath, clangPath)
		if err := os.WriteFile(shim, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		asmPath := assemblyLanguageTestWrite(t, dir, "const_asm.S", cgoConstTestAssembly())
		ctx := &context{
			mode:             ModeBuild,
			buildConf:        &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
			crossCompile:     crosscompile.Export{CC: shim},
			commands:         commandEnv{dir: dir, environ: os.Environ()},
			nativeAsmObjects: make(map[llgoFileHashKey]string),
		}
		defer ctx.cleanupNativeAsmObjects()

		inputs := []llgoFileInput{{path: asmPath, compilerArgs: []string{"-DTEST_CONST=30"}}}
		digests, err := digestLLGoFileInputs(ctx, inputs, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(digests) != 1 || digests[0].ObjectHash == "" {
			t.Fatalf("unexpected digests = %#v", digests)
		}
		if len(ctx.nativeAsmObjects) != 1 {
			t.Fatalf("expected 1 retained native assembly object, got %d", len(ctx.nativeAsmObjects))
		}
		cachedPath := ""
		for _, p := range ctx.nativeAsmObjects {
			cachedPath = p
		}

		// Calling clFile with the identical inputs must reuse the retained .o
		// without invoking clang a second time.
		obj, err := clFile(ctx, inputs[0].compilerArgs, asmPath, filepath.Join(dir, "pkg.a"), "example.com/reuse", false)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(obj)
		if obj == cachedPath {
			t.Fatal("clFile returned shared cached object directly instead of a package-owned path")
		}
		objHash, err := digestFile(obj)
		if err != nil {
			t.Fatal(err)
		}
		if objHash != digests[0].ObjectHash {
			t.Fatalf("reused object hash = %q, want %q", objHash, digests[0].ObjectHash)
		}
		rawLog, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatal(err)
		}
		calls := strings.Split(strings.TrimSpace(string(rawLog)), "\n")
		if len(calls) != 1 {
			t.Fatalf("clang invocations = %d (%q), want 1 (fingerprint reused by clFile)", len(calls), calls)
		}

		// Cleanup removes the build-scoped retained object while leaving the
		// package-owned copy intact until the package removes it.
		ctx.cleanupNativeAsmObjects()
		if _, err := os.Stat(cachedPath); !os.IsNotExist(err) {
			t.Fatalf("retained cached object %q was not cleaned up: %v", cachedPath, err)
		}
		if _, err := os.Stat(obj); err != nil {
			t.Fatalf("package-owned copy %q disappeared on context cleanup: %v", obj, err)
		}
	})

	t.Run("Plan9AsmPreprocessReusedAcrossFingerprintAndCapture", func(t *testing.T) {
		ctx, pkg := assemblyLanguageTestFixture(t, "example.com/plan9reuse", false, false)
		ctx.plan9AsmSources = &sync.Map{}
		asm := assemblyLanguageTestWrite(t, pkg.Dir, "entry_amd64.s", "#include \"body.h\"\n")
		header := assemblyLanguageTestWrite(t, pkg.Dir, "body.h", "#define VALUE 11\nTEXT ·Foo(SB),4,$0-0\nMOVQ $VALUE, AX\nRET\n")
		pkg.OtherFiles = []string{asm, header}

		digests, err := digestPlan9AsmFiles(ctx, pkg, []string{asm})
		if err != nil {
			t.Fatal(err)
		}
		if len(digests) != 1 {
			t.Fatalf("digests = %#v, want 1", digests)
		}
		// Removing the included header proves captureSelectedPlan9Asm reuses
		// the in-build preprocessed source instead of re-expanding from disk.
		if err := os.Remove(header); err != nil {
			t.Fatal(err)
		}
		sources, err := captureSelectedPlan9Asm(ctx, pkg, []string{asm})
		if err != nil {
			t.Fatalf("captureSelectedPlan9Asm failed to reuse cached preprocessed source: %v", err)
		}
		if len(sources) != 1 || digestBytes(sources[0].source) != digests[0].SourceHash {
			t.Fatalf("captured source hash mismatch: got %q, want %q", digestBytes(sources[0].source), digests[0].SourceHash)
		}
	})
}

func cgoConstTestAssembly() string {
	switch runtime.GOARCH {
	case "arm64":
		return `#ifndef TEST_CONST
#define TEST_CONST 10
#endif
#if defined(__APPLE__)
#define ASM_SYM _asm_const
#else
#define ASM_SYM asm_const
#endif
.text
.globl ASM_SYM
ASM_SYM:
	mov w0, #TEST_CONST
	ret
`
	default:
		return `#ifndef TEST_CONST
#define TEST_CONST 10
#endif
#if defined(__APPLE__)
#define ASM_SYM _asm_const
#else
#define ASM_SYM asm_const
#endif
.text
.globl ASM_SYM
ASM_SYM:
	movl $TEST_CONST, %eax
	ret
`
	}
}

package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("go run (%v) failed: %v\nstdout:\n%s\nstderr:\n%s", extraEnv, err, stdout.String(), stderr.String())
		}
		return strings.TrimSpace(stdout.String())
	}

	runLLGo := func(name string, extraEnv ...string) string {
		t.Helper()
		t.Setenv("GOWORK", "off")
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
#if defined(__linux__) && defined(__ELF__)
.section .note.GNU-stack,"",@progbits
#endif
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
#if defined(__linux__) && defined(__ELF__)
.section .note.GNU-stack,"",@progbits
#endif
`
	}
}

func TestCgoCompilationFlagsPreservesGroupOrder(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	preambles := []cgoPreamble{{goFile: goFile}}
	decls := []cgoDecl{
		{cppflags: []string{"-DREVIEW_VALUE=20", "-DCPP_DECL=1"}},
		{cflags: []string{"-DC_DECL=1"}},
		{cxxflags: []string{"-DCXX_DECL=1"}},
		{ldflags: []string{"-ldecl"}},
	}
	commands := commandEnv{
		dir: dir,
		environ: []string{
			"CGO_CPPFLAGS=-DREVIEW_VALUE=10 -DCPP_ENV=1",
			"CGO_CFLAGS=-DREVIEW_VALUE=30 -DC_ENV=1",
			"CGO_CXXFLAGS=-DREVIEW_VALUE=35 -DCXX_ENV=1",
			"CGO_LDFLAGS=-lenv",
		},
	}
	cflags, cxxflags, ldflags, err := cgoCompilationFlags(commands, preambles, decls, true)
	if err != nil {
		t.Fatal(err)
	}
	incFlag := "-I" + dir + string(filepath.Separator)
	wantC := []string{
		"-DREVIEW_VALUE=10", "-DCPP_ENV=1",
		"-DREVIEW_VALUE=20", "-DCPP_DECL=1",
		"-DREVIEW_VALUE=30", "-DC_ENV=1",
		"-DC_DECL=1",
		incFlag,
	}
	wantCXX := []string{
		"-DREVIEW_VALUE=10", "-DCPP_ENV=1",
		"-DREVIEW_VALUE=20", "-DCPP_DECL=1",
		"-DREVIEW_VALUE=35", "-DCXX_ENV=1",
		"-DCXX_DECL=1",
		incFlag,
	}
	wantLD := []string{"-lenv", "-ldecl"}
	if !reflect.DeepEqual(cflags, wantC) {
		t.Fatalf("cflags = %#v, want %#v", cflags, wantC)
	}
	if !reflect.DeepEqual(cxxflags, wantCXX) {
		t.Fatalf("cxxflags = %#v, want %#v", cxxflags, wantCXX)
	}
	if !reflect.DeepEqual(ldflags, wantLD) {
		t.Fatalf("ldflags = %#v, want %#v", ldflags, wantLD)
	}

	// When includeEnvFlags is false (non-cgo native preambles or CGO_ENABLED=0),
	// #cgo directives and include dirs stay intact while CGO_* env vars are omitted.
	noEnvC, noEnvCXX, noEnvLD, err := cgoCompilationFlags(commands, preambles, decls, false)
	if err != nil {
		t.Fatal(err)
	}
	wantNoEnvC := []string{"-DREVIEW_VALUE=20", "-DCPP_DECL=1", "-DC_DECL=1", incFlag}
	wantNoEnvCXX := []string{"-DREVIEW_VALUE=20", "-DCPP_DECL=1", "-DCXX_DECL=1", incFlag}
	wantNoEnvLD := []string{"-ldecl"}
	if !reflect.DeepEqual(noEnvC, wantNoEnvC) || !reflect.DeepEqual(noEnvCXX, wantNoEnvCXX) || !reflect.DeepEqual(noEnvLD, wantNoEnvLD) {
		t.Fatalf("includeEnvFlags=false got (%#v, %#v, %#v), want (%#v, %#v, %#v)",
			noEnvC, noEnvCXX, noEnvLD, wantNoEnvC, wantNoEnvCXX, wantNoEnvLD)
	}
}

func TestCgoAssemblyFlagPrecedenceMatchesGo(t *testing.T) {
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
	writeFile("go.mod", "module example.com/cgoprecedence\n\ngo 1.24\n")
	writeFile("const_asm.S", strings.ReplaceAll(cgoConstTestAssembly(), "TEST_CONST", "REVIEW_VALUE"))
	writeFile("main.go", `package main

/*
#cgo CPPFLAGS: -DREVIEW_VALUE=20
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
		base := []string{"CGO_ENABLED=1", "GOWORK=off", "CGO_CPPFLAGS=", "CGO_CFLAGS=", "CGO_CXXFLAGS=", "CGO_LDFLAGS="}
		cmd.Env = withEnv(os.Environ(), append(base, extraEnv...)...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("go run (%v) failed: %v\nstdout:\n%s\nstderr:\n%s", extraEnv, err, stdout.String(), stderr.String())
		}
		return strings.TrimSpace(stdout.String())
	}

	runLLGo := func(name string, extraEnv ...string) string {
		t.Helper()
		t.Setenv("GOWORK", "off")
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

	cases := []struct {
		name string
		env  []string
		want string
	}{
		{name: "directive-cppflags-only", want: "20"},
		{name: "cgo-cflags-overrides-directive-cppflags", env: []string{"CGO_CFLAGS=-DREVIEW_VALUE=30"}, want: "30"},
		{name: "directive-cppflags-overrides-env-cppflags", env: []string{"CGO_CPPFLAGS=-DREVIEW_VALUE=40"}, want: "20"},
		{name: "return-to-directive-cppflags-only", want: "20"},
	}
	for _, tc := range cases {
		gotGo := runGo(tc.env...)
		gotLLGo := runLLGo(tc.name, tc.env...)
		if gotGo != tc.want || gotLLGo != tc.want {
			t.Fatalf("%s (%v): go=%q llgo=%q, want both %q", tc.name, tc.env, gotGo, gotLLGo, tc.want)
		}
	}
}

func TestPureGoAndDisabledCgoIgnoreCgoEnvFlags(t *testing.T) {
	t.Run("PureGoPackageWithCGO_LDFLAGS", func(t *testing.T) {
		ctx, pkg := assemblyLanguageTestFixture(t, "example.com/purego", false, false)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, pkg.GoFiles[0], nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg.Fset = fset
		pkg.Syntax = []*ast.File{file}
		aPkg := &aPackage{Package: pkg}

		baseManifest := newManifestBuilder()
		ctx.collectEnvInputs(baseManifest)
		if err := ctx.collectPackageInputs(baseManifest, aPkg); err != nil {
			t.Fatal(err)
		}
		baseFP := baseManifest.Fingerprint()

		ctx.commands.environ = withEnv(ctx.commands.environ,
			"CGO_ENABLED=1",
			"CGO_CPPFLAGS=-DUNUSED=1",
			"CGO_CFLAGS=-DUNUSED=2",
			"CGO_LDFLAGS=-lnonexistent_llgo_test_lib",
		)
		llfiles, ldflags, err := buildCgo(ctx, aPkg, pkg.Syntax, nil, false)
		if err != nil {
			t.Fatalf("buildCgo(pure Go) failed: %v", err)
		}
		if len(llfiles) != 0 || len(ldflags) != 0 {
			t.Fatalf("buildCgo(pure Go) = (%v, %v), want empty", llfiles, ldflags)
		}
		flagManifest := newManifestBuilder()
		ctx.collectEnvInputs(flagManifest)
		if err := ctx.collectPackageInputs(flagManifest, aPkg); err != nil {
			t.Fatal(err)
		}
		if len(flagManifest.pkg.CgoEnvFlags) != 0 {
			t.Fatalf("pure Go package recorded CgoEnvFlags = %v, want empty", flagManifest.pkg.CgoEnvFlags)
		}
		if flagManifest.Fingerprint() != baseFP {
			t.Fatal("pure Go fingerprint changed when CGO_* env flags were set")
		}
	})

	t.Run("CgoDisabledIgnoresAllCgoInputsAndFlags", func(t *testing.T) {
		ctx, pkg := assemblyLanguageTestFixture(t, "example.com/cgopkg", true, false)
		asm := assemblyLanguageTestWrite(t, pkg.Dir, "const_asm.S", cgoConstTestAssembly())
		pkg.OtherFiles = []string{asm}
		ctx.commands.environ = withEnv(ctx.commands.environ,
			"CGO_ENABLED=0",
			"CGO_CPPFLAGS=-DTEST_CONST=30",
			"CGO_LDFLAGS=-lnonexistent_llgo_test_lib",
		)
		if inputs, err := cgoAssemblyInputs(ctx, pkg, nil); err != nil || len(inputs) != 0 {
			t.Fatalf("cgoAssemblyInputs(CGO_ENABLED=0) = (%v, %v), want empty", inputs, err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), pkg.GoFiles[0], nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg.Fset = token.NewFileSet()
		pkg.Syntax = []*ast.File{file}
		if llfiles, ldflags, err := buildCgo(ctx, &aPackage{Package: pkg}, pkg.Syntax, nil, false); err != nil || len(llfiles) != 0 || len(ldflags) != 0 {
			t.Fatalf("buildCgo(CGO_ENABLED=0) = (%v, %v, %v), want empty", llfiles, ldflags, err)
		}
		m := newManifestBuilder()
		ctx.collectEnvInputs(m)
		if err := ctx.collectPackageInputs(m, &aPackage{Package: pkg}); err != nil {
			t.Fatal(err)
		}
		if len(m.pkg.CgoEnvFlags) != 0 {
			t.Fatalf("CGO_ENABLED=0 recorded CgoEnvFlags = %v, want empty", m.pkg.CgoEnvFlags)
		}
	})
}

func TestPureGoBuildIgnoresCgoLdflags(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native execution test targets darwin/linux")
	}
	dir := t.TempDir()
	cacheDir := t.TempDir()
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/purego42\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(42)\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, cgoEnabled := range []string{"0", "1"} {
		t.Run("CGO_ENABLED="+cgoEnabled, func(t *testing.T) {
			goCmd := exec.Command("go", "run", ".")
			goCmd.Dir = dir
			goCmd.Env = withEnv(os.Environ(),
				"GOWORK=off",
				"CGO_ENABLED="+cgoEnabled,
				"CGO_LDFLAGS=-lnonexistent_llgo_review_lib",
			)
			goOut, err := goCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go run failed: %v\n%s", err, goOut)
			}
			if got := strings.TrimSpace(string(goOut)); got != "42" {
				t.Fatalf("go run output = %q, want \"42\"", got)
			}

			t.Setenv("GOWORK", "off")
			t.Setenv("LLGO_BUILD_CACHE", "on")
			t.Setenv("CGO_ENABLED", cgoEnabled)
			t.Setenv("CGO_LDFLAGS", "-lnonexistent_llgo_review_lib")
			outBin := filepath.Join(t.TempDir(), "purego42")
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if _, err := Do([]string{"."}, &Config{
				Mode:    ModeBuild,
				OutFile: outBin,
				Goos:    runtime.GOOS,
				Goarch:  runtime.GOARCH,
			}); err != nil {
				t.Fatalf("llgo build (CGO_ENABLED=%s, bad CGO_LDFLAGS) failed: %v", cgoEnabled, err)
			}
			out, err := exec.Command(outBin).CombinedOutput()
			if err != nil {
				t.Fatalf("run %s failed: %v\n%s", outBin, err, out)
			}
			if got := strings.TrimSpace(string(out)); got != "42" {
				t.Fatalf("llgo output = %q, want \"42\"", got)
			}
		})
	}
}

func TestDigestNativeAssemblyObjectCleansTempOnFailure(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not found")
	}
	for _, retain := range []bool{false, true} {
		t.Run(fmt.Sprintf("retain=%v", retain), func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("TMPDIR", tmpDir)
			srcDir := t.TempDir()
			badAsm := assemblyLanguageTestWrite(t, srcDir, "broken.S", ".llgo_invalid_directive_for_cleanup_test\n")
			ctx := &context{
				mode:      ModeBuild,
				buildConf: &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH},
				commands:  commandEnv{dir: srcDir, environ: os.Environ()},
			}
			_, obj, err := digestNativeAssemblyObject(ctx, badAsm, nil, retain)
			if err == nil {
				t.Fatal("digestNativeAssemblyObject unexpectedly succeeded on invalid assembly")
			}
			if obj != "" {
				t.Fatalf("digestNativeAssemblyObject returned object path %q on error, want empty", obj)
			}
			leftovers, globErr := filepath.Glob(filepath.Join(tmpDir, "llgofile-*.o"))
			if globErr != nil {
				t.Fatal(globErr)
			}
			if len(leftovers) != 0 {
				t.Fatalf("failed digestNativeAssemblyObject leaked temporary objects: %v", leftovers)
			}
		})
	}
}

func unsetEnvForTest(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		prev, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, prev)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func TestCgoAssemblyGOENVFlagsBuildRunAndCacheInvalidation(t *testing.T) {
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
	goenvPath := filepath.Join(t.TempDir(), "goenv")
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()

	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("go.mod", "module example.com/cgogoenv\n\ngo 1.24\n")
	writeFile("const_asm.S", strings.ReplaceAll(cgoConstTestAssembly(), "TEST_CONST", "REVIEW_VALUE"))
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

	stripCgoVars := func(env []string) []string {
		out := make([]string, 0, len(env))
		for _, kv := range env {
			if strings.HasPrefix(kv, "CGO_CPPFLAGS=") ||
				strings.HasPrefix(kv, "CGO_CFLAGS=") ||
				strings.HasPrefix(kv, "CGO_CXXFLAGS=") ||
				strings.HasPrefix(kv, "CGO_LDFLAGS=") {
				continue
			}
			out = append(out, kv)
		}
		return out
	}

	runCase := func(name, goenvContent string, extraEnv ...string) (string, string) {
		t.Helper()
		if err := os.WriteFile(goenvPath, []byte(goenvContent), 0o600); err != nil {
			t.Fatal(err)
		}
		goCmd := exec.Command("go", "run", ".")
		goCmd.Dir = dir
		goCmd.Env = withEnv(stripCgoVars(os.Environ()), append([]string{
			"GOWORK=off",
			"CGO_ENABLED=1",
			"GOENV=" + goenvPath,
		}, extraEnv...)...)
		var goOut, goErr bytes.Buffer
		goCmd.Stdout = &goOut
		goCmd.Stderr = &goErr
		if err := goCmd.Run(); err != nil {
			t.Fatalf("go run (%s) failed: %v\nstderr:\n%s", name, err, goErr.String())
		}

		// Ensure CGO_* are unset in the process environment unless explicitly
		// supplied in extraEnv, so GOENV supplies them just as cmd/go sees them.
		unsetEnvForTest(t, "CGO_CPPFLAGS", "CGO_CFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS")
		t.Setenv("GOWORK", "off")
		t.Setenv("LLGO_BUILD_CACHE", "on")
		t.Setenv("CGO_ENABLED", "1")
		t.Setenv("GOENV", goenvPath)
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

		if _, err := Do([]string{"."}, &Config{
			Mode:    ModeBuild,
			OutFile: outBin,
			Goos:    runtime.GOOS,
			Goarch:  runtime.GOARCH,
		}); err != nil {
			t.Fatalf("llgo build (%s) failed: %v", name, err)
		}
		llgoOut, err := exec.Command(outBin).CombinedOutput()
		if err != nil {
			t.Fatalf("run %s failed: %v\n%s", outBin, err, llgoOut)
		}
		return strings.TrimSpace(goOut.String()), strings.TrimSpace(string(llgoOut))
	}

	cases := []struct {
		name         string
		goenvContent string
		extraEnv     []string
		want         string
	}{
		{
			name:         "goenv-cgo-cflags-30",
			goenvContent: "CGO_CFLAGS=-DREVIEW_VALUE=30\n",
			want:         "30",
		},
		{
			name:         "goenv-empty-default-10",
			goenvContent: "",
			want:         "10",
		},
		{
			name:         "process-env-overrides-goenv-40",
			goenvContent: "CGO_CFLAGS=-DREVIEW_VALUE=30\n",
			extraEnv:     []string{"CGO_CFLAGS=-DREVIEW_VALUE=40"},
			want:         "40",
		},
		{
			name:         "process-empty-falls-back-to-goenv-30",
			goenvContent: "CGO_CFLAGS=-DREVIEW_VALUE=30\n",
			extraEnv:     []string{"CGO_CFLAGS="},
			want:         "30",
		},
		{
			name:         "goenv-cgo-cflags-30-again",
			goenvContent: "CGO_CFLAGS=-DREVIEW_VALUE=30\n",
			want:         "30",
		},
	}
	for _, tc := range cases {
		gotGo, gotLLGo := runCase(tc.name, tc.goenvContent, tc.extraEnv...)
		if gotGo != tc.want || gotLLGo != tc.want {
			t.Fatalf("%s: go=%q llgo=%q, want both %q", tc.name, gotGo, gotLLGo, tc.want)
		}
	}
}

func TestLLGoUnsafePreambleWorksWithCgoDisabledAndEnabled(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native execution test targets darwin/linux")
	}
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not found")
	}

	dir := t.TempDir()
	cacheDir := t.TempDir()
	oldCacheRoot := cacheRootFunc
	cacheRootFunc = func() string { return cacheDir }
	defer func() { cacheRootFunc = oldCacheRoot }()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/unsafepreamble\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainSrc := `package main

/*
int review_probe(void) { return 7; }
*/
import "unsafe"

//go:linkname reviewProbe C.review_probe
func reviewProbe() int32

func main() {
	_ = unsafe.Sizeof(int32(0))
	println(reviewProbe())
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, cgoEnabled := range []string{"0", "1"} {
		t.Run("CGO_ENABLED="+cgoEnabled, func(t *testing.T) {
			t.Setenv("GOWORK", "off")
			t.Setenv("GOENV", "off")
			t.Setenv("LLGO_BUILD_CACHE", "on")
			t.Setenv("CGO_ENABLED", cgoEnabled)
			// Non-import-"C" LLGo native preambles must compile and run under both
			// CGO_ENABLED=0 and CGO_ENABLED=1 without inheriting CGO_LDFLAGS.
			t.Setenv("CGO_LDFLAGS", "-lnonexistent_llgo_review_lib")

			outBin := filepath.Join(t.TempDir(), "probe-"+cgoEnabled)
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if _, err := Do([]string{"."}, &Config{
				Mode:    ModeBuild,
				OutFile: outBin,
				Goos:    runtime.GOOS,
				Goarch:  runtime.GOARCH,
			}); err != nil {
				t.Fatalf("llgo build (unsafe preamble, CGO_ENABLED=%s) failed: %v", cgoEnabled, err)
			}
			out, err := exec.Command(outBin).CombinedOutput()
			if err != nil {
				t.Fatalf("run %s failed: %v\n%s", outBin, err, out)
			}
			if got := strings.TrimSpace(string(out)); got != "7" {
				t.Fatalf("llgo output (CGO_ENABLED=%s) = %q, want \"7\"", cgoEnabled, got)
			}
		})
	}
}

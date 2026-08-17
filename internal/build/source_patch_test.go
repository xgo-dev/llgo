package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/packages"
	llruntime "github.com/xgo-dev/llgo/runtime"
)

func TestWasmRuntimeSourcePatchTypeChecks(t *testing.T) {
	for _, goos := range []string{"js", "wasip1"} {
		t.Run(goos, func(t *testing.T) {
			cfgEnv := append(os.Environ(), "GOOS="+goos, "GOARCH=wasm")
			goroot, goversion, err := env.GOROOTAndGOVERSIONWithEnv(cfgEnv)
			if err != nil {
				t.Fatal(err)
			}
			overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
				goos:      goos,
				goarch:    "wasm",
				goversion: goversion,
			})
			if err != nil {
				t.Fatal(err)
			}

			pkgs, err := packages.LoadEx(nil, func(types.Sizes, string, string) types.Sizes {
				return &types.StdSizes{WordSize: 4, MaxAlign: 4}
			}, &packages.Config{
				Mode:    loadSyntax | packages.NeedDeps | packages.NeedModule | packages.NeedExportFile,
				Env:     cfgEnv,
				Fset:    token.NewFileSet(),
				Overlay: overlay,
			}, "runtime")
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 {
				t.Fatalf("loaded %d runtime packages, want 1", len(pkgs))
			}
			if pkgs[0].IllTyped {
				logPackageErrors(t, pkgs[0], make(map[string]bool))
				t.Fatal("runtime did not type-check with wasm32 sizes")
			}
		})
	}
}

func logPackageErrors(t *testing.T, pkg *packages.Package, seen map[string]bool) {
	t.Helper()
	if pkg == nil || seen[pkg.ID] {
		return
	}
	seen[pkg.ID] = true
	for _, err := range pkg.Errors {
		t.Log(err)
	}
	for _, imported := range pkg.Imports {
		if imported.IllTyped {
			logPackageErrors(t, imported, seen)
		}
	}
}

func TestWasmBytealgSourcePatchReplacesAsm(t *testing.T) {
	for _, pkgPath := range []string{"internal/bytealg", "internal/chacha8rand", "internal/runtime/atomic"} {
		if !llruntime.HasSourcePatchPkg(pkgPath) {
			t.Fatalf("%s should be registered as a source patch package", pkgPath)
		}
		if !llruntime.SourcePatchReplacesAsmForGOARCH(pkgPath, "wasm") {
			t.Fatalf("%s wasm assembly should be replaced by its source patch", pkgPath)
		}
		if llruntime.SourcePatchReplacesAsmForGOARCH(pkgPath, "arm64") {
			t.Fatalf("%s native assembly should remain enabled", pkgPath)
		}
	}

	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{
		goos:      "js",
		goarch:    "wasm",
		goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		"internal/bytealg/compare_wasm.s",
		"internal/bytealg/equal_wasm.s",
		"internal/bytealg/indexbyte_wasm.s",
		"internal/chacha8rand/chacha8_stub.s",
		"internal/runtime/atomic/atomic_wasm.s",
	} {
		path := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(file))
		if got := string(overlay[path]); got != "// replaced by LLGo source patch\n" {
			t.Fatalf("overlay[%q] = %q, want assembly replacement", path, got)
		}
	}
}

func TestSyncAtomicSourcePatchReplacesAsm(t *testing.T) {
	for _, goarch := range []string{
		"386", "amd64", "arm", "arm64", "loong64", "mips", "mips64", "mips64le",
		"ppc64", "ppc64le", "riscv64", "s390x", "wasm",
	} {
		if !llruntime.SourcePatchReplacesAsmForGOARCH("sync/atomic", goarch) {
			t.Fatalf("sync/atomic assembly should be replaced on %s", goarch)
		}
	}

	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	asmFile := filepath.Join(runtime.GOROOT(), "src", "sync", "atomic", "asm.s")
	if got := string(overlay[asmFile]); got != "// replaced by LLGo source patch\n" {
		t.Fatalf("overlay[%q] = %q, want assembly replacement", asmFile, got)
	}
	patchFile := filepath.Join(runtime.GOROOT(), "src", "sync", "atomic", "z_llgo_patch_atomic.go")
	if got := string(overlay[patchFile]); !strings.Contains(got, "//go:linkname LoadPointer llgo.atomicLoad") {
		t.Fatalf("overlay[%q] does not contain atomic intrinsic linkname:\n%s", patchFile, got)
	}
}

func TestSyncPoolSourcePatchUsesStdlibQueue(t *testing.T) {
	const pkgPath = "sync"
	if !llruntime.HasSourcePatchPkg(pkgPath) {
		t.Fatal("sync should be registered as a source patch package")
	}
	if llruntime.HasAltPkg(pkgPath) {
		t.Fatal("sync should not remain an alt package")
	}

	changed, overlay, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), pkgPath, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || len(files) != 1 {
		t.Fatalf("sync patch changed = %v, files = %v, want one selected patch", changed, files)
	}

	patchFile := filepath.Join(runtime.GOROOT(), "src", "sync", "z_llgo_patch_pool.go")
	patchSrc := string(overlay[patchFile])
	if !strings.Contains(patchSrc, "func runtime_poolLocalAlloc") {
		t.Fatalf("overlay[%q] does not contain the Pool TLS hooks:\n%s", patchFile, patchSrc)
	}
	if strings.Contains(patchSrc, "runtime/internal/clite/tls") {
		t.Fatalf("overlay[%q] adds a private TLS dependency", patchFile)
	}

	queueFile := filepath.Join(runtime.GOROOT(), "src", "sync", "poolqueue.go")
	if _, ok := overlay[queueFile]; ok {
		t.Fatalf("official Pool queue implementation should remain unchanged: %s", queueFile)
	}
}

func TestSyscallSourcePatchPreservesTargetImplementations(t *testing.T) {
	const pkgPath = "syscall"
	if !llruntime.HasSourcePatchPkg(pkgPath) {
		t.Fatal("syscall should be registered as a source patch package")
	}
	if llruntime.HasAltPkg(pkgPath) {
		t.Fatal("syscall should not remain an alt package")
	}

	for _, target := range []struct {
		goos   string
		goarch string
		asm    []string
	}{
		{goos: "darwin", goarch: "amd64", asm: []string{"asm_darwin_amd64.s", "zsyscall_darwin_amd64.s"}},
		{goos: "darwin", goarch: "arm64", asm: []string{"asm_darwin_arm64.s", "zsyscall_darwin_arm64.s"}},
		{goos: "linux", goarch: "amd64", asm: []string{"asm_linux_amd64.s"}},
		{goos: "linux", goarch: "arm64", asm: []string{"asm_linux_arm64.s"}},
	} {
		t.Run(target.goos+"-"+target.goarch, func(t *testing.T) {
			changed, overlay, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), pkgPath, sourcePatchBuildContext{
				goos:      target.goos,
				goarch:    target.goarch,
				goversion: runtime.Version(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !changed || len(files) != 2 {
				t.Fatalf("syscall patch changed = %v, files = %v, want two selected patches", changed, files)
			}
			for _, file := range files {
				patchFile := filepath.Join(runtime.GOROOT(), "src", "syscall", "z_llgo_patch_"+filepath.Base(file))
				if strings.Contains(string(overlay[patchFile]), "github.com/xgo-dev/llgo/runtime") {
					t.Fatalf("overlay[%q] adds a private runtime dependency", patchFile)
				}
			}
			for _, name := range target.asm {
				asmFile := filepath.Join(runtime.GOROOT(), "src", "syscall", name)
				if got := string(overlay[asmFile]); got != "// replaced by LLGo source patch\n" {
					t.Fatalf("overlay[%q] = %q, want assembly replacement", asmFile, got)
				}
			}
		})
	}

	changed, _, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), pkgPath, sourcePatchBuildContext{
		goos:      "wasip1",
		goarch:    "wasm",
		goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed || len(files) != 0 {
		t.Fatalf("wasm syscall patch changed = %v, files = %v, want official implementation", changed, files)
	}
}

func TestArmBaremetalAtomicSourcePatchReplacesAsm(t *testing.T) {
	const pkgPath = "internal/runtime/atomic"
	if !llruntime.HasSourcePatchPkg(pkgPath) {
		t.Fatal("internal/runtime/atomic should be registered as a source patch package")
	}
	if llruntime.HasAltPkg(pkgPath) {
		t.Fatal("internal/runtime/atomic should not remain an alt package")
	}
	if !llruntime.SourcePatchReplacesAsmForGOARCH(pkgPath, "arm") {
		t.Fatal("internal/runtime/atomic ARM assembly should be replaceable by its source patch")
	}

	runtimeDir := env.LLGoRuntimeDir()
	ctx := sourcePatchBuildContext{
		goos:       "linux",
		goarch:     "arm",
		goversion:  runtime.Version(),
		buildFlags: []string{"-tags=llgo,baremetal"},
	}
	changed, overlay, files, err := applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), pkgPath, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || len(files) != 1 {
		t.Fatalf("baremetal ARM patch changed = %v, files = %v, want one selected patch", changed, files)
	}
	asmFile := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic", "atomic_arm.s")
	if got := string(overlay[asmFile]); got != "// replaced by LLGo source patch\n" {
		t.Fatalf("overlay[%q] = %q, want assembly replacement", asmFile, got)
	}

	ctx.buildFlags = []string{"-tags=llgo"}
	changed, _, files, err = applySourcePatchForPkg(nil, nil, runtimeDir, runtime.GOROOT(), pkgPath, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if changed || len(files) != 0 {
		t.Fatalf("non-baremetal ARM patch changed = %v, files = %v, want no selected patch", changed, files)
	}
}

func TestUniqueSourcePatchUsesStdlibDependencyGraph(t *testing.T) {
	const pkgPath = "unique"
	if !llruntime.HasSourcePatchPkg(pkgPath) {
		t.Fatal("unique should be registered as a source patch package")
	}
	if llruntime.HasAltPkg(pkgPath) {
		t.Fatal("unique should not remain an alt package")
	}

	for _, tc := range []struct {
		version   string
		wantFiles int
	}{
		{version: "go1.24.0", wantFiles: 1},
		{version: "go1.25.0", wantFiles: 1},
		{version: "go1.26.0", wantFiles: 2},
	} {
		t.Run(tc.version, func(t *testing.T) {
			changed, overlay, files, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), pkgPath, sourcePatchBuildContext{
				goos:      runtime.GOOS,
				goarch:    runtime.GOARCH,
				goversion: tc.version,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !changed || len(files) != tc.wantFiles {
				t.Fatalf("unique patch changed = %v, files = %v, want %d selected patches", changed, files, tc.wantFiles)
			}
			for _, file := range files {
				src := string(overlay[filepath.Join(runtime.GOROOT(), "src", "unique", "z_llgo_patch_"+filepath.Base(file))])
				if strings.Contains(src, "github.com/xgo-dev/llgo/runtime/abi") {
					t.Fatalf("source patch %s adds the private runtime/abi dependency", file)
				}
			}
			clonePatch := filepath.Join(runtime.GOROOT(), "src", "unique", "z_llgo_patch_clone.go")
			if got := string(overlay[clonePatch]); !strings.Contains(got, `"internal/abi"`) || !strings.Contains(got, "func clone") {
				t.Fatalf("overlay[%q] does not contain the clone replacement:\n%s", clonePatch, got)
			}
			if tc.wantFiles == 2 {
				handlePatch := filepath.Join(runtime.GOROOT(), "src", "unique", "z_llgo_patch_handle.go")
				if got := string(overlay[handlePatch]); !strings.Contains(got, "type Handle") {
					t.Fatalf("overlay[%q] does not contain the Handle replacement:\n%s", handlePatch, got)
				}
			}
		})
	}
}

func TestRuntimeMapsTypeStringPatchMatchesGo125(t *testing.T) {
	const pkgPath = "internal/runtime/maps"
	patchFile := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkgPath), "z_llgo_patch_typestring_go125.go")
	for _, version := range []string{"go1.24.0", "go1.25.0", "go1.26.0"} {
		t.Run(version, func(t *testing.T) {
			_, overlay, _, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), pkgPath, sourcePatchBuildContext{
				goos:      runtime.GOOS,
				goarch:    runtime.GOARCH,
				goversion: version,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, got := overlay[patchFile]
			if want := version == "go1.25.0"; got != want {
				t.Fatalf("Go 1.25 typeString patch present = %v, want %v", got, want)
			}
			if version == "go1.25.0" && !strings.Contains(string(overlay[patchFile]), "if typ == nil") {
				t.Fatal("Go 1.25 typeString patch should preserve the nil guard")
			}
			if version == "go1.26.0" {
				mapKeyPatch := filepath.Join(runtime.GOROOT(), "src", filepath.FromSlash(pkgPath), "z_llgo_patch_mapkey_go126.go")
				if !strings.Contains(string(overlay[mapKeyPatch]), "if typ == nil") {
					t.Fatal("Go 1.26 typeString patch should preserve the nil guard")
				}
			}
		})
	}
}

func TestCompilePkgSFilesSkipsSourcePatchedAssembly(t *testing.T) {
	got, err := compilePkgSFiles(
		&context{
			buildConf:  &Config{Goarch: "wasm"},
			patchFiles: map[string][]string{"internal/bytealg": {"bytealg_wasm.go"}},
		},
		nil,
		&packages.Package{PkgPath: "internal/bytealg"},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("compilePkgSFiles returned %v, want no object files", got)
	}
}

func TestSourcePatchAssemblyMatchError(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	const pkgPath = "internal/bytealg"
	srcDir := filepath.Join(goroot, "src", filepath.FromSlash(pkgPath))
	patchDir := filepath.Join(runtimeDir, "_patch", filepath.FromSlash(pkgPath))

	if err := os.MkdirAll(filepath.Join(srcDir, "adir"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(srcDir, "bad_wasm.s"), "//go:build (\n")
	mustWriteFile(t, filepath.Join(patchDir, "bytealg_wasm.go"), `//go:build wasm

package bytealg
`)

	_, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:   "js",
		goarch: "wasm",
	})
	if err == nil || !strings.Contains(err.Error(), "match stdlib assembly file") {
		t.Fatalf("applySourcePatchForPkg error = %v, want assembly match error", err)
	}
}

func TestSourcePatchSourceMatchError(t *testing.T) {
	for _, directive := range []string{"//llgo:skipall", "//llgo:skip Target"} {
		t.Run(directive, func(t *testing.T) {
			goroot := t.TempDir()
			runtimeDir := t.TempDir()
			const pkgPath = "demo"
			srcDir := filepath.Join(goroot, "src", pkgPath)
			patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
			mustWriteFile(t, filepath.Join(srcDir, "bad_linux.go"), "//go:build (\n")
			mustWriteFile(t, filepath.Join(patchDir, "patch.go"), "package demo\n\n"+directive+"\nfunc Target() {}\n")

			_, _, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
				goos:   "linux",
				goarch: "amd64",
			})
			if err == nil || !strings.Contains(err.Error(), "match stdlib source file") {
				t.Fatalf("applySourcePatchForPkg error = %v, want source match error", err)
			}
		})
	}
}

func TestBuildSourcePatchOverlayForIter(t *testing.T) {
	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}

	iterDir := filepath.Join(runtime.GOROOT(), "src", "iter")
	patchFile := filepath.Join(iterDir, "z_llgo_patch_iter.go")
	patchSrc, ok := overlay[patchFile]
	if !ok {
		t.Fatalf("missing source patch file %s", patchFile)
	}
	if !strings.Contains(string(patchSrc), "func Pull[V any]") {
		t.Fatalf("source patch file %s does not contain iter replacement", patchFile)
	}
	if !strings.HasPrefix(string(patchSrc), sourcePatchLineDirective(filepath.Join(env.LLGoRuntimeDir(), "_patch", "iter", "iter.go"))) {
		t.Fatalf("source patch file %s is missing line directive, got:\n%s", patchFile, patchSrc)
	}

	stdFile := filepath.Join(iterDir, "iter.go")
	stdSrc, ok := overlay[stdFile]
	if !ok {
		t.Fatalf("missing stub overlay for %s", stdFile)
	}
	got := string(stdSrc)
	if !strings.Contains(got, "package iter") {
		t.Fatalf("stub overlay for %s lost package clause", stdFile)
	}
	if strings.Contains(got, "func Pull") {
		t.Fatalf("stub overlay for %s still contains original declarations", stdFile)
	}
}

func TestIterUsesSourcePatchInsteadOfAltPkg(t *testing.T) {
	if !llruntime.HasSourcePatchPkg("iter") {
		t.Fatal("iter should be registered as a source patch package")
	}
	if llruntime.HasAltPkg("iter") {
		t.Fatal("iter should not remain an alt package")
	}
}

func TestBuildSourcePatchOverlayForGo126Payloads(t *testing.T) {
	goroot := t.TempDir()
	mustWriteFile(t, filepath.Join(goroot, "src", "internal", "sync", "hashtriemap.go"), `package sync

type HashTrieMap[K comparable, V any] struct{}
`)
	mustWriteFile(t, filepath.Join(goroot, "src", "internal", "sync", "mutex.go"), `package sync

type Mutex struct{}
`)
	mustWriteFile(t, filepath.Join(goroot, "src", "crypto", "internal", "constanttime", "constant_time.go"), `package constanttime

func boolToUint8(bool) uint8
`)

	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.26.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	syncDir := filepath.Join(goroot, "src", "internal", "sync")
	syncPatch := filepath.Join(syncDir, "z_llgo_patch_hashtriemap.go")
	if src, ok := overlay[syncPatch]; !ok {
		t.Fatalf("missing source patch file %s", syncPatch)
	} else if !strings.Contains(string(src), "type HashTrieMap") {
		t.Fatalf("source patch file %s does not contain HashTrieMap replacement", syncPatch)
	}
	if stdSrc := string(overlay[filepath.Join(syncDir, "hashtriemap.go")]); strings.Contains(stdSrc, "type HashTrieMap") {
		t.Fatalf("stub overlay for internal/sync still contains HashTrieMap: %s", stdSrc)
	}

	constanttimeDir := filepath.Join(goroot, "src", "crypto", "internal", "constanttime")
	constanttimePatch := filepath.Join(constanttimeDir, "z_llgo_patch_constant_time.go")
	if src, ok := overlay[constanttimePatch]; !ok {
		t.Fatalf("missing source patch file %s", constanttimePatch)
	} else if !strings.Contains(string(src), "//go:linkname boolToUint8 llgo.boolToUint8") {
		t.Fatalf("source patch file %s does not contain boolToUint8 linkname", constanttimePatch)
	}
}

func TestGo126PayloadsUseSourcePatchInsteadOfAltPkg(t *testing.T) {
	for _, pkgPath := range []string{"internal/sync", "crypto/internal/constanttime"} {
		if !llruntime.HasSourcePatchPkg(pkgPath) {
			t.Fatalf("%s should be registered as a source patch package", pkgPath)
		}
		if llruntime.HasAltPkg(pkgPath) {
			t.Fatalf("%s should not remain an alt package", pkgPath)
		}
	}
}

func TestRuntimeHooksUseSourcePatchesInsteadOfAltPkgs(t *testing.T) {
	for _, pkgPath := range []string{"internal/runtime/maps", "internal/runtime/sys", "sync/atomic", "unique"} {
		if !llruntime.HasSourcePatchPkg(pkgPath) {
			t.Fatalf("%s should be registered as a source patch package", pkgPath)
		}
		if llruntime.HasAltPkg(pkgPath) {
			t.Fatalf("%s should not remain an alt package", pkgPath)
		}
	}
}

func TestApplySourcePatchForPkg_Cases(t *testing.T) {
	for _, caseName := range []string{
		"default-override",
		"generic-constraints-and-interface",
		"generic-type-and-method",
		"multi-file-skipall",
		"multi-file-with-asm",
		"skip-and-override",
		"skipall",
		"type-alias-and-grouped-values",
	} {
		t.Run(caseName, func(t *testing.T) {
			runSourcePatchCase(t, caseName)
		})
	}
}

func TestApplySourcePatchForPkg_MissingStdlibPkg(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "iter"
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(patchDir, "iter.go"), `package iter

//llgo:skipall

func Pull[V any](seq func(func(V) bool)) {}
`)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected missing stdlib package to skip source patching")
	}
	if overlay != nil {
		t.Fatalf("expected no overlay for missing stdlib package, got %v entries", len(overlay))
	}
}

func TestApplySourcePatchForPkg_BuildTaggedPatch(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(srcDir, "demo.go"), `package demo

func Old() string { return "old" }
`)
	mustWriteFile(t, filepath.Join(patchDir, "patch.go"), `//go:build go1.26
//llgo:skipall
package demo

const Only = "patched"
`)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.24.11",
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("expected go1.26-tagged patch to be ignored on go1.24, got overlay: %#v", overlay)
	}

	changed, overlay, _, err = applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.26.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected go1.26-tagged patch to apply on go1.26")
	}
}

func TestApplySourcePatchForPkg_FiltersOnlyTargetFiles(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(srcDir, "demo_linux.go"), `package demo

func Target() string { return "linux" }
`)
	mustWriteFile(t, filepath.Join(srcDir, "demo_windows.go"), `package demo

func Target() string { return "windows" }
`)
	mustWriteFile(t, filepath.Join(srcDir, "demo_experiment.go"), `//go:build goexperiment.future

package demo

func Experiment() string { return "experiment" }
`)
	mustWriteFile(t, filepath.Join(patchDir, "patch.go"), `package demo

//llgo:skip Target Experiment
func Target() string { return "patched" }
func Experiment() string { return "patched" }
`)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:   "linux",
		goarch: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected source patch overlay to change package")
	}
	if _, ok := overlay[filepath.Join(srcDir, "demo_linux.go")]; !ok {
		t.Fatal("active target source was not filtered")
	}
	if _, ok := overlay[filepath.Join(srcDir, "demo_experiment.go")]; !ok {
		t.Fatal("target source with a different tool tag was not filtered")
	}
	if _, ok := overlay[filepath.Join(srcDir, "demo_windows.go")]; ok {
		t.Fatal("source for a different target should not be parsed or overlaid")
	}
}

func TestApplySourcePatchForPkg_SkipAllFiltersTargetFiles(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	const pkgPath = "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(srcDir, "demo_linux.go"), "package demo\n")
	mustWriteFile(t, filepath.Join(srcDir, "demo_windows.go"), "package demo\n")
	mustWriteFile(t, filepath.Join(patchDir, "patch.go"), `package demo

//llgo:skipall
`)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:   "linux",
		goarch: "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected source patch overlay to change package")
	}
	if _, ok := overlay[filepath.Join(srcDir, "demo_linux.go")]; !ok {
		t.Fatal("active target source was not stubbed")
	}
	if _, ok := overlay[filepath.Join(srcDir, "demo_windows.go")]; ok {
		t.Fatal("source for a different target should not be stubbed")
	}
}

func TestSourcePatchMayContainSkip(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		skips []string
		want  bool
	}{
		{"function", "package p\nfunc Target() {}\n", []string{"Target"}, true},
		{"qualified function", "package p\nfunc Target() {}\n", []string{"p.Target"}, true},
		{"method", "package p\nfunc (*T) Method() {}\n", []string{"(*T).Method"}, true},
		{"absent", "package p\nfunc Other() {}\n", []string{"Target"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			skips := make(map[string]struct{}, len(test.skips))
			for _, name := range test.skips {
				skips[name] = struct{}{}
			}
			if got := sourcePatchMayContainSkip([]byte(test.src), skips); got != test.want {
				t.Fatalf("sourcePatchMayContainSkip() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestApplySourcePatchForPkg_UnreadableStdlibPkg(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test is Unix-only")
	}
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "iter"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(srcDir, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(srcDir, 0755)
	mustWriteFile(t, filepath.Join(patchDir, "iter.go"), `package iter

//llgo:skipall

func Pull[V any](seq func(func(V) bool)) {}
`)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unreadable stdlib package to skip source patching")
	}
	if overlay != nil {
		t.Fatalf("expected no overlay for unreadable stdlib package, got %v entries", len(overlay))
	}
}

func runSourcePatchCase(t *testing.T, caseName string) {
	t.Helper()

	assetRoot := filepath.Join(env.LLGoRuntimeDir(), "_patch", "_test", caseName)
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	const pkgPath = "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)

	copyTree(t, filepath.Join(assetRoot, "pkg"), srcDir)
	copyTree(t, filepath.Join(assetRoot, "patch"), patchDir)

	changed, overlay, _, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected source patch overlay to change package")
	}

	assertOverlayMatchesOutput(t, overlay, srcDir, filepath.Join(assetRoot, "output"), runtimeDir)
	assertGeneratedPatchPositions(t, overlay, srcDir, patchDir)
}

func copyTree(t *testing.T, srcRoot, dstRoot string) {
	t.Helper()
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertOverlayMatchesOutput(t *testing.T, overlay map[string][]byte, srcRoot, outputRoot, runtimeDir string) {
	t.Helper()

	got := overlayFilesUnderRoot(t, overlay, srcRoot)
	want := readTextFiles(t, outputRoot, runtimeDir)

	gotNames := sortedMapKeys(got)
	wantNames := sortedMapKeys(want)
	assertExactString(t, "overlay file list", strings.Join(gotNames, "\n"), strings.Join(wantNames, "\n"))

	for _, name := range wantNames {
		assertExactString(t, "overlay file "+name, got[name], want[name])
	}
}

func overlayFilesUnderRoot(t *testing.T, overlay map[string][]byte, root string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for filename, src := range overlay {
		rel, err := filepath.Rel(root, filename)
		if err != nil {
			t.Fatal(err)
		}
		if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
			continue
		}
		out[filepath.ToSlash(rel)] = string(src)
	}
	return out
}

func readTextFiles(t *testing.T, root, runtimeDir string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if strings.HasSuffix(key, ".txt") {
			key = strings.TrimSuffix(key, ".txt")
		}
		out[key] = expandSourcePatchOutputTemplate(string(data), runtimeDir)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func expandSourcePatchOutputTemplate(src, runtimeDir string) string {
	patchRoot := filepath.ToSlash(filepath.Join(runtimeDir, "_patch"))
	return strings.ReplaceAll(src, "{{PATCH_ROOT}}", patchRoot)
}

func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertGeneratedPatchPositions(t *testing.T, overlay map[string][]byte, srcRoot, patchRoot string) {
	t.Helper()
	for rel, src := range overlayFilesUnderRoot(t, overlay, srcRoot) {
		base := filepath.Base(rel)
		if !strings.HasPrefix(base, "z_llgo_patch_") {
			continue
		}
		original := strings.TrimPrefix(base, "z_llgo_patch_")
		patchFile := filepath.Join(patchRoot, filepath.Dir(rel), original)
		for _, target := range patchedTargetsOfFile(t, patchFile) {
			assertPatchedPosition(t, src, filepath.Join(srcRoot, filepath.FromSlash(rel)), patchFile, target.key, target.line)
		}
	}
}

type patchedTarget struct {
	key  string
	line int
}

func patchedTargetsOfFile(t *testing.T, filename string) []patchedTarget {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	targets := []patchedTarget{{
		key:  "package",
		line: fset.Position(file.Package).Line,
	}}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			key := "func:" + decl.Name.Name
			if decl.Recv != nil && len(decl.Recv.List) != 0 {
				key = "method:" + recvPatchKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
			}
			targets = append(targets, patchedTarget{
				key:  key,
				line: fset.Position(decl.Name.Pos()).Line,
			})
		case *ast.GenDecl:
			kind := strings.ToLower(decl.Tok.String())
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					targets = append(targets, patchedTarget{
						key:  "type:" + spec.Name.Name,
						line: fset.Position(spec.Name.Pos()).Line,
					})
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						targets = append(targets, patchedTarget{
							key:  kind + ":" + name.Name,
							line: fset.Position(name.Pos()).Line,
						})
					}
				}
			}
		}
	}
	return targets
}

func mustWriteFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sourcePatchLineDirective(filename string) string {
	return "//line " + filepath.ToSlash(filename) + ":1\n"
}

func assertExactString(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch\nwant:\n%q\n\ngot:\n%q", label, want, got)
	}
}

func assertPatchedPosition(t *testing.T, src, generatedFilename, wantFilename, target string, wantLine int) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, generatedFilename, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pos, ok := findPatchedPosition(file, target)
	if !ok {
		t.Fatalf("target %q not found", target)
	}
	got := fset.Position(pos)
	if filepath.ToSlash(got.Filename) != filepath.ToSlash(wantFilename) || got.Line != wantLine {
		t.Fatalf("target %q position mismatch: want %s:%d, got %s:%d", target, filepath.ToSlash(wantFilename), wantLine, filepath.ToSlash(got.Filename), got.Line)
	}
}

func findPatchedPosition(file *ast.File, target string) (token.Pos, bool) {
	if target == "package" {
		return file.Package, true
	}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			key := "func:" + decl.Name.Name
			if decl.Recv != nil && len(decl.Recv.List) != 0 {
				key = "method:" + recvPatchKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
			}
			if key == target {
				return decl.Name.Pos(), true
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if "type:"+spec.Name.Name == target {
						return spec.Name.Pos(), true
					}
				case *ast.ValueSpec:
					kind := strings.ToLower(decl.Tok.String())
					for _, name := range spec.Names {
						if kind+":"+name.Name == target {
							return name.Pos(), true
						}
					}
				}
			}
		}
	}
	return token.NoPos, false
}

//go:build !llgo
// +build !llgo

package crosscompile

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile/compile"
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/xtool/llvm"
)

const (
	sysrootPrefix     = "--sysroot="
	resourceDirPrefix = "-resource-dir="
	includePrefix     = "-I"
	libPrefix         = "-L"
)

func TestESPClangHostDownload(t *testing.T) {
	tests := []struct {
		goos, goarch string
		wantPlatform string
	}{
		{"darwin", "arm64", "aarch64-apple-darwin"},
		{"darwin", "amd64", "x86_64-apple-darwin"},
		{"linux", "arm64", "aarch64-linux-gnu"},
		{"linux", "amd64", "x86_64-linux-gnu"},
		{"windows", "386", "x86_64-w64-mingw32"},
		{"windows", "amd64", "x86_64-w64-mingw32"},
		{"windows", "arm64", "aarch64-w64-mingw32"},
	}
	for _, test := range tests {
		platform := getESPClangPlatform(test.goos, test.goarch)
		if platform != test.wantPlatform {
			t.Errorf("getESPClangPlatform(%q, %q) = %q, want %q", test.goos, test.goarch, platform, test.wantPlatform)
			continue
		}
		if espClangSHA256[platform] == "" {
			t.Errorf("ESP Clang %s %s has no checksum", espClangVersion, platform)
		}
	}
}

func TestESPClangCacheSeparatesHosts(t *testing.T) {
	legacy := filepath.Join(cacheDir(), "esp-clang-"+espClangVersion)
	x64 := espClangCacheDir(getESPClangPlatform("windows", "amd64"))
	arm64 := espClangCacheDir(getESPClangPlatform("windows", "arm64"))
	if x64 == arm64 || x64 == legacy || arm64 == legacy {
		t.Fatalf("ESP payload caches overlap: x64=%q arm64=%q legacy=%q", x64, arm64, legacy)
	}
}

func TestESPClangDoesNotReuseLegacyCache(t *testing.T) {
	writeWasmTargetFixture(t, "", "")
	originalCacheRoot, originalBaseURL := cacheRoot, espClangBaseUrl
	root := t.TempDir()
	cacheRoot = func() string { return root }
	server := httptest.NewServer(http.NotFoundHandler())
	espClangBaseUrl = server.URL
	t.Cleanup(func() {
		cacheRoot, espClangBaseUrl = originalCacheRoot, originalBaseURL
		server.Close()
	})
	legacy := filepath.Join(cacheDir(), "esp-clang-"+espClangVersion)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := getESPClangRoot(true)
	if got != "" || err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("getESPClangRoot() = %q, %v; want a fresh download instead of legacy cache %q", got, err, legacy)
	}
}

func TestESPClangRejectsMissingChecksum(t *testing.T) {
	writeWasmTargetFixture(t, "", "")
	checksums := espClangSHA256
	espClangSHA256 = nil
	t.Cleanup(func() { espClangSHA256 = checksums })

	root, err := getESPClangRoot(true)
	platform := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	if platform == "" {
		if err == nil || !strings.Contains(err.Error(), "is not supported for download") {
			t.Fatalf("unsupported host error = %v", err)
		}
		return
	}
	if root != "" || err == nil || err.Error() != "missing ESP Clang checksum for "+platform {
		t.Fatalf("getESPClangRoot() = %q, %v; want missing checksum error", root, err)
	}
}

func TestCompileWithConfigRejectsFileAsOutputDir(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	if err := os.WriteFile(output, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := compileWithConfig(compile.CompileConfig{}, output, compile.CompileOptions{}); err == nil || !strings.Contains(err.Error(), "create compiled library cache") {
		t.Fatalf("compileWithConfig error = %v", err)
	}
}

func TestUseCrossCompileSDK(t *testing.T) {
	// Skip long-running tests unless explicitly enabled
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Test cases
	testCases := []struct {
		name          string
		goos          string
		goarch        string
		expectSDK     bool
		expectCCFlags bool
		expectCFlags  bool
		expectLDFlags bool
	}{
		{
			name:          "Same Platform",
			goos:          runtime.GOOS,
			goarch:        runtime.GOARCH,
			expectSDK:     true,  // We expect flags even for same platform
			expectCCFlags: true,  // CCFLAGS will contain sysroot
			expectCFlags:  false, // CFLAGS will not contain include paths
			expectLDFlags: false, // LDFLAGS will not contain library paths
		},
		{
			name:          "WASM Target",
			goos:          "wasip1",
			goarch:        "wasm",
			expectSDK:     true,
			expectCCFlags: true,
			expectCFlags:  true,
			expectLDFlags: true,
		},
		{
			name:          "Unsupported Target",
			goos:          "windows",
			goarch:        "amd64",
			expectSDK:     false, // Still false as it won't set up specific SDK
			expectCCFlags: false, // No cross-compile specific flags
			expectCFlags:  false, // No cross-compile specific flags
			expectLDFlags: false, // No cross-compile specific flags
		},
	}

	// Create a temporary directory for the cache
	tempDir, err := os.MkdirTemp("", "crosscompile_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalCacheRoot := cacheRoot
	cacheRoot = func() string { return tempDir }
	defer func() { cacheRoot = originalCacheRoot }()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			export, err := use(tc.goos, tc.goarch, false, false, optlevel.O2, lto.Off, false)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			t.Logf("export: %+v", export)

			if tc.expectSDK {
				// Check if flags are set correctly
				if tc.expectCCFlags && len(export.CCFLAGS) == 0 {
					t.Error("Expected CCFLAGS to be set, but they are empty")
				}

				if tc.expectCFlags && len(export.CFLAGS) == 0 {
					t.Error("Expected CFLAGS to be set, but they are empty")
				}

				if tc.expectLDFlags && len(export.LDFLAGS) == 0 {
					t.Error("Expected LDFLAGS to be set, but they are empty")
				}

				// Check for specific flags
				if tc.expectCCFlags {
					hasSysroot := false
					hasResourceDir := false

					for _, flag := range export.CCFLAGS {
						if len(flag) >= len(sysrootPrefix) && flag[:len(sysrootPrefix)] == sysrootPrefix {
							hasSysroot = true
						}
						if len(flag) >= len(resourceDirPrefix) && flag[:len(resourceDirPrefix)] == resourceDirPrefix {
							hasResourceDir = true
						}
					}

					// For WASM target, both sysroot and resource-dir are expected
					if tc.name == "WASM Target" {
						if !hasSysroot {
							t.Error("Missing --sysroot flag in CCFLAGS")
						}
						if !hasResourceDir {
							t.Error("Missing -resource-dir flag in CCFLAGS")
						}
						if !slices.Contains(export.CCFLAGS, "-fwasm-exceptions") ||
							!hasMllvmOption(export.CCFLAGS, "-wasm-enable-sjlj") {
							t.Errorf("CCFLAGS do not enable WebAssembly SjLj lowering: %v", export.CCFLAGS)
						}
						if !export.WasmPostLink.Asyncify {
							t.Error("WASI target does not request Asyncify post-link processing")
						}
						if slices.Contains(export.LDFLAGS, "-Wl,--import-memory") {
							t.Errorf("single-worker WASI imports host memory: %v", export.LDFLAGS)
						}
						if !slices.Contains(export.LDFLAGS, "-Wl,--stack-first") {
							t.Errorf("single-worker WASI does not fix the Asyncify stack layout: %v", export.LDFLAGS)
						}
					} else if tc.name == "Same Platform" {
						// For same platform, we expect sysroot only on macOS
						if runtime.GOOS == "darwin" && !hasSysroot {
							t.Error("Missing --sysroot flag in CCFLAGS on macOS")
						}
						// On Linux and other platforms, sysroot is not necessarily required
					}
				}

				if tc.expectCFlags {
					hasInclude := false

					for _, flag := range export.CFLAGS {
						if len(flag) >= len(includePrefix) && flag[:len(includePrefix)] == includePrefix {
							hasInclude = true
						}
					}

					if !hasInclude {
						t.Error("Missing -I flag in CFLAGS")
					}
				}

				if tc.expectLDFlags {
					hasLib := false

					for _, flag := range export.LDFLAGS {
						if len(flag) >= len(libPrefix) && flag[:len(libPrefix)] == libPrefix {
							hasLib = true
						}
					}

					if !hasLib {
						t.Error("Missing -L flag in LDFLAGS")
					}
				}
			} else {
				// For unsupported targets, we still expect some basic flags to be set
				// since the implementation now always sets up ESP Clang environment
				// Only check that we don't have specific SDK-related flags for unsupported targets
				if tc.name == "Unsupported Target" && len(export.CFLAGS) != 0 {
					t.Errorf("Expected empty CFLAGS for unsupported target, got CFLAGS=%v", export.CFLAGS)
				}
			}
		})
	}
}

func TestUseWASIThreadsImportsMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the external WASI SDK link test in short mode")
	}
	export, err := use("wasip1", "wasm", true, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(export.CCFLAGS, "-pthread") {
		t.Fatalf("CCFLAGS do not enable WASI threads: %v", export.CCFLAGS)
	}
	if !slices.Contains(export.BuildTags, "llgo.wasi_threads") {
		t.Fatalf("BuildTags do not select the WASI pthread backend: %v", export.BuildTags)
	}
	if !slices.Contains(export.LDFLAGS, "-Wl,--import-memory") {
		t.Fatalf("LDFLAGS do not import shared host memory: %v", export.LDFLAGS)
	}
	if export.WasmPostLink.Asyncify {
		t.Fatal("WASI pthread mode requests single-worker Asyncify processing")
	}
}

func TestUseWASILTOEnablesSjLjAtLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the external WASI SDK link test in short mode")
	}
	export, err := use("wasip1", "wasm", false, false, optlevel.O2, lto.Thin, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(export.LDFLAGS, "-Wl,--mllvm=-wasm-enable-sjlj") {
		t.Fatalf("LDFLAGS do not enable Wasm SjLj for LTO: %v", export.LDFLAGS)
	}
}

func TestUseTarget(t *testing.T) {
	// Test cases for target-based configuration
	testCases := []struct {
		name        string
		targetName  string
		expectError bool
		expectLLVM  string
		expectCPU   string
		expectMarch string
	}{
		// FIXME(MeteorsLiu): wasi in useTarget
		// {
		// 	name:        "WASI Target",
		// 	targetName:  "wasi",
		// 	expectError: false,
		// 	expectLLVM:  "",
		// 	expectCPU:   "generic",
		// },
		{
			name:        "RP2040 Target",
			targetName:  "rp2040",
			expectError: false,
			expectLLVM:  "thumbv6m-unknown-unknown-eabi",
			expectCPU:   "cortex-m0plus",
		},
		{
			name:        "Cortex-M Target",
			targetName:  "cortex-m",
			expectError: true,
			expectLLVM:  "",
			expectCPU:   "",
		},
		{
			name:        "Arduino Target (with filtered flags)",
			targetName:  "arduino",
			expectError: false,
			expectLLVM:  "avr",
			expectCPU:   "atmega328p",
		},
		{
			name:        "RISC-V32 Target (generic)",
			targetName:  "riscv32",
			expectError: false,
			expectLLVM:  "riscv32-unknown-none",
			expectCPU:   "generic-rv32",
			expectMarch: "-march=rv32imac", // Generic RISC-V32 uses rv32imac (with A extension)
		},
		{
			name:        "ESP32 Target (Xtensa)",
			targetName:  "esp32",
			expectError: false,
			expectLLVM:  "xtensa",
			expectCPU:   "esp32",
		},
		{
			name:        "ESP32-C3 Target (ESP RISC-V)",
			targetName:  "esp32c3",
			expectError: false,
			expectLLVM:  "riscv32-esp-elf",
			expectCPU:   "generic-rv32",
			expectMarch: "-march=rv32imc", // ESP32-C3 uses rv32imc (no A extension)
		},
		{
			name:        "Nonexistent Target",
			targetName:  "nonexistent-target",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			export, err := UseTarget(tc.targetName, optlevel.Oz, lto.Thin)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for target %s, but got none", tc.targetName)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for target %s: %v", tc.targetName, err)
			}
			if !export.DebugInfo.AlwaysOmit {
				t.Fatalf("target %s debug-info policy = %+v, want AlwaysOmit", tc.targetName, export.DebugInfo)
			}
			if !slices.Contains(export.LDFLAGS, "-S") {
				t.Fatalf("target %s declares AlwaysOmit without linker -S: %v", tc.targetName, export.LDFLAGS)
			}
			// Check if LLVM target is in CCFLAGS
			if tc.expectLLVM != "" {
				found := false
				expectedFlag := "--target=" + tc.expectLLVM
				for _, flag := range export.CCFLAGS {
					if flag == expectedFlag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected LLVM target %s in CCFLAGS, got %v", expectedFlag, export.CCFLAGS)
				}
			}

			// Check if CPU is in LDFLAGS (for ld.lld linker) or CCFLAGS (for other cases)
			if tc.expectCPU != "" {
				found := false
				// First check LDFLAGS for -mllvm -mcpu= pattern
				for i, flag := range export.LDFLAGS {
					if flag == "-mllvm" && i+1 < len(export.LDFLAGS) {
						nextFlag := export.LDFLAGS[i+1]
						if nextFlag == "-mcpu="+tc.expectCPU {
							found = true
							break
						}
					}
				}
				// If not found in LDFLAGS, check CCFLAGS for direct CPU flags
				if !found {
					expectedFlags := []string{"-mmcu=" + tc.expectCPU, "-mcpu=" + tc.expectCPU}
					for _, flag := range export.CCFLAGS {
						for _, expectedFlag := range expectedFlags {
							if flag == expectedFlag {
								found = true
								break
							}
						}
					}
				}
				if !found {
					t.Errorf("Expected CPU %s in LDFLAGS or CCFLAGS, got LDFLAGS=%v, CCFLAGS=%v", tc.expectCPU, export.LDFLAGS, export.CCFLAGS)
				}
			}

			// Check if -march flag is correct
			if tc.expectMarch != "" {
				found := false
				for _, flag := range export.CCFLAGS {
					if flag == tc.expectMarch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected %s in CCFLAGS, got %v", tc.expectMarch, export.CCFLAGS)
				}
			}
			t.Logf("Target %s: BuildTags=%v, CFlags=%v, CCFlags=%v, LDFlags=%v",
				tc.targetName, export.BuildTags, export.CFLAGS, export.CCFLAGS, export.LDFLAGS)
		})
	}
}

func TestEmscriptenTargetProfiles(t *testing.T) {
	tests := []struct {
		name       string
		wantABI    WasmABI
		wantTriple string
		wantTags   []string
		memory64   bool
	}{
		{"emscripten", WasmABIEmscripten, "wasm32-unknown-emscripten", []string{"llgo.wasm.emscripten"}, false},
		{"wasm", WasmABIEmscripten, "wasm32-unknown-emscripten", []string{"llgo.wasm.emscripten", "tinygo.wasm"}, false},
		{"emscripten-memory64", WasmABIEmscriptenMemory64, "wasm64-unknown-emscripten", []string{"llgo.wasm.emscripten", "llgo.wasm.emscripten.memory64"}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			export, err := Use("", "", test.name, false, false, optlevel.O2, lto.Off, false)
			if err != nil {
				t.Fatal(err)
			}
			if export.WasmABI != test.wantABI || export.LLVMTarget != test.wantTriple ||
				export.GOOS != "js" || export.GOARCH != "wasm" || export.CC != "emcc" {
				t.Fatalf("export = ABI %q, LLVM %q, %s/%s, CC %q",
					export.WasmABI, export.LLVMTarget, export.GOOS, export.GOARCH, export.CC)
			}
			for _, tag := range test.wantTags {
				if !slices.Contains(export.BuildTags, tag) {
					t.Errorf("build tags %v do not contain %q", export.BuildTags, tag)
				}
			}
			ccMemory64 := slices.Contains(export.CCFLAGS, "wasm64-unknown-emscripten")
			ldMemory64 := slices.Contains(export.LDFLAGS, "wasm64-unknown-emscripten")
			if got := ccMemory64 && ldMemory64; got != test.memory64 {
				t.Errorf("wasm64 compile/link target present = %v, want %v; CCFLAGS=%v LDFLAGS=%v", got, test.memory64, export.CCFLAGS, export.LDFLAGS)
			}
			if !slices.Contains(export.CCFLAGS, "-O2") || !slices.Contains(export.LDFLAGS, "-O2") {
				t.Errorf("Emscripten optimization level is not consistent across compile/link: CCFLAGS=%v LDFLAGS=%v", export.CCFLAGS, export.LDFLAGS)
			}
			if !slices.Contains(export.LDFLAGS, "-sENVIRONMENT=web,worker,node") {
				t.Errorf("named target does not enable its Node emulator: %v", export.LDFLAGS)
			}
			if !slices.Contains(export.LDFLAGS, emscriptenAsyncifyImports) {
				t.Errorf("named target does not mark interruptible host wait and ffi_call_js as async: %v", export.LDFLAGS)
			}
			if !slices.Contains(export.LDFLAGS, "-sEXIT_RUNTIME=1") {
				t.Errorf("named target does not let fatal Asyncify programs exit: %v", export.LDFLAGS)
			}
			wantRunner := "emscripten-runner.mjs"
			if test.memory64 {
				wantRunner = "emscripten-memory64-runner.mjs"
			}
			if strings.Contains(export.Emulator, "{root}") || !strings.Contains(export.Emulator, wantRunner) ||
				!strings.Contains(export.Emulator, "{}") {
				t.Errorf("named target emulator was not resolved: %q", export.Emulator)
			}
		})
	}
}

func TestEmscriptenAsyncifyLinkOptimization(t *testing.T) {
	for _, level := range []optlevel.Level{optlevel.O3, optlevel.Os, optlevel.Oz} {
		t.Run(level.Name(), func(t *testing.T) {
			export, err := Use("", "", "emscripten", false, false, level, lto.Off, false)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(export.CCFLAGS, level.Flag()) {
				t.Fatalf("compiler flags %v do not preserve requested %s", export.CCFLAGS, level)
			}
			if !slices.Contains(export.LDFLAGS, "-O2") || slices.Contains(export.LDFLAGS, level.Flag()) {
				t.Fatalf("link flags %v do not avoid Emscripten MetaDCE for %s", export.LDFLAGS, level)
			}
			if slices.Contains(export.LDFLAGS, "-sASSERTIONS=1") {
				t.Fatalf("link flags use assertions to disable MetaDCE: %v", export.LDFLAGS)
			}
		})
	}
}

func TestWASIProfileTargets(t *testing.T) {
	for _, test := range []struct {
		name     string
		wantTags []string
	}{
		{"wasi", []string{"llgo.wasm.wasi"}},
		{"wasip1", []string{"llgo.wasm.wasi", "tinygo.wasm"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			export, err := Use("", "", test.name, false, false, optlevel.O2, lto.Off, false)
			if err != nil {
				t.Fatal(err)
			}
			if export.WasmABI != WasmABIWASIPreview1 || export.LLVMTarget != "wasm32-unknown-wasip1" ||
				export.GOOS != "wasip1" || export.GOARCH != "wasm" {
				t.Fatalf("export = ABI %q, LLVM %q, %s/%s", export.WasmABI, export.LLVMTarget, export.GOOS, export.GOARCH)
			}
			for _, tag := range test.wantTags {
				if !slices.Contains(export.BuildTags, tag) {
					t.Errorf("build tags %v do not contain %q", export.BuildTags, tag)
				}
			}
			if slices.Contains(export.LDFLAGS, "-Wl,--import-memory,") || slices.Contains(export.LDFLAGS, "-Wl,--import-memory") {
				t.Fatalf("single-worker WASI unexpectedly imports host memory: %v", export.LDFLAGS)
			}
			if !export.WasmPostLink.Asyncify {
				t.Fatal("single-worker WASI does not request Asyncify post-link processing")
			}
		})
	}
}

func TestAppendEmscriptenLibffiSearchPath(t *testing.T) {
	var export Export
	appendEmscriptenLibffiSearchPath(&export, "", WasmABIEmscripten)
	if len(export.LDFLAGS) != 0 {
		t.Fatalf("empty LLGO_ROOT appended %v", export.LDFLAGS)
	}
	appendEmscriptenLibffiSearchPath(&export, "/llgo", WasmABIEmscriptenMemory64)
	want64 := "-L" + filepath.Join("/llgo", wasm64LibffiRelDir)
	if !slices.Contains(export.LDFLAGS, want64) {
		t.Fatalf("memory64 LDFLAGS %v do not search %s", export.LDFLAGS, want64)
	}
	appendEmscriptenLibffiSearchPath(&export, "/llgo", WasmABIUnspecified)
	want32 := "-L" + filepath.Join("/llgo", wasm32LibffiRelDir)
	if !slices.Contains(export.LDFLAGS, want32) {
		t.Fatalf("unspecified ABI LDFLAGS %v do not search %s", export.LDFLAGS, want32)
	}
	appendEmscriptenLibffiSearchPath(&export, "/llgo", WasmABIEmscripten)
	n := 0
	for _, flag := range export.LDFLAGS {
		if flag == want32 {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("emscripten search path count = %d, want 2 in %v", n, export.LDFLAGS)
	}
}

func TestEmscriptenLibffiSearchPath(t *testing.T) {
	root := env.LLGoROOT()
	if root == "" {
		t.Fatal("LLGO_ROOT is required to locate the vendored wasm32 libffi archive")
	}
	libDir := filepath.Join(root, wasm32LibffiRelDir)
	if _, err := os.Stat(filepath.Join(libDir, "libffi.a")); err != nil {
		t.Fatalf("vendored wasm32 libffi archive: %v", err)
	}
	wantL := "-L" + libDir

	js, err := use("js", "wasm", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(js.LDFLAGS, emscriptenAllowTableGrowth) {
		t.Errorf("raw js/wasm LDFLAGS %v do not allow wasm table growth for libffi closures", js.LDFLAGS)
	}
	if !slices.Contains(js.LDFLAGS, wantL) {
		t.Errorf("raw js/wasm LDFLAGS %v do not search %s", js.LDFLAGS, wantL)
	}

	named, err := Use("", "", "emscripten", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(named.LDFLAGS, emscriptenAllowTableGrowth) {
		t.Errorf("emscripten LDFLAGS %v do not allow wasm table growth for libffi closures", named.LDFLAGS)
	}
	if !slices.Contains(named.LDFLAGS, wantL) {
		t.Errorf("emscripten LDFLAGS %v do not search %s", named.LDFLAGS, wantL)
	}

	memory64, err := Use("", "", "emscripten-memory64", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	want64 := "-L" + filepath.Join(root, wasm64LibffiRelDir)
	if slices.Contains(memory64.LDFLAGS, wantL) {
		t.Errorf("emscripten-memory64 LDFLAGS %v unexpectedly search the wasm32 libffi archive", memory64.LDFLAGS)
	}
	if !slices.Contains(memory64.LDFLAGS, want64) {
		t.Errorf("emscripten-memory64 LDFLAGS %v do not search %s", memory64.LDFLAGS, want64)
	}
	if _, err := os.Stat(filepath.Join(root, wasm64LibffiRelDir, "libffi.a")); err != nil {
		t.Fatalf("vendored wasm64 libffi archive: %v", err)
	}

	wasi, err := use("wasip1", "wasm", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(wasi.LDFLAGS, wantL) || slices.Contains(wasi.LDFLAGS, emscriptenAllowTableGrowth) {
		t.Errorf("wasip1/wasm LDFLAGS %v unexpectedly include Emscripten libffi flags", wasi.LDFLAGS)
	}
}

func TestRawWasmStandardTagsRemainUnqualified(t *testing.T) {
	js, err := use("js", "wasm", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if js.WasmABI != WasmABIUnspecified || js.LLVMTarget != "" {
		t.Fatalf("raw js/wasm was relabeled as ABI %q, LLVM profile %q", js.WasmABI, js.LLVMTarget)
	}
	if !slices.Contains(js.LDFLAGS, "-sENVIRONMENT=web,worker") ||
		slices.Contains(js.LDFLAGS, "-sENVIRONMENT=web,worker,node") {
		t.Fatalf("raw js/wasm host environment unexpectedly changed: %v", js.LDFLAGS)
	}
	if slices.Contains(js.BuildTags, "llgo.wasm.emscripten") {
		t.Fatalf("raw js/wasm acquired an Emscripten source tag: %v", js.BuildTags)
	}

	wasi, err := use("wasip1", "wasm", false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatal(err)
	}
	if wasi.WasmABI != WasmABIUnspecified || wasi.LLVMTarget != "" {
		t.Fatalf("raw wasip1/wasm was relabeled as ABI %q, LLVM profile %q", wasi.WasmABI, wasi.LLVMTarget)
	}
	if slices.Contains(wasi.LDFLAGS, "-Wl,--import-memory,") || slices.Contains(wasi.LDFLAGS, "-Wl,--import-memory") {
		t.Fatal("raw single-worker WASI unexpectedly imports host memory")
	}
	if !wasi.WasmPostLink.Asyncify {
		t.Fatal("raw single-worker WASI does not request Asyncify post-link processing")
	}
	if slices.Contains(wasi.BuildTags, "llgo.wasm.wasi") {
		t.Fatalf("raw wasip1/wasm acquired a WASI C-profile source tag: %v", wasi.BuildTags)
	}
}

func writeWasmTargetFixture(t *testing.T, name, config string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "runtime", "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if name != "" {
		if err := os.WriteFile(filepath.Join(root, "targets", name+".json"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("LLGO_ROOT", root)
}

func TestWasmProfileValidationErrors(t *testing.T) {
	if _, err := useWithGOARMAndToolchain(
		"js", "wasm", "", false, false, optlevel.O2, lto.Off, false,
		NativeToolchainInput{}, WasmABI("invalid"),
	); err == nil || !strings.Contains(err.Error(), "unsupported WebAssembly ABI profile") {
		t.Fatalf("invalid direct profile error = %v", err)
	}

	t.Run("generic target", func(t *testing.T) {
		writeWasmTargetFixture(t, "invalid-wasm", `{
			"llvm-target":"wasm32-unknown-unknown",
			"cpu":"generic",
			"goos":"linux",
			"goarch":"wasm",
			"wasm-abi":"invalid"
		}`)
		_, err := UseTarget("invalid-wasm", optlevel.O2, lto.Off)
		if err == nil || !strings.Contains(err.Error(), "unsupported WebAssembly ABI profile") {
			t.Fatalf("invalid target profile error = %v", err)
		}
	})

	t.Run("missing named target", func(t *testing.T) {
		writeWasmTargetFixture(t, "", "")
		_, err := Use("", "", "emscripten", false, false, optlevel.O2, lto.Off, false)
		if err == nil || !strings.Contains(err.Error(), "failed to resolve target emscripten") {
			t.Fatalf("missing named target error = %v", err)
		}
	})

	t.Run("invalid named profile", func(t *testing.T) {
		writeWasmTargetFixture(t, "emscripten", `{
			"llvm-target":"wasm32-unknown-emscripten",
			"goos":"js",
			"goarch":"wasm",
			"wasm-abi":"invalid"
		}`)
		_, err := Use("", "", "emscripten", false, false, optlevel.O2, lto.Off, false)
		if err == nil || !strings.Contains(err.Error(), "unsupported WebAssembly ABI profile") {
			t.Fatalf("invalid named profile error = %v", err)
		}
	})

	t.Run("invalid named platform", func(t *testing.T) {
		writeWasmTargetFixture(t, "emscripten", `{
			"llvm-target":"wasm32-unknown-emscripten",
			"goos":"plan9",
			"goarch":"wasm",
			"wasm-abi":"emscripten"
		}`)
		_, err := Use("", "", "emscripten", false, false, optlevel.O2, lto.Off, false)
		if err == nil || !strings.Contains(err.Error(), "unsupported GOOS for WebAssembly") {
			t.Fatalf("invalid named platform error = %v", err)
		}
	})

	t.Run("LLVM target mismatch", func(t *testing.T) {
		writeWasmTargetFixture(t, "emscripten", `{
			"llvm-target":"wasm32-wrong-emscripten",
			"goos":"js",
			"goarch":"wasm",
			"wasm-abi":"emscripten"
		}`)
		_, err := Use("", "", "emscripten", false, false, optlevel.O2, lto.Off, false)
		if err == nil || !strings.Contains(err.Error(), "requires \"wasm32-unknown-emscripten\"") {
			t.Fatalf("mismatched target error = %v", err)
		}
	})
}

func TestUseTargetESPClang(t *testing.T) {
	// Use the real toolchain: UseTarget queries its version for the library
	// cache key and builds the target libraries before returning.
	embeddedRoot, err := getESPClangRoot(true)
	if err != nil {
		t.Fatal(err)
	}

	export, err := UseTarget("rp2040", optlevel.Oz, lto.Thin)
	if err != nil {
		t.Fatal(err)
	}
	wantCC := filepath.Join(embeddedRoot, "bin", "clang++")
	if export.CC != wantCC || export.ClangRoot != embeddedRoot {
		t.Fatalf("RP2040 compiler = %q under %q, want %q under %q",
			export.CC, export.ClangRoot, wantCC, embeddedRoot)
	}
	wantLinker := filepath.Join(embeddedRoot, "bin", "ld.lld")
	if export.Linker != wantLinker {
		t.Fatalf("RP2040 linker = %q, want %q", export.Linker, wantLinker)
	}
}

func TestUseTargetESPClangDownloadError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	llgoRoot := t.TempDir()
	runtimeDir := filepath.Join(llgoRoot, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	targetsDir := filepath.Join(llgoRoot, "targets")
	if err := os.MkdirAll(targetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(targetsDir, "esp-test.json"),
		[]byte(`{"llvm-target":"xtensa","cpu":"esp32","build-tags":["esp"]}`), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", llgoRoot)

	originalCacheRoot := cacheRoot
	originalBaseURL := espClangBaseUrl
	cacheDir := t.TempDir()
	cacheRoot = func() string { return cacheDir }
	espClangBaseUrl = server.URL
	t.Cleanup(func() {
		cacheRoot = originalCacheRoot
		espClangBaseUrl = originalBaseURL
	})

	_, err := UseTarget("esp-test", optlevel.Oz, lto.Thin)
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("UseTarget(esp-test) error = %v, want download 404", err)
	}
}

func TestUseWithTarget(t *testing.T) {
	// Test target-based configuration takes precedence
	export, err := Use("linux", "amd64", "esp32", false, true, optlevel.Oz, lto.Thin, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if LLVM target is in CCFLAGS
	found := slices.Contains(export.CCFLAGS, "-mcpu=esp32")
	if !found {
		t.Errorf("Expected CPU generic in CCFLAGS, got %v", export.CCFLAGS)
	}

	// Test fallback to goos/goarch when no target specified
	export, err = Use(runtime.GOOS, runtime.GOARCH, "", false, false, optlevel.O2, lto.Thin, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use native configuration (only check for macOS since that's where tests run)
	if runtime.GOOS == "darwin" && len(export.LDFLAGS) == 0 {
		t.Error("Expected LDFLAGS to be set for native build")
	}
	wantDebugInfo := nativeDebugInfoPolicy(export.Toolchain)
	if export.DebugInfo.AlwaysOmit != wantDebugInfo.AlwaysOmit ||
		!slices.Equal(export.DebugInfo.OmitLinkFlags, wantDebugInfo.OmitLinkFlags) ||
		!slices.Equal(export.DebugInfo.PreserveLinkFlags, wantDebugInfo.PreserveLinkFlags) {
		t.Fatalf("native debug-info policy = %+v, want %+v", export.DebugInfo, wantDebugInfo)
	}
}

func TestNativeToolchain(t *testing.T) {
	tests := []struct {
		goos string
		want NativeToolchain
	}{
		{"darwin", NativeToolchain{ABI: PlatformABIDarwin, ObjectFormat: ObjectFormatMachO, Driver: DriverFlavorClangGNU, Linker: LinkerFlavorMachO}},
		{"linux", NativeToolchain{ABI: PlatformABIGNU, ObjectFormat: ObjectFormatELF, Driver: DriverFlavorClangGNU, Linker: LinkerFlavorELFLLD}},
		{"windows", NativeToolchain{ABI: PlatformABIMsvc, ObjectFormat: ObjectFormatCOFF, Driver: DriverFlavorClangGNU, Linker: LinkerFlavorCOFFLLD}},
		{"freebsd", NativeToolchain{}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			if got := nativeToolchain(tt.goos); got != tt.want {
				t.Fatalf("nativeToolchain(%q) = %+v, want %+v", tt.goos, got, tt.want)
			}
		})
	}
}

func TestNativeDebugInfoPolicy(t *testing.T) {
	tests := []struct {
		goos     string
		omit     []string
		preserve []string
	}{
		{goos: "darwin", omit: []string{"-Wl,-S"}},
		{goos: "linux", omit: []string{"-Wl,-S"}},
		{goos: "windows", omit: []string{"-Wl,/debug:none"}, preserve: []string{"-Wl,/debug:dwarf"}},
		{goos: "freebsd"},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			policy := nativeDebugInfoPolicy(nativeToolchain(tt.goos))
			if policy.AlwaysOmit || !slices.Equal(policy.OmitLinkFlags, tt.omit) || !slices.Equal(policy.PreserveLinkFlags, tt.preserve) {
				t.Fatalf("nativeDebugInfoPolicy(%q) = %+v, want omit=%v preserve=%v", tt.goos, policy, tt.omit, tt.preserve)
			}
		})
	}
}

func TestOptimizationFlagPlacement(t *testing.T) {
	export, err := UseTarget("rp2040", optlevel.Oz, lto.Off)
	if err != nil {
		t.Fatalf("UseTargetWithOptLevel(rp2040) failed: %v", err)
	}
	if len(export.CCFLAGS) == 0 || export.CCFLAGS[0] != "-Oz" {
		t.Fatalf("target CCFLAGS = %v, want first flag -Oz", export.CCFLAGS)
	}

	export, err = Use(runtime.GOOS, runtime.GOARCH, "", false, false, optlevel.O3, lto.Off, false)
	if err != nil {
		t.Fatalf("UseWithOptLevel(host, O3) failed: %v", err)
	}
	if !slices.Contains(export.CCFLAGS, "-O3") {
		t.Fatalf("host CCFLAGS = %v, want -O3", export.CCFLAGS)
	}
	wantTarget := llvm.GetTargetTriple(runtime.GOOS, runtime.GOARCH)
	if export.Toolchain.TargetTriple != "" {
		wantTarget = export.Toolchain.TargetTriple
	}
	if !hasFlagValue(export.CCFLAGS, "-target", wantTarget) {
		t.Fatalf("host CCFLAGS = %v, want native -target", export.CCFLAGS)
	}
	if !hasFlagValue(export.LDFLAGS, "-target", wantTarget) {
		t.Fatalf("host LDFLAGS = %v, want native -target", export.LDFLAGS)
	}
}

func TestLTOLinkerOptFlag(t *testing.T) {
	tests := []struct {
		level optlevel.Level
		want  string
	}{
		{level: optlevel.O0, want: "--lto-O0"},
		{level: optlevel.O1, want: "--lto-O1"},
		{level: optlevel.O2, want: "--lto-O2"},
		{level: optlevel.O3, want: "--lto-O3"},
		{level: optlevel.Os, want: "--lto-O2"},
		{level: optlevel.Oz, want: "--lto-O2"},
		{level: optlevel.Unset, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if got := ltoLinkerOptFlag(tt.level); got != tt.want {
				t.Fatalf("ltoLinkerOptFlag(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestNativeWindowsLLDFlags(t *testing.T) {
	toolchain := nativeToolchain("windows")
	flags := nativeLLDFlags(toolchain, optlevel.O2, lto.Off)
	for _, want := range []string{
		"-fuse-ld=lld",
		"-Wl,/errorlimit:0",
		"-Wl,/opt:noicf",
	} {
		if !slices.Contains(flags, want) {
			t.Errorf("native Windows LLD flags = %v, want %q", flags, want)
		}
	}
	for _, unwanted := range []string{"-Wl,--error-limit=0", "-Wl,--icf=none"} {
		if slices.Contains(flags, unwanted) {
			t.Errorf("native Windows LLD flags = %v, do not want %q", flags, unwanted)
		}
	}

	thin := nativeLLDFlags(toolchain, optlevel.O3, lto.Thin)
	for _, want := range []string{"-flto=thin", "-Wl,/opt:lldlto=3"} {
		if !slices.Contains(thin, want) {
			t.Errorf("native Windows ThinLTO flags = %v, want %q", thin, want)
		}
	}
	if slices.Contains(thin, "-Wl,--lto-O3") {
		t.Errorf("native Windows ThinLTO flags = %v, contain ELF LTO syntax", thin)
	}
}

func TestCOFFLTOLevel(t *testing.T) {
	for _, tt := range []struct {
		level optlevel.Level
		want  string
	}{
		{optlevel.O0, "0"},
		{optlevel.O1, "1"},
		{optlevel.O2, "2"},
		{optlevel.O3, "3"},
		{optlevel.Os, "2"},
		{optlevel.Oz, "2"},
	} {
		if got := coffLTOLevel(tt.level); got != tt.want {
			t.Errorf("coffLTOLevel(%s) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestNativeWindowsSectionFlags(t *testing.T) {
	ccflags, ldflags := nativeSectionFlags(nativeToolchain("windows"))
	for _, want := range []string{"-fdata-sections", "-ffunction-sections"} {
		if !slices.Contains(ccflags, want) {
			t.Errorf("native Windows CCFLAGS = %v, want %q", ccflags, want)
		}
	}
	for _, want := range []string{"-fdata-sections", "-ffunction-sections", "--rtlib=compiler-rt", "-Wl,/opt:ref", "-llegacy_stdio_definitions"} {
		if !slices.Contains(ldflags, want) {
			t.Errorf("native Windows LDFLAGS = %v, want %q", ldflags, want)
		}
	}
	for _, unwanted := range []string{"--gc-sections", "-latomic", "-lpthread", "-ldl"} {
		if slices.Contains(ldflags, unwanted) {
			t.Errorf("native Windows LDFLAGS = %v, do not want %q", ldflags, unwanted)
		}
	}
}

func TestNativeWindowsExportFlags(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires a native Windows host")
	}

	export, err := use("windows", runtime.GOARCH, false, false, optlevel.O2, lto.Thin, false)
	if err != nil {
		t.Fatal(err)
	}
	wantTriple, err := windowsTargetTriple(runtime.GOARCH, export.Toolchain.ABI)
	if err != nil {
		t.Fatal(err)
	}
	if export.Toolchain.ObjectFormat != ObjectFormatCOFF ||
		export.Toolchain.Driver != DriverFlavorClangGNU ||
		export.Toolchain.TargetTriple != wantTriple {
		t.Fatalf("native Windows toolchain = %+v", export.Toolchain)
	}
	var wanted, unwanted []string
	switch export.Toolchain.ABI {
	case PlatformABIMsvc:
		if export.Toolchain.Linker != LinkerFlavorCOFFLLD ||
			export.Toolchain.CRT != CRTFlavorUCRT ||
			export.Toolchain.CXXRuntime != CXXRuntimeMSVC {
			t.Fatalf("native MSVC toolchain = %+v", export.Toolchain)
		}
		wanted = []string{
			"-Wl,/errorlimit:0",
			"-Wl,/opt:noicf",
			"-Wl,/opt:ref",
			"-Wl,/opt:lldlto=2",
			"-llegacy_stdio_definitions",
		}
		unwanted = []string{
			"-Wl,--error-limit=0",
			"-Wl,--icf=none",
			"--gc-sections",
		}
	case PlatformABIGNU:
		if export.Toolchain.Linker != LinkerFlavorMinGWLLD ||
			export.Toolchain.CRT != CRTFlavorUnknown ||
			export.Toolchain.CXXRuntime != CXXRuntimeUnknown {
			t.Fatalf("native GNU/MinGW toolchain = %+v", export.Toolchain)
		}
		wanted = []string{
			"-Wl,--error-limit=0",
			"-Wl,--icf=none",
			"-Wl,--gc-sections",
			"-Wl,--lto-O2",
		}
		unwanted = []string{
			"-Wl,/errorlimit:0",
			"-Wl,/opt:noicf",
			"-Wl,/opt:ref",
			"-Wl,/opt:lldlto=2",
			"-llegacy_stdio_definitions",
		}
	default:
		t.Fatalf("unsupported native Windows ABI: %+v", export.Toolchain)
	}
	for _, want := range wanted {
		if !slices.Contains(export.LDFLAGS, want) {
			t.Errorf("native Windows LDFLAGS = %v, want %q", export.LDFLAGS, want)
		}
	}
	for _, flag := range append(unwanted, "-latomic", "-lpthread", "-ldl") {
		if slices.Contains(export.LDFLAGS, flag) {
			t.Errorf("native Windows LDFLAGS = %v, do not want %q", export.LDFLAGS, flag)
		}
	}
}

func TestUsesNativePlatformToolchain(t *testing.T) {
	for _, test := range []struct {
		name                                   string
		hostOS, hostArch, targetOS, targetArch string
		resolveWindows, want                   bool
	}{
		{name: "same platform", hostOS: "linux", hostArch: "amd64", targetOS: "linux", targetArch: "amd64", want: true},
		{name: "Windows linked cross architecture", hostOS: "windows", hostArch: "amd64", targetOS: "windows", targetArch: "arm64", resolveWindows: true, want: true},
		{name: "Windows IR-only cross architecture", hostOS: "windows", hostArch: "amd64", targetOS: "windows", targetArch: "arm64"},
		{name: "Windows linked cross host", hostOS: "darwin", hostArch: "arm64", targetOS: "windows", targetArch: "386", resolveWindows: true, want: true},
		{name: "Windows IR-only cross host", hostOS: "linux", hostArch: "amd64", targetOS: "windows", targetArch: "amd64"},
		{name: "non-Windows cross architecture", hostOS: "darwin", hostArch: "arm64", targetOS: "darwin", targetArch: "amd64"},
		{name: "cross OS", hostOS: "windows", hostArch: "amd64", targetOS: "linux", targetArch: "amd64"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := usesNativePlatformToolchain(test.hostOS, test.hostArch, test.targetOS, test.targetArch, test.resolveWindows); got != test.want {
				t.Fatalf("usesNativePlatformToolchain() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDevLTOGlobalDCEUseLTOFlagsControlledByOption(t *testing.T) {
	export, err := use(runtime.GOOS, runtime.GOARCH, false, false, optlevel.O2, lto.Off, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	for _, flag := range export.CCFLAGS {
		if strings.HasPrefix(flag, "-flto") {
			t.Fatalf("unexpected LTO ccflag when disabled: %q", flag)
		}
	}
	for _, flag := range export.LDFLAGS {
		if strings.Contains(flag, "lto-") {
			t.Fatalf("unexpected LTO ldflag when disabled: %q", flag)
		}
	}

	thin, err := use(runtime.GOOS, runtime.GOARCH, false, false, optlevel.O2, lto.Thin, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !slices.Contains(thin.CCFLAGS, "-flto=thin") {
		t.Fatalf("missing thin LTO ccflag: %v", thin.CCFLAGS)
	}
	if slices.Contains(thin.CCFLAGS, "-fvirtual-function-elimination") {
		t.Fatalf("unexpected virtual function elimination ccflag for thin LTO: %v", thin.CCFLAGS)
	}
	if slices.Contains(thin.CCFLAGS, "-fwhole-program-vtables") {
		t.Fatalf("unexpected whole-program vtables ccflag for thin LTO: %v", thin.CCFLAGS)
	}
	if !slices.Contains(thin.LDFLAGS, "-flto=thin") {
		t.Fatalf("missing thin LTO link driver flag: %v", thin.LDFLAGS)
	}
	wantLTOOpt := nativeLTOOptFlag(thin.Toolchain, optlevel.O2)
	if !slices.Contains(thin.LDFLAGS, wantLTOOpt) {
		t.Fatalf("missing thin LTO linker opt flag: %v", thin.LDFLAGS)
	}

	thinSize, err := use(runtime.GOOS, runtime.GOARCH, false, false, optlevel.Oz, lto.Thin, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !slices.Contains(thinSize.LDFLAGS, nativeLTOOptFlag(thinSize.Toolchain, optlevel.Oz)) {
		t.Fatalf("missing numeric thin LTO linker opt flag for Oz: %v", thinSize.LDFLAGS)
	}
	if slices.Contains(thinSize.LDFLAGS, "-Wl,--lto-Oz") || slices.Contains(thinSize.LDFLAGS, "-Wl,/opt:lldlto=Oz") {
		t.Fatalf("invalid size-valued thin LTO linker opt flag: %v", thinSize.LDFLAGS)
	}

	full, err := use(runtime.GOOS, runtime.GOARCH, false, false, optlevel.O2, lto.Full, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !slices.Contains(full.CCFLAGS, "-flto=full") {
		t.Fatalf("missing full LTO ccflag: %v", full.CCFLAGS)
	}
	if slices.Contains(full.CCFLAGS, "-fvirtual-function-elimination") {
		t.Fatalf("unexpected virtual function elimination ccflag when global DCE is disabled: %v", full.CCFLAGS)
	}
	if slices.Contains(full.CCFLAGS, "-fwhole-program-vtables") {
		t.Fatalf("unexpected whole-program vtables ccflag when global DCE is disabled: %v", full.CCFLAGS)
	}
	if !slices.Contains(full.LDFLAGS, "-flto=full") {
		t.Fatalf("missing full LTO link driver flag: %v", full.LDFLAGS)
	}

	fullGlobalDCE, err := use(runtime.GOOS, runtime.GOARCH, false, false, optlevel.O2, lto.Full, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !slices.Contains(fullGlobalDCE.CCFLAGS, "-fvirtual-function-elimination") {
		t.Fatalf("missing virtual function elimination ccflag for full LTO with global DCE: %v", fullGlobalDCE.CCFLAGS)
	}
	if !slices.Contains(fullGlobalDCE.CCFLAGS, "-fwhole-program-vtables") {
		t.Fatalf("missing whole-program vtables ccflag for full LTO with global DCE: %v", fullGlobalDCE.CCFLAGS)
	}
}

func nativeLTOOptFlag(toolchain NativeToolchain, level optlevel.Level) string {
	if toolchain.Linker == LinkerFlavorCOFFLLD {
		return "-Wl,/opt:lldlto=" + coffLTOLevel(level)
	}
	return "-Wl," + ltoLinkerOptFlag(level)
}

func hasMllvmOption(flags []string, opt string) bool {
	for i := 0; i+1 < len(flags); i++ {
		if flags[i] == "-mllvm" && flags[i+1] == opt {
			return true
		}
	}
	return false
}

func hasFlagValue(flags []string, flag, value string) bool {
	for i := 0; i+1 < len(flags); i++ {
		if flags[i] == flag && flags[i+1] == value {
			return true
		}
	}
	return false
}

func TestUseTargetCodegenFlagsOnlyAddedToLDFlagsWithLTO(t *testing.T) {
	const target = "k210"

	noLTO, err := UseTarget(target, optlevel.Oz, lto.Off)
	if err != nil {
		t.Fatalf("UseTarget(%q, off) error: %v", target, err)
	}
	if hasMllvmOption(noLTO.LDFLAGS, "-code-model=medium") {
		t.Fatalf("unexpected -mllvm -code-model=medium in LDFLAGS when LTO disabled: %v", noLTO.LDFLAGS)
	}
	if hasMllvmOption(noLTO.LDFLAGS, "-target-abi=lp64") {
		t.Fatalf("unexpected -mllvm -target-abi=lp64 in LDFLAGS when LTO disabled: %v", noLTO.LDFLAGS)
	}
	if !slices.Contains(noLTO.CCFLAGS, "-mcmodel=medium") {
		t.Fatalf("missing -mcmodel=medium in CCFLAGS: %v", noLTO.CCFLAGS)
	}
	if !slices.Contains(noLTO.CCFLAGS, "-mabi=lp64") {
		t.Fatalf("missing -mabi=lp64 in CCFLAGS: %v", noLTO.CCFLAGS)
	}

	withLTO, err := UseTarget(target, optlevel.Oz, lto.Thin)
	if err != nil {
		t.Fatalf("UseTarget(%q, thin) error: %v", target, err)
	}
	if !slices.Contains(withLTO.CCFLAGS, "-flto=thin") {
		t.Fatalf("missing thin LTO ccflag: %v", withLTO.CCFLAGS)
	}
	if !slices.Contains(withLTO.LDFLAGS, "--lto-O2") {
		t.Fatalf("missing numeric thin LTO linker opt flag for Oz: %v", withLTO.LDFLAGS)
	}
	if slices.Contains(withLTO.LDFLAGS, "--lto-Oz") {
		t.Fatalf("invalid size-valued thin LTO linker opt flag: %v", withLTO.LDFLAGS)
	}
	if !hasMllvmOption(withLTO.LDFLAGS, "-code-model=medium") {
		t.Fatalf("missing -mllvm -code-model=medium in LDFLAGS when LTO enabled: %v", withLTO.LDFLAGS)
	}
	if !hasMllvmOption(withLTO.LDFLAGS, "-target-abi=lp64") {
		t.Fatalf("missing -mllvm -target-abi=lp64 in LDFLAGS when LTO enabled: %v", withLTO.LDFLAGS)
	}

	fullLTO, err := UseTarget(target, optlevel.Oz, lto.Full)
	if err != nil {
		t.Fatalf("UseTarget(%q, full) error: %v", target, err)
	}
	if !slices.Contains(fullLTO.CCFLAGS, "-flto=full") {
		t.Fatalf("missing full LTO ccflag: %v", fullLTO.CCFLAGS)
	}
}

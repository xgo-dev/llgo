//go:build !llgo
// +build !llgo

package crosscompile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	compilepkg "github.com/goplus/llgo/internal/crosscompile/compile"
	"github.com/goplus/llgo/internal/optlevel"
)

const (
	sysrootPrefix     = "--sysroot="
	resourceDirPrefix = "-resource-dir="
	includePrefix     = "-I"
	libPrefix         = "-L"
)

func resetMacOSSysrootForTest(t *testing.T) {
	t.Helper()
	macOSSysrootMu.Lock()
	oldCached := macOSSysrootCached
	oldLookup := macOSSysrootLookup
	macOSSysrootCached = ""
	macOSSysrootMu.Unlock()
	t.Cleanup(func() {
		macOSSysrootMu.Lock()
		macOSSysrootCached = oldCached
		macOSSysrootLookup = oldLookup
		macOSSysrootMu.Unlock()
	})
}

func TestGetMacOSSysrootCachesSuccess(t *testing.T) {
	resetMacOSSysrootForTest(t)
	calls := 0
	macOSSysrootLookup = func() (string, error) {
		calls++
		return "/test/sdk", nil
	}
	first, err := getMacOSSysroot()
	if err != nil {
		t.Fatalf("first getMacOSSysroot: %v", err)
	}
	second, err := getMacOSSysroot()
	if err != nil {
		t.Fatalf("second getMacOSSysroot: %v", err)
	}
	if first != "/test/sdk" || second != "/test/sdk" {
		t.Fatalf("sysroots = %q, %q; want cached /test/sdk", first, second)
	}
	if calls != 1 {
		t.Fatalf("lookup calls = %d, want 1", calls)
	}
}

func TestGetMacOSSysrootDoesNotCacheErrors(t *testing.T) {
	resetMacOSSysrootForTest(t)
	calls := 0
	macOSSysrootLookup = func() (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("temporary xcrun failure")
		}
		return "/test/sdk", nil
	}
	if _, err := getMacOSSysroot(); err == nil {
		t.Fatal("first getMacOSSysroot succeeded, want error")
	}
	got, err := getMacOSSysroot()
	if err != nil {
		t.Fatalf("second getMacOSSysroot: %v", err)
	}
	if got != "/test/sdk" {
		t.Fatalf("sysroot = %q, want /test/sdk", got)
	}
	if calls != 2 {
		t.Fatalf("lookup calls = %d, want retry after error", calls)
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

	// Set environment variable for cache directory
	oldEnv := os.Getenv("LLGO_CACHE_DIR")
	os.Setenv("LLGO_CACHE_DIR", tempDir)
	defer os.Setenv("LLGO_CACHE_DIR", oldEnv)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			export, err := use(tc.goos, tc.goarch, false, false, optlevel.O2)

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
			export, err := UseTarget(tc.targetName, optlevel.Oz)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for target %s, but got none", tc.targetName)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for target %s: %v", tc.targetName, err)
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

func TestCompileWithConfigCompileError(t *testing.T) {
	tmpDir := t.TempDir()
	bad := filepath.Join(tmpDir, "bad.c")
	if err := os.WriteFile(bad, []byte("int bad(void) {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := compilepkg.CompileConfig{Groups: []compilepkg.CompileGroup{
		{OutputFileName: "libbad.a", Files: []string{bad}},
	}}
	if _, err := compileWithConfig(cfg, tmpDir, compilepkg.CompileOptions{CC: "clang"}); err == nil {
		t.Fatal("compileWithConfig should report compile errors")
	}
}

func TestCompileWithConfigSkipsObjectOutputLinkFlags(t *testing.T) {
	tmpDir := t.TempDir()
	writeC := func(name, symbol string) string {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("int "+symbol+"(void) { return 1; }\n"), 0644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	cfg := compilepkg.CompileConfig{Groups: []compilepkg.CompileGroup{
		{OutputFileName: "libkeep.a", Files: []string{writeC("keep.c", "keep")}},
		{OutputFileName: "startup.o", Files: []string{writeC("startup.c", "startup")}},
	}}

	ldflags, err := compileWithConfig(cfg, tmpDir, compilepkg.CompileOptions{CC: "clang"})
	if err != nil {
		t.Fatalf("compileWithConfig: %v", err)
	}
	if !slices.Contains(ldflags, "-lkeep") {
		t.Fatalf("ldflags = %v, want -lkeep", ldflags)
	}
	if slices.Contains(ldflags, "-lstartup.o") {
		t.Fatalf("ldflags = %v, object outputs should not become library flags", ldflags)
	}
}

func TestUseWithTarget(t *testing.T) {
	// Test target-based configuration takes precedence
	export, err := Use("linux", "amd64", "esp32", false, true, optlevel.Oz)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if LLVM target is in CCFLAGS
	found := slices.Contains(export.CCFLAGS, "-mcpu=esp32")
	if !found {
		t.Errorf("Expected CPU generic in CCFLAGS, got %v", export.CCFLAGS)
	}

	// Test fallback to goos/goarch when no target specified
	export, err = Use(runtime.GOOS, runtime.GOARCH, "", false, false, optlevel.O2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use native configuration (only check for macOS since that's where tests run)
	if runtime.GOOS == "darwin" && len(export.LDFLAGS) == 0 {
		t.Error("Expected LDFLAGS to be set for native build")
	}
}

func TestOptimizationFlagPlacement(t *testing.T) {
	export, err := UseTarget("rp2040", optlevel.Oz)
	if err != nil {
		t.Fatalf("UseTargetWithOptLevel(rp2040) failed: %v", err)
	}
	if len(export.CCFLAGS) == 0 || export.CCFLAGS[0] != "-Oz" {
		t.Fatalf("target CCFLAGS = %v, want first flag -Oz", export.CCFLAGS)
	}

	export, err = Use(runtime.GOOS, runtime.GOARCH, "", false, false, optlevel.O3)
	if err != nil {
		t.Fatalf("UseWithOptLevel(host, O3) failed: %v", err)
	}
	if !slices.Contains(export.CCFLAGS, "-O3") {
		t.Fatalf("host CCFLAGS = %v, want -O3", export.CCFLAGS)
	}
}

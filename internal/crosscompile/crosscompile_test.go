//go:build !llgo
// +build !llgo

package crosscompile

import (
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/lto"
	"github.com/goplus/llgo/internal/optlevel"
	"github.com/goplus/llgo/internal/xtool/llvm"
)

const (
	sysrootPrefix     = "--sysroot="
	resourceDirPrefix = "-resource-dir="
	includePrefix     = "-I"
	libPrefix         = "-L"
)

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
	if !hasFlagValue(export.CCFLAGS, "-target", llvm.GetTargetTriple(runtime.GOOS, runtime.GOARCH)) {
		t.Fatalf("host CCFLAGS = %v, want native -target", export.CCFLAGS)
	}
	if !hasFlagValue(export.LDFLAGS, "-target", llvm.GetTargetTriple(runtime.GOOS, runtime.GOARCH)) {
		t.Fatalf("host LDFLAGS = %v, want native -target", export.LDFLAGS)
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
	if !slices.Contains(thin.LDFLAGS, "-Wl,--lto-O2") {
		t.Fatalf("missing thin LTO linker opt flag: %v", thin.LDFLAGS)
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

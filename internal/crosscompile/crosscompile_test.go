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

	"github.com/xgo-dev/llgo/internal/llvmpayload"
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
	payload, err := llvmpayload.Default()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		goos, goarch string
		wantPlatform string
		wantVersion  string
	}{
		{"darwin", "arm64", "aarch64-apple-darwin", payload.Version()},
		{"linux", "amd64", "x86_64-linux-gnu", payload.Version()},
		{"windows", "amd64", espClangWindowsPlatform, espClangWindowsVersion},
		{"windows", "arm64", espClangWindowsPlatform, espClangWindowsVersion},
	}
	for _, test := range tests {
		platform := getESPClangPlatform(test.goos, test.goarch)
		if platform != test.wantPlatform {
			t.Errorf("getESPClangPlatform(%q, %q) = %q, want %q", test.goos, test.goarch, platform, test.wantPlatform)
			continue
		}
		artifact, err := espClangArtifact(payload, platform)
		if err != nil {
			t.Errorf("espClangArtifact(%q) error = %v", platform, err)
			continue
		}
		if artifact.Version != test.wantVersion {
			t.Errorf("espClangArtifact(%q) version = %q, want %q", platform, artifact.Version, test.wantVersion)
		}
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
			if export.CPU != tc.expectCPU {
				t.Fatalf("target %s exported CPU = %q, want %q", tc.targetName, export.CPU, tc.expectCPU)
			}

			// Check if LLVM target is in CCFLAGS
			if tc.expectLLVM != "" {
				found := false
				expectedLLVM := clangDriverTargetForHost(runtime.GOOS, tc.expectLLVM, export.BuildTags)
				expectedFlag := "--target=" + expectedLLVM
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

func TestUseTargetWindowsSystemClang(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows host toolchain selection")
	}

	export, err := UseTarget("rp2040", optlevel.Oz, lto.Thin)
	if err != nil {
		t.Fatal(err)
	}
	if export.CC != "clang++" {
		t.Fatalf("RP2040 compiler on Windows = %q, want clang++", export.CC)
	}
	if export.ClangRoot != "" {
		t.Fatalf("RP2040 Clang root on Windows = %q, want system toolchain", export.ClangRoot)
	}
	if export.Linker != "ld.lld" {
		t.Fatalf("RP2040 linker on Windows = %q, want ld.lld", export.Linker)
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
	originalResolver := resolveESPClangArtifact
	cacheDir := t.TempDir()
	cacheRoot = func() string { return cacheDir }
	resolveESPClangArtifact = func(_ llvmpayload.Manifest, platform string) (llvmpayload.Artifact, error) {
		return llvmpayload.Artifact{
			Platform: platform,
			Version:  "test",
			URL:      server.URL + "/clang-esp-test.tar.xz",
		}, nil
	}
	t.Cleanup(func() {
		cacheRoot = originalCacheRoot
		resolveESPClangArtifact = originalResolver
	})

	_, err := UseTarget("esp-test", optlevel.Oz, lto.Thin)
	if err == nil || !strings.Contains(err.Error(), "404 Not Found") {
		t.Fatalf("UseTarget(esp-test) error = %v, want download 404", err)
	}
}

func TestUseSystemClangForTarget(t *testing.T) {
	for _, test := range []struct {
		goos      string
		target    string
		buildTags []string
		want      bool
	}{
		{goos: "windows", target: "thumbv6m-unknown-unknown-eabi", want: true},
		{goos: "windows", target: "avr", want: true},
		{goos: "windows", target: "riscv32-esp-elf", buildTags: []string{"esp"}, want: false},
		{goos: "windows", target: "xtensa", buildTags: []string{"esp32", "esp"}, want: false},
		{goos: "windows", target: "xtensa", want: false},
		{goos: "linux", target: "thumbv6m-unknown-unknown-eabi", want: false},
	} {
		if got := useSystemClangForTarget(test.goos, test.target, test.buildTags); got != test.want {
			t.Errorf("useSystemClangForTarget(%q, %q, %v) = %v, want %v", test.goos, test.target, test.buildTags, got, test.want)
		}
	}
}

func TestClangDriverTargetForHost(t *testing.T) {
	for _, test := range []struct {
		goos      string
		target    string
		buildTags []string
		want      string
	}{
		{goos: "windows", target: "xtensa", buildTags: []string{"esp32", "esp"}, want: "xtensa-esp-unknown-elf"},
		{goos: "windows", target: "xtensa", want: "xtensa"},
		{goos: "windows", target: "riscv32-esp-elf", buildTags: []string{"esp"}, want: "riscv32-esp-elf"},
		{goos: "linux", target: "xtensa", buildTags: []string{"esp"}, want: "xtensa"},
		{goos: "darwin", target: "xtensa", buildTags: []string{"esp"}, want: "xtensa"},
	} {
		if got := clangDriverTargetForHost(test.goos, test.target, test.buildTags); got != test.want {
			t.Errorf("clangDriverTargetForHost(%q, %q, %v) = %q, want %q", test.goos, test.target, test.buildTags, got, test.want)
		}
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

func TestLLDLTOOptFlag(t *testing.T) {
	tests := []struct {
		level optlevel.Level
		want  string
	}{
		{optlevel.O0, "--lto-O0"},
		{optlevel.O1, "--lto-O1"},
		{optlevel.O2, "--lto-O2"},
		{optlevel.O3, "--lto-O3"},
		{optlevel.Os, "--lto-O2"},
		{optlevel.Oz, "--lto-O2"},
	}
	for _, test := range tests {
		got, err := lldLTOOptFlag(test.level)
		if err != nil || got != test.want {
			t.Errorf("lldLTOOptFlag(%v) = %q, %v; want %q", test.level, got, err, test.want)
		}
	}
	if _, err := lldLTOOptFlag(optlevel.Unset); err == nil {
		t.Fatal("lldLTOOptFlag accepted an unset level")
	}
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

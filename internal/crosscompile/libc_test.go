package crosscompile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile/compile"
	"github.com/xgo-dev/llgo/internal/crosscompile/compile/libc"
	"github.com/xgo-dev/llgo/internal/crosscompile/compile/rtlib"
)

const testCompilerKey = "llvm-22.1.4-deadbeef"

func TestCompilerVersionCacheKey(t *testing.T) {
	identityKey, err := compilerVersionCacheKey("Espressif clang version 22.1.4 (abc123)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(identityKey, "llvm-22.1.4-") {
		t.Fatalf("compiler identity = %q", identityKey)
	}
	warned, err := compilerVersionCacheKey(
		"clang++: warning: no multilib found matching flags: --target=riscv32-esp-unknown-elf [-Wmissing-multilib]\r\n"+
			"Espressif clang version 22.1.4 (abc123)\r\n",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if warned != identityKey {
		t.Fatalf("warning-prefixed compiler identity = %q, want %q", warned, identityKey)
	}

	first, err := compilerVersionCacheKey("clang version 22.1.4 (https://example.test/llvm abc123)\nInstalledDir: /one", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := compilerVersionCacheKey("clang version 22.1.4 (https://example.test/llvm abc123)\nInstalledDir: /two", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("cache key depends on install directory: %q != %q", first, second)
	}
	different, err := compilerVersionCacheKey("clang version 22.1.4 (https://example.test/llvm def456)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if different == first {
		t.Fatalf("cache key ignores compiler revision: %q", first)
	}
	patched, err := compilerVersionCacheKey("clang version 22.1.4 (https://example.test/llvm abc123)", []byte("llvm_source_patch_sha256=abc"))
	if err != nil {
		t.Fatal(err)
	}
	if patched == first {
		t.Fatalf("cache key ignores payload contract: %q", first)
	}
	if a, b := compiledLibraryCacheKey(first, []string{"-Oz", "-flto=thin"}), compiledLibraryCacheKey(first, []string{"-Oz"}); a == b {
		t.Fatalf("compiled library cache key ignores code-generation flags: %q", a)
	}
	if _, err := compilerVersionCacheKey("not a compiler version", nil); err == nil {
		t.Fatal("compilerVersionCacheKey accepted output without a version")
	}
}

func TestCompilerCacheKeyErrors(t *testing.T) {
	if _, err := compilerCacheKey(filepath.Join(t.TempDir(), "missing-clang")); err == nil {
		t.Fatal("compilerCacheKey accepted a missing compiler")
	}
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a shell script")
	}

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	clang := filepath.Join(binDir, "clang")
	if err := os.WriteFile(clang, []byte("#!/bin/sh\nprintf '%s\\n' 'clang version 22.1.4'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "LLGO-LLVM-MANIFEST.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := compilerCacheKey(clang); err == nil || !strings.Contains(err.Error(), "read compiler payload contract") {
		t.Fatalf("compilerCacheKey unreadable manifest error = %v", err)
	}
}

func TestCompiledLibraryDirTracksCompileGroups(t *testing.T) {
	baseDir := t.TempDir()
	library := compile.LibConfig{Name: "lib", Version: "1"}
	config := compile.CompileConfig{Groups: []compile.CompileGroup{
		{OutputFileName: "first.a", Files: []string{"first.c"}},
		{OutputFileName: "second.a", Files: []string{"second.c", "third.c"},
			CFlags: []string{"-DORIGINAL"}, CCFlags: []string{"-Oz"}, LDFlags: []string{"-s"}},
	}}
	original := compiledLibraryDir(baseDir, library, testCompilerKey, config)

	for _, test := range []struct {
		name string
		edit func(*compile.CompileGroup)
	}{
		{"archive", func(g *compile.CompileGroup) { g.OutputFileName = "renamed.a" }},
		{"source files", func(g *compile.CompileGroup) { g.Files = []string{"third.c", "second.c"} }},
		{"C flags", func(g *compile.CompileGroup) { g.CFlags = []string{"-DREPLACEMENT"} }},
		{"compiler flags", func(g *compile.CompileGroup) { g.CCFlags = []string{"-O2"} }},
		{"linker flags", func(g *compile.CompileGroup) { g.LDFlags = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := config
			changed.Groups = slices.Clone(config.Groups)
			test.edit(&changed.Groups[1])
			if got := compiledLibraryDir(baseDir, library, testCompilerKey, changed); got == original {
				t.Fatalf("reused archive directory %q after changing %s", got, test.name)
			}
		})
	}
}

func TestGetLibcCompileConfigByName(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base")
	target := "armv7"
	mcpu := "cortex-m4"

	needSkipDownload = true
	t.Run("EmptyName", func(t *testing.T) {
		_, _, err := getLibcCompileConfigByName(baseDir, "", target, mcpu, testCompilerKey)
		if err == nil || err.Error() != "libc name cannot be empty" {
			t.Errorf("Expected empty name error, got: %v", err)
		}
	})

	t.Run("UnsupportedLibc", func(t *testing.T) {
		_, _, err := getLibcCompileConfigByName(baseDir, "invalid", target, mcpu, testCompilerKey)
		if err == nil || err.Error() != "unsupported libc: invalid" {
			t.Errorf("Expected unsupported libc error, got: %v", err)
		}
	})

	t.Run("Picolibc", func(t *testing.T) {
		outputDir, cfg, err := getLibcCompileConfigByName(baseDir, "picolibc", target, mcpu, testCompilerKey)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

		expectedDir := compiledLibraryDir(baseDir, libc.GetPicolibcConfig(), testCompilerKey, cfg)
		if outputDir != expectedDir {
			t.Fatalf("output dir = %q, want %q", outputDir, expectedDir)
		}
		expectedFile := filepath.Join(baseDir, libc.GetPicolibcConfig().String(), "newlib", "libc", "string", "memmem.c")
		if !slices.Contains(group.Files, expectedFile) {
			t.Errorf("Expected files [%s], got: %v", expectedFile, group.Files)
		}

		expectedFlag := "-I" + filepath.Join(baseDir, libc.GetPicolibcConfig().String())
		if !slices.Contains(group.CFlags, expectedFlag) {
			t.Errorf("Expected flags [%s], got: %v", expectedFlag, group.CFlags)
		}
	})

	t.Run("NewlibESP32", func(t *testing.T) {
		outputDir, cfg, err := getLibcCompileConfigByName(baseDir, "newlib-esp32", target, mcpu, testCompilerKey)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 3 {
			t.Fatalf("Expected 3 group, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

		expectedDir := compiledLibraryDir(baseDir, libc.GetNewlibESP32Config(), testCompilerKey, cfg)
		if outputDir != expectedDir {
			t.Fatalf("output dir = %q, want %q", outputDir, expectedDir)
		}
		expectedFile := filepath.Join(baseDir, libc.GetNewlibESP32Config().String(), "libgloss", "xtensa", "crt1-boards.S")
		if !slices.Contains(group.Files, expectedFile) {
			t.Errorf("Expected files [%s], got: %v", expectedFile, group.Files)
		}

		expectedFlags := "-I" + filepath.Join(baseDir, libc.GetNewlibESP32Config().String(), "libgloss")
		if !slices.Contains(group.CFlags, expectedFlags) {
			t.Errorf("Expected flags %v, got: %v", expectedFlags, group.CFlags)
		}
	})
}

func TestGetRTCompileConfigByName(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base")
	target := "wasm32"
	needSkipDownload = true

	t.Run("EmptyName", func(t *testing.T) {
		_, _, err := getRTCompileConfigByName(baseDir, "", target, testCompilerKey)
		if err == nil || err.Error() != "rt name cannot be empty" {
			t.Errorf("Expected empty name error, got: %v", err)
		}
	})

	t.Run("UnsupportedRT", func(t *testing.T) {
		_, _, err := getRTCompileConfigByName(baseDir, "invalid", target, testCompilerKey)
		if err == nil || err.Error() != "unsupported rt: invalid" {
			t.Errorf("Expected unsupported rt error, got: %v", err)
		}
	})

	t.Run("CompilerRT", func(t *testing.T) {
		outputDir, cfg, err := getRTCompileConfigByName(baseDir, "compiler-rt", target, testCompilerKey)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

		compilerRTConfig := rtlib.GetCompilerRTConfig()
		expectedDir := compiledLibraryDir(baseDir, compilerRTConfig, testCompilerKey, cfg)
		if outputDir != expectedDir {
			t.Fatalf("output dir = %q, want %q", outputDir, expectedDir)
		}
		expectedFile := filepath.Join(baseDir, compilerRTConfig.String(), "lib", "builtins", "absvdi2.c")
		if !slices.Contains(group.Files, expectedFile) {
			t.Errorf("Expected files [%s], got: %v", expectedFile, group.Files)
		}

		thumbDir, thumb, err := getRTCompileConfigByName(baseDir, "compiler-rt", "thumbv7m-unknown-unknown-eabi", testCompilerKey)
		if err != nil {
			t.Fatal(err)
		}
		if thumbDir == outputDir {
			t.Fatalf("Thumb and %s reuse archive directory %q", target, outputDir)
		}
		thumbFile := filepath.Join(baseDir, compilerRTConfig.String(), "lib", "builtins", "arm", "aeabi_memcpy.S")
		if len(thumb.Groups) != 1 || !slices.Contains(thumb.Groups[0].Files, thumbFile) {
			t.Fatalf("Thumb compiler-rt needs %s, got %v", thumbFile, thumb.Groups)
		}
	})
}

// TestCompilerRTCompileConfigPaths tests that file paths in the CompileConfig
// are correctly based on the provided baseDir for various target platforms.
func TestCompilerRTCompileConfigPaths(t *testing.T) {
	// Define test cases for different target platforms
	tests := []struct {
		name     string // Test case name
		baseDir  string // Input base directory
		target   string // Target platform
		expected string // Expected platform-specific file
	}{
		{
			name:     "RISC-V 32",
			baseDir:  filepath.FromSlash("/test/base/dir"),
			target:   "riscv32-unknown-elf",
			expected: filepath.FromSlash("riscv/mulsi3.S"), // Expected platform file for RISC-V 32
		},
		{
			name:     "RISC-V 64",
			baseDir:  filepath.FromSlash("/another/dir"),
			target:   "riscv64-unknown-elf",
			expected: "addtf3.c", // Expected platform file for RISC-V 64
		},
		{
			name:     "ARM",
			baseDir:  filepath.FromSlash("/arm/dir"),
			target:   "armv7-unknown-linux-gnueabihf",
			expected: filepath.FromSlash("arm/aeabi_cdcmp.S"), // Expected platform file for ARM
		},
		{
			name:     "AVR",
			baseDir:  filepath.FromSlash("/avr/dir"),
			target:   "avr",
			expected: filepath.FromSlash("avr/divmodhi4.S"), // Expected platform file for AVR
		},
		{
			name:     "XTENSA",
			baseDir:  filepath.FromSlash("/xtensa/dir"),
			target:   "xtensa",
			expected: filepath.FromSlash("xtensa/ieee754_sqrtf.S"), // Expected platform file for XTENSA
		},
	}
	needSkipDownload = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the compile configuration for this target
			cfg := rtlib.GetCompilerRTCompileConfig(tt.baseDir, tt.target)

			// Verify there is at least one compile group
			if len(cfg.Groups) == 0 {
				t.Fatal("CompileConfig has no groups")
			}

			group := cfg.Groups[0]
			found := false

			// Check all files in the group
			for _, file := range group.Files {
				// Verify file path starts with baseDir
				if !strings.HasPrefix(file, tt.baseDir) {
					t.Errorf("File path %q should start with baseDir %q", file, tt.baseDir)
				}

				// Verify file path contains the expected platform-specific file
				if strings.Contains(file, tt.expected) {
					found = true

					// Construct the expected full path
					expectedPath := filepath.Join(tt.baseDir, "lib", "builtins", tt.expected)

					// Verify the actual path matches the expected path
					if file != expectedPath {
						t.Errorf("Expected file path %q, got %q", expectedPath, file)
					}
				}
			}

			// Verify the platform-specific file was found
			if !found {
				t.Errorf("Expected platform-specific file %q not found in file list", tt.expected)
			}

			// Verify the output file name format
			expectedOutput := fmt.Sprintf("libclang_builtins-%s.a", tt.target)
			if !strings.HasSuffix(group.OutputFileName, expectedOutput) {
				t.Errorf("OutputFileName should end with %q, got %q", expectedOutput, group.OutputFileName)
			}
		})
	}
}

// TestCompilerRTCompileConfigPathRelations tests the general path relationships
// in the CompileConfig for a specific target.
func TestCompilerRTCompileConfigPathRelations(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base/dir")
	target := "riscv64-unknown-elf"

	// Get the compile configuration
	cfg := rtlib.GetCompilerRTCompileConfig(baseDir, target)

	// Verify there is at least one compile group
	if len(cfg.Groups) == 0 {
		t.Fatal("CompileConfig has no groups")
	}
	needSkipDownload = true

	group := cfg.Groups[0]

	// Check all files in the group
	for _, file := range group.Files {
		// Verify file path starts with baseDir
		if !strings.HasPrefix(file, baseDir) {
			t.Errorf("File path %q should start with baseDir %q", file, baseDir)
		}

		// Verify file path contains the expected subdirectory structure
		expectedSubdir := filepath.Join(baseDir, "lib", "builtins")
		if !strings.Contains(file, expectedSubdir) {
			t.Errorf("File path %q should contain %q", file, expectedSubdir)
		}
	}

	// Verify the output file name format
	expectedOutput := fmt.Sprintf("libclang_builtins-%s.a", target)
	if !strings.HasSuffix(group.OutputFileName, expectedOutput) {
		t.Errorf("OutputFileName should end with %q, got %q", expectedOutput, group.OutputFileName)
	}
}

// TestGetPicolibcCompileConfigPaths tests that all paths in the CompileConfig
// are correctly based on the provided baseDir.
func TestGetPicolibcCompileConfigPaths(t *testing.T) {
	// Define test cases with different base directories
	tests := []struct {
		name    string
		baseDir string
		target  string
	}{
		{
			name:    "Absolute path",
			baseDir: filepath.FromSlash("/test/base/dir"),
			target:  "riscv64-unknown-elf",
		},
		{
			name:    "Cleaned path",
			baseDir: filepath.Join(filepath.FromSlash("/test/base"), "parent", "..", "dir"),
			target:  "x86_64-pc-windows-msvc",
		},
		{
			name:    "Relative path",
			baseDir: filepath.FromSlash("test/base/dir"),
			target:  "armv7-unknown-linux-gnueabihf",
		},
	}
	needSkipDownload = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the compile configuration
			cfg := libc.GetPicolibcCompileConfig(tt.baseDir, tt.target)

			// Verify ExportCFlags paths
			for _, flag := range cfg.ExportCFlags {
				if strings.HasPrefix(flag, "-I") || strings.HasPrefix(flag, "-isystem") {
					path := strings.TrimPrefix(flag, "-I")
					path = strings.TrimPrefix(path, "-isystem")
					path = strings.TrimSpace(path)

					if !strings.HasPrefix(path, tt.baseDir) {
						t.Errorf("ExportCFlags path %q should start with baseDir %q", path, tt.baseDir)
					}
				}
			}

			// Verify there is at least one compile group
			if len(cfg.Groups) == 0 {
				t.Fatal("CompileConfig has no groups")
			}

			group := cfg.Groups[0]

			// Verify output file name format
			expectedOutput := fmt.Sprintf("libc-%s.a", tt.target)
			if group.OutputFileName != expectedOutput {
				t.Errorf("Expected OutputFileName %q, got %q", expectedOutput, group.OutputFileName)
			}

			// Verify all file paths start with baseDir
			for _, file := range group.Files {
				if !strings.HasPrefix(file, tt.baseDir) {
					t.Errorf("File path %q should start with baseDir %q", file, tt.baseDir)
				}

				// Verify file path contains expected subdirectories
				if !strings.Contains(file, filepath.Join(tt.baseDir, "newlib")) {
					t.Errorf("File path %q should contain 'newlib' subdirectory", file)
				}
			}

			// Verify CFlags paths
			for _, flag := range group.CFlags {
				if strings.HasPrefix(flag, "-I") {
					path := strings.TrimPrefix(flag, "-I")
					path = strings.TrimSpace(path)

					if !strings.HasPrefix(path, tt.baseDir) {
						t.Errorf("CFlags path %q should start with baseDir %q", path, tt.baseDir)
					}
				}
			}
		})
	}
}

// TestGetPicolibcCompileConfigSpecificPaths tests specific path constructions
// in the CompileConfig for a given baseDir and target.
func TestGetPicolibcCompileConfigSpecificPaths(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base/dir")
	target := "riscv64-unknown-elf"
	needSkipDownload = true

	// Get the compile configuration
	cfg := libc.GetPicolibcCompileConfig(baseDir, target)

	// Verify ExportCFlags
	expectedInclude := filepath.Join(baseDir, "newlib", "libc", "include")
	foundInclude := false
	for _, flag := range cfg.ExportCFlags {
		if flag == "-I"+baseDir || flag == "-isystem"+expectedInclude {
			foundInclude = true
		}
	}
	if !foundInclude {
		t.Errorf("Expected ExportCFlags to contain -I%s and -isystem%s", baseDir, expectedInclude)
	}

	// Verify there is at least one compile group
	if len(cfg.Groups) == 0 {
		t.Fatal("CompileConfig has no groups")
	}

	group := cfg.Groups[0]

	// Verify output file name
	expectedOutput := fmt.Sprintf("libc-%s.a", target)
	if group.OutputFileName != expectedOutput {
		t.Errorf("Expected OutputFileName %q, got %q", expectedOutput, group.OutputFileName)
	}

	// Verify specific file paths
	expectedFiles := []string{
		filepath.Join(baseDir, "newlib", "libc", "string", "memcpy.c"),
		filepath.Join(baseDir, "newlib", "libc", "string", "strlen.c"),
		filepath.Join(baseDir, "newlib", "libc", "stdlib", "nano-malloc.c"),
		filepath.Join(baseDir, "newlib", "libc", "tinystdio", "printf.c"),
	}

	for _, expected := range expectedFiles {
		found := false
		for _, file := range group.Files {
			if file == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %q not found in file list", expected)
		}
	}

	// Verify CFlags paths
	expectedCFlags := []string{
		"-I" + baseDir,
		"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
		"-I" + filepath.Join(baseDir, "newlib", "libm", "common"),
		"-I" + filepath.Join(baseDir, "newlib", "libc", "locale"),
		"-I" + filepath.Join(baseDir, "newlib", "libc", "tinystdio"),
	}

	for _, expected := range expectedCFlags {
		found := false
		for _, flag := range group.CFlags {
			if flag == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CFlag %q not found", expected)
		}
	}
}

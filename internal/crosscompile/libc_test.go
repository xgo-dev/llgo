//go:build !llgo

package crosscompile

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/crosscompile/compile/libc"
	"github.com/goplus/llgo/internal/crosscompile/compile/rtlib"
)

func TestGetLibcCompileConfigByName(t *testing.T) {
	baseDir := "/test/base"
	target := "armv7"
	mcpu := "cortex-m4"

	needSkipDownload = true
	t.Run("EmptyName", func(t *testing.T) {
		_, _, err := getLibcCompileConfigByName(baseDir, "", target, mcpu)
		if err == nil || err.Error() != "libc name cannot be empty" {
			t.Errorf("Expected empty name error, got: %v", err)
		}
	})

	t.Run("UnsupportedLibc", func(t *testing.T) {
		_, _, err := getLibcCompileConfigByName(baseDir, "invalid", target, mcpu)
		if err == nil || err.Error() != "unsupported libc: invalid" {
			t.Errorf("Expected unsupported libc error, got: %v", err)
		}
	})

	t.Run("Picolibc", func(t *testing.T) {
		_, cfg, err := getLibcCompileConfigByName(baseDir, "picolibc", target, mcpu)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

		expectedFile := filepath.Join(baseDir, libc.GetPicolibcConfig().String(), "newlib", "libc", "string", "memmem.c")
		if !slices.Contains(group.Files, expectedFile) {
			t.Errorf("Expected files [%s], got: %v", expectedFile, group.Files)
		}

		expectedFlag := "-I" + filepath.Join("/test", "base", libc.GetPicolibcConfig().String())
		if !slices.Contains(group.CFlags, expectedFlag) {
			t.Errorf("Expected flags [%s], got: %v", expectedFlag, group.CFlags)
		}
	})

	t.Run("NewlibESP32", func(t *testing.T) {
		_, cfg, err := getLibcCompileConfigByName(baseDir, "newlib-esp32", target, mcpu)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 6 {
			t.Fatalf("Expected 6 groups, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

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
	baseDir := "/test/base"
	target := "wasm32"
	needSkipDownload = true

	t.Run("EmptyName", func(t *testing.T) {
		_, _, err := getRTCompileConfigByName(baseDir, "", target)
		if err == nil || err.Error() != "rt name cannot be empty" {
			t.Errorf("Expected empty name error, got: %v", err)
		}
	})

	t.Run("UnsupportedRT", func(t *testing.T) {
		_, _, err := getRTCompileConfigByName(baseDir, "invalid", target)
		if err == nil || err.Error() != "unsupported rt: invalid" {
			t.Errorf("Expected unsupported rt error, got: %v", err)
		}
	})

	t.Run("CompilerRT", func(t *testing.T) {
		_, cfg, err := getRTCompileConfigByName(baseDir, "compiler-rt", target)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(cfg.Groups) != 1 {
			t.Fatalf("Expected 1 group, got %d", len(cfg.Groups))
		}
		group := cfg.Groups[0]

		expectedFile := filepath.Join(baseDir, rtlib.GetCompilerRTConfig().String(), "lib", "builtins", "absvdi2.c")
		if !slices.Contains(group.Files, expectedFile) {
			t.Errorf("Expected files [%s], got: %v", expectedFile, group.Files)
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
			baseDir:  "/test/base/dir",
			target:   "riscv32-unknown-elf",
			expected: "riscv/mulsi3.S", // Expected platform file for RISC-V 32
		},
		{
			name:     "RISC-V 64",
			baseDir:  "/another/dir",
			target:   "riscv64-unknown-elf",
			expected: "addtf3.c", // Expected platform file for RISC-V 64
		},
		{
			name:     "ARM",
			baseDir:  "/arm/dir",
			target:   "armv7-unknown-linux-gnueabihf",
			expected: "arm/aeabi_cdcmp.S", // Expected platform file for ARM
		},
		{
			name:     "AVR",
			baseDir:  "/avr/dir",
			target:   "avr",
			expected: "avr/divmodhi4.S", // Expected platform file for AVR
		},
		{
			name:     "XTENSA",
			baseDir:  "/xtensa/dir",
			target:   "xtensa",
			expected: "xtensa/ieee754_sqrtf.S", // Expected platform file for XTENSA
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
	baseDir := "/test/base/dir"
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
			name:    "Unix-like path",
			baseDir: "/test/base/dir",
			target:  "riscv64-unknown-elf",
		},
		{
			name:    "Windows-like path",
			baseDir: "C:\\test\\base\\dir",
			target:  "x86_64-pc-windows-msvc",
		},
		{
			name:    "Relative path",
			baseDir: "test/base/dir",
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
	baseDir := "/test/base/dir"
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

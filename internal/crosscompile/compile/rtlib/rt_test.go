package rtlib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCompilerRTConfig_LibConfig(t *testing.T) {
	config := GetCompilerRTConfig()

	// Test basic configuration fields
	expectedName := "compiler-rt"
	if config.Name != expectedName {
		t.Errorf("Expected Name '%s', got '%s'", expectedName, config.Name)
	}

	expectedVersion := "xtensa_release_22.1.4_20260903"
	if config.Version != expectedVersion {
		t.Errorf("Expected Version '%s', got '%s'", expectedVersion, config.Version)
	}

	expectedUrl := "https://github.com/goplus/compiler-rt/archive/refs/tags/xtensa_release_22.1.4_20260903.tar.gz"
	if config.Url != expectedUrl {
		t.Errorf("Expected Url '%s', got '%s'", expectedUrl, config.Url)
	}

	expectedArchiveSrcDir := "compiler-rt-xtensa_release_22.1.4_20260903"
	if config.ResourceSubDir != expectedArchiveSrcDir {
		t.Errorf("Expected ResourceSubDir '%s', got '%s'", expectedArchiveSrcDir, config.ResourceSubDir)
	}

	// Test String() method
	expectedString := "compiler-rt-xtensa_release_22.1.4_20260903"
	if config.String() != expectedString {
		t.Errorf("Expected String() '%s', got '%s'", expectedString, config.String())
	}
}

func TestPlatformSpecifiedFiles(t *testing.T) {
	tests := []struct {
		target   string
		expected int // Number of expected files
	}{
		{"riscv32-unknown-elf", 5},
		{"riscv32-esp-elf", 5},
		{"riscv64-unknown-elf", 27},
		{"arm-none-eabi", 19},
		{"thumbv7m-unknown-unknown-eabi", 19},
		{"avr-unknown-elf", 6},
		{"xtensa", 2},
		{"x86_64-pc-windows", 0},
	}

	builtinsDir := filepath.FromSlash("/test/builtins")
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := platformSpecifiedFiles(builtinsDir, tt.target)
			if len(result) != tt.expected {
				t.Errorf("For target %s, expected %d files, got %d", tt.target, tt.expected, len(result))
			}
		})
	}
}

// TestRiscv32UsesVendoredAtomic pins the riscv32 exclusion of compiler-rt's
// atomic.c: every riscv32 rtlib must link the vendored interrupt-masked
// fallback, whose symbols are weak, in its place.
func TestRiscv32UsesVendoredAtomic(t *testing.T) {
	for _, target := range []string{"riscv32-unknown-elf", "riscv32-esp-elf"} {
		t.Run(target, func(t *testing.T) {
			result := platformSpecifiedFiles(filepath.FromSlash("/test/builtins"), target)
			var hasVendored, hasCompilerRtAtomic bool
			for _, file := range result {
				if file == espAtomicRiscv32File() {
					hasVendored = true
				}
				if strings.HasSuffix(file, "atomic.c") {
					hasCompilerRtAtomic = true
				}
			}
			if hasCompilerRtAtomic {
				t.Errorf("target %s still links compiler-rt atomic.c", target)
			}
			if !hasVendored {
				t.Errorf("target %s misses the vendored atomic fallback", target)
			}
		})
	}
}

func TestWithPlatformSpecifiedFiles(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base")
	target := "riscv32-unknown-elf"
	inputFiles := []string{"file1.c", "file2.c"}

	result := withPlatformSpecifiedFiles(baseDir, target, inputFiles)

	// Should have input files + platform specific files
	if len(result) <= len(inputFiles) {
		t.Errorf("Expected more files than input, got %d", len(result))
	}

	// Check that input files are preserved
	for _, inputFile := range inputFiles {
		found := false
		for _, resultFile := range result {
			if resultFile == inputFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Input file %s not found in result", inputFile)
		}
	}
}

func TestGetCompilerRTConfig(t *testing.T) {
	baseDir := filepath.FromSlash("/test/base")
	target := "riscv32-unknown-elf"

	config := GetCompilerRTCompileConfig(baseDir, target)

	// Test groups configuration
	if len(config.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(config.Groups))
	} else {
		group := config.Groups[0]
		expectedOutput := "libclang_builtins-" + target + ".a"
		if group.OutputFileName != expectedOutput {
			t.Errorf("Expected output file %s, got %s", expectedOutput, group.OutputFileName)
		}

		// Check that files list contains platform-specific files
		if len(group.Files) == 0 {
			t.Error("Expected non-empty files list")
		}

		// Check that CFlags are set
		if len(group.CFlags) == 0 {
			t.Error("Expected non-empty CFlags")
		}

		// Check that CCFlags are set
		if len(group.CCFlags) == 0 {
			t.Error("Expected non-empty CCFlags")
		}
	}
}

func TestGetCompilerRTConfig_DifferentTargets(t *testing.T) {
	targets := []string{
		"riscv32-unknown-elf",
		"riscv64-unknown-elf",
		"arm-none-eabi",
		"avr-unknown-elf",
		"xtensa",
	}

	baseDir := filepath.FromSlash("/test/base")
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			config := GetCompilerRTCompileConfig(baseDir, target)

			if len(config.Groups) == 0 {
				t.Error("Should have at least one group")
			}

			// Check output filename contains target
			group := config.Groups[0]
			if !strings.Contains(group.OutputFileName, target) {
				t.Errorf("Output filename %s should contain target %s", group.OutputFileName, target)
			}
		})
	}
}

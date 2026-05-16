//go:build !llgo

package libc

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGetNewlibESP32Config_LibConfig(t *testing.T) {
	config := GetNewlibESP32Config()

	// Test basic configuration fields
	expectedName := "newlib-esp32"
	if config.Name != expectedName {
		t.Errorf("Expected Name '%s', got '%s'", expectedName, config.Name)
	}

	expectedVersion := "esp-4.3.0_20250211-patch7"
	if config.Version != expectedVersion {
		t.Errorf("Expected Version '%s', got '%s'", expectedVersion, config.Version)
	}

	expectedUrl := "https://github.com/goplus/newlib/archive/refs/tags/esp-4.3.0_20250211-patch7.tar.gz"
	if config.Url != expectedUrl {
		t.Errorf("Expected Url '%s', got '%s'", expectedUrl, config.Url)
	}

	expectedArchiveSrcDir := "newlib-esp-4.3.0_20250211-patch7"
	if config.ResourceSubDir != expectedArchiveSrcDir {
		t.Errorf("Expected ResourceSubDir '%s', got '%s'", expectedArchiveSrcDir, config.ResourceSubDir)
	}

	// Test String() method
	expectedString := "newlib-esp32-esp-4.3.0_20250211-patch7"
	if config.String() != expectedString {
		t.Errorf("Expected String() '%s', got '%s'", expectedString, config.String())
	}
}

func TestGetPicolibcConfig_LibConfig(t *testing.T) {
	config := GetPicolibcConfig()

	// Test basic configuration fields
	expectedName := "picolibc"
	if config.Name != expectedName {
		t.Errorf("Expected Name '%s', got '%s'", expectedName, config.Name)
	}

	expectedVersion := "v0.1.0"
	if config.Version != expectedVersion {
		t.Errorf("Expected Version '%s', got '%s'", expectedVersion, config.Version)
	}

	expectedUrl := "https://github.com/goplus/picolibc/archive/refs/heads/main.zip"
	if config.Url != expectedUrl {
		t.Errorf("Expected Url '%s', got '%s'", expectedUrl, config.Url)
	}

	expectedArchiveSrcDir := "picolibc-main"
	if config.ResourceSubDir != expectedArchiveSrcDir {
		t.Errorf("Expected ResourceSubDir '%s', got '%s'", expectedArchiveSrcDir, config.ResourceSubDir)
	}

	// Test String() method
	expectedString := "picolibc-v0.1.0"
	if config.String() != expectedString {
		t.Errorf("Expected String() '%s', got '%s'", expectedString, config.String())
	}
}

func TestGetPicolibcCompileConfig(t *testing.T) {
	baseDir := "/test/base"
	target := "test-target"

	config := GetPicolibcCompileConfig(baseDir, target)

	// Test LibcCFlags
	if len(config.ExportCFlags) != 2 {
		t.Errorf("Expected 2 LibcCFlags, got %d", len(config.ExportCFlags))
	} else {
		expected := "-I" + baseDir
		if config.ExportCFlags[0] != expected {
			t.Errorf("Expected LibcCFlags[0] to be '%s', got '%s'", expected, config.ExportCFlags[0])
		}

		expected = "-isystem" + filepath.Join(baseDir, "newlib", "libc", "include")
		if config.ExportCFlags[1] != expected {
			t.Errorf("Expected LibcCFlags[1] to be '%s', got '%s'", expected, config.ExportCFlags[1])
		}
	}

	// Test Groups configuration
	if len(config.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(config.Groups))
	} else {
		group := config.Groups[0]

		// Test output file name
		expectedOutput := "libc-" + target + ".a"
		if group.OutputFileName != expectedOutput {
			t.Errorf("Expected OutputFileName '%s', got '%s'", expectedOutput, group.OutputFileName)
		}

		// Test files list
		if len(group.Files) == 0 {
			t.Error("Expected non-empty files list")
		} else {
			// Check a few sample files
			sampleFiles := []string{
				filepath.Join(baseDir, "newlib", "libc", "string", "bcmp.c"),
				filepath.Join(baseDir, "newlib", "libc", "string", "memcpy.c"),
				filepath.Join(baseDir, "newlib", "libc", "string", "strlen.c"),
				filepath.Join(baseDir, "newlib", "libc", "stdlib", "nano-malloc.c"),
				filepath.Join(baseDir, "newlib", "libc", "tinystdio", "printf.c"),
			}

			for _, sample := range sampleFiles {
				found := false
				for _, file := range group.Files {
					if file == sample {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file '%s' not found in group files", sample)
				}
			}
		}

		// Test CFlags
		expectedCFlags := []string{
			"-D_COMPILING_NEWLIB",
			"-D_HAVE_ALIAS_ATTRIBUTE",
			"-DTINY_STDIO",
			"-DPOSIX_IO",
			"-DFORMAT_DEFAULT_INTEGER",
			"-D_IEEE_LIBM",
			"-D__OBSOLETE_MATH_FLOAT=1",
			"-D__OBSOLETE_MATH_DOUBLE=0",
			"-D_WANT_IO_C99_FORMATS",
			"-nostdlib",
			"-I" + baseDir,
			"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
			"-I" + filepath.Join(baseDir, "newlib", "libm", "common"),
			"-I" + filepath.Join(baseDir, "newlib", "libc", "locale"),
			"-I" + filepath.Join(baseDir, "newlib", "libc", "tinystdio"),
		}

		if len(group.CFlags) != len(expectedCFlags) {
			t.Errorf("Expected %d CFlags, got %d", len(expectedCFlags), len(group.CFlags))
		} else {
			for i, expected := range expectedCFlags {
				if group.CFlags[i] != expected {
					t.Errorf("CFlags[%d] mismatch. Expected '%s', got '%s'", i, expected, group.CFlags[i])
				}
			}
		}

		// Test LDFlags and CCFlags
		if len(group.LDFlags) == 0 {
			t.Error("Expected non-empty LDFlags")
		}
		if len(group.CCFlags) == 0 {
			t.Error("Expected non-empty CCFlags")
		}
	}
}

func TestGetPicolibcConfig_EdgeCases(t *testing.T) {
	t.Run("EmptyBaseDir", func(t *testing.T) {
		config := GetPicolibcCompileConfig("", "test-target")

		// Check that paths are constructed correctly even with empty baseDir
		expected := "-I"
		if config.ExportCFlags[0] != expected {
			t.Errorf("Expected LibcCFlags[0] to be '%s', got '%s'", expected, config.ExportCFlags[0])
		}

		expected = "-isystem" + filepath.Join("", "newlib", "libc", "include")
		if config.ExportCFlags[1] != expected {
			t.Errorf("Expected LibcCFlags[1] to be '%s', got '%s'", expected, config.ExportCFlags[1])
		}
	})

	t.Run("EmptyTarget", func(t *testing.T) {
		config := GetPicolibcCompileConfig("/test/base", "")

		// Check output file name formatting
		expectedOutput := "libc-.a"
		if config.Groups[0].OutputFileName != expectedOutput {
			t.Errorf("Expected OutputFileName '%s', got '%s'", expectedOutput, config.Groups[0].OutputFileName)
		}
	})
}

func TestPicolibcFileStructure(t *testing.T) {
	baseDir := "/test/base"
	target := "test-target"

	config := GetPicolibcCompileConfig(baseDir, target)
	group := config.Groups[0]

	// Test that all files have .c extension (no assembly files in picolibc config)
	for _, file := range group.Files {
		if !strings.HasSuffix(file, ".c") {
			t.Errorf("File '%s' does not have .c extension", file)
		}
	}

	// Test that files are from expected directories
	stringFiles := 0
	stdlibFiles := 0
	tinystdioFiles := 0

	for _, file := range group.Files {
		if strings.Contains(file, "/string/") {
			stringFiles++
		} else if strings.Contains(file, "/stdlib/") {
			stdlibFiles++
		} else if strings.Contains(file, "/tinystdio/") {
			tinystdioFiles++
		}
	}

	if stringFiles < 50 {
		t.Errorf("Expected at least 50 string files, got %d", stringFiles)
	}
	if stdlibFiles < 5 {
		t.Errorf("Expected at least 5 stdlib files, got %d", stdlibFiles)
	}
	if tinystdioFiles < 3 {
		t.Errorf("Expected at least 3 tinystdio files, got %d", tinystdioFiles)
	}

	// Test that all files have correct base directory
	for _, file := range group.Files {
		if !strings.HasPrefix(file, baseDir) {
			t.Errorf("File '%s' does not have expected base directory '%s'", file, baseDir)
		}
	}
}

func TestPicolibcCompilerFlags(t *testing.T) {
	baseDir := "/test/base"
	target := "test-target"

	config := GetPicolibcCompileConfig(baseDir, target)
	group := config.Groups[0]

	// Test that required preprocessor definitions are present
	requiredDefines := []string{
		"-D_COMPILING_NEWLIB",
		"-D_HAVE_ALIAS_ATTRIBUTE",
		"-DTINY_STDIO",
		"-DPOSIX_IO",
		"-DFORMAT_DEFAULT_INTEGER",
		"-D_IEEE_LIBM",
		"-D_WANT_IO_C99_FORMATS",
	}

	for _, define := range requiredDefines {
		found := false
		for _, flag := range group.CFlags {
			if flag == define {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required define '%s' not found in CFlags", define)
		}
	}

	// Test that required include paths are present
	requiredIncludes := []string{
		"-I" + baseDir,
		"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
		"-I" + filepath.Join(baseDir, "newlib", "libm", "common"),
		"-I" + filepath.Join(baseDir, "newlib", "libc", "locale"),
		"-I" + filepath.Join(baseDir, "newlib", "libc", "tinystdio"),
	}

	for _, include := range requiredIncludes {
		found := false
		for _, flag := range group.CFlags {
			if flag == include {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required include '%s' not found in CFlags", include)
		}
	}

	// Test that nostdlib is present
	found := false
	for _, flag := range group.CFlags {
		if flag == "-nostdlib" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Required flag '-nostdlib' not found in CFlags")
	}
}

func TestGetNewlibESP32ConfigRISCV(t *testing.T) {
	baseDir := "/test/base"
	target := "riscv32-unknown-elf"

	config := getNewlibESP32ConfigRISCV(baseDir, target)

	// Test LibcCFlags
	libcDir := filepath.Join(baseDir, "newlib", "libc")
	expectedCFlags := []string{
		"-isystem" + filepath.Join(libcDir, "include"),
		"-I" + filepath.Join(baseDir, "newlib"),
		"-I" + libcDir,
	}
	if len(config.ExportCFlags) != len(expectedCFlags) {
		t.Errorf("Expected %d LibcCFlags, got %d", len(expectedCFlags), len(config.ExportCFlags))
	} else {
		for i, expected := range expectedCFlags {
			if config.ExportCFlags[i] != expected {
				t.Errorf("ExportCFlags[%d] mismatch. Expected '%s', got '%s'", i, expected, config.ExportCFlags[i])
			}
		}
	}
	// Test Groups configuration
	if len(config.Groups) != 6 {
		t.Errorf("Expected 6 groups, got %d", len(config.Groups))
	} else {
		// Group 0: libsemihost
		group0 := config.Groups[0]
		expectedOutput0 := "libsemihost-" + target + ".a"
		if group0.OutputFileName != expectedOutput0 {
			t.Errorf("Group0 OutputFileName expected '%s', got '%s'", expectedOutput0, group0.OutputFileName)
		}

		// Check sample files in group0
		sampleFiles0 := []string{
			filepath.Join(baseDir, "libgloss", "riscv", "semihost-sys_exit.c"),
		}
		for _, sample := range sampleFiles0 {
			found := false
			for _, file := range group0.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group0 files", sample)
			}
		}

		// Group 1: libcrt0
		group1 := config.Groups[1]
		expectedOutput1 := "libcrt0-" + target + ".a"
		if group1.OutputFileName != expectedOutput1 {
			t.Errorf("Group1 OutputFileName expected '%s', got '%s'", expectedOutput1, group1.OutputFileName)
		}

		// Check sample files in group1
		sampleFiles1 := []string{
			filepath.Join(baseDir, "libgloss", "riscv", "esp", "esp_board.c"),
			filepath.Join(baseDir, "libgloss", "riscv", "esp", "crt1-board.S"),
		}
		for _, sample := range sampleFiles1 {
			found := false
			for _, file := range group1.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group1 files", sample)
			}
		}

		// Group 2: libgloss
		group2 := config.Groups[2]
		expectedOutput2 := "libgloss-" + target + ".a"
		if group2.OutputFileName != expectedOutput2 {
			t.Errorf("Group2 OutputFileName expected '%s', got '%s'", expectedOutput2, group2.OutputFileName)
		}

		// Check sample files in group2
		sampleFiles2 := []string{
			filepath.Join(baseDir, "libgloss", "libnosys", "close.c"),
			filepath.Join(baseDir, "libgloss", "libnosys", "sbrk.c"),
		}
		for _, sample := range sampleFiles2 {
			found := false
			for _, file := range group2.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group2 files", sample)
			}
		}

		// Group 3: libc
		group3 := config.Groups[3]
		expectedOutput3 := "libc-" + target + ".a"
		if group3.OutputFileName != expectedOutput3 {
			t.Errorf("Group3 OutputFileName expected '%s', got '%s'", expectedOutput3, group3.OutputFileName)
		}

		// Check sample files in group3
		sampleFiles3 := []string{
			filepath.Join(libcDir, "string", "memcpy.c"),
			filepath.Join(libcDir, "stdlib", "malloc.c"),
		}
		for _, sample := range sampleFiles3 {
			found := false
			for _, file := range group3.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group3 files", sample)
			}
		}

		// Test CFlags for group3 (libc)
		expectedCFlagsGroup3 := []string{
			"-DHAVE_CONFIG_H",
			"-D_LIBC",
			"-DHAVE_NANOSLEEP",
			"-D__NO_SYSCALLS__",
			// ... (other expected flags)
		}
		for _, expectedFlag := range expectedCFlagsGroup3 {
			found := false
			for _, flag := range group3.CFlags {
				if flag == expectedFlag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected flag '%s' not found in group3 CFlags", expectedFlag)
			}
		}

		// Test LDFlags and CCFlags
		if len(group0.LDFlags) == 0 {
			t.Error("Expected non-empty LDFlags in group0")
		}
		if len(group0.CCFlags) == 0 {
			t.Error("Expected non-empty CCFlags in group0")
		}

		// Group 4: libm (extra flags)
		group4 := config.Groups[4]
		expectedOutput4 := "libm-fbuiltin_fno_math_errno-" + target + ".a"
		if group4.OutputFileName != expectedOutput4 {
			t.Errorf("Group4 OutputFileName expected '%s', got '%s'", expectedOutput4, group4.OutputFileName)
		}
		sampleFiles4 := []string{
			filepath.Join(baseDir, "newlib", "libm", "common", "s_fpclassify.c"),
			filepath.Join(baseDir, "newlib", "libm", "common", "sf_fpclassify.c"),
		}
		for _, sample := range sampleFiles4 {
			found := false
			for _, file := range group4.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group4 files", sample)
			}
		}
		if !slices.Contains(group4.CFlags, "-fbuiltin") {
			t.Errorf("Expected group4 CFlags to contain -fbuiltin")
		}
		if !slices.Contains(group4.CFlags, "-fno-math-errno") {
			t.Errorf("Expected group4 CFlags to contain -fno-math-errno")
		}
		if slices.Contains(group4.LDFlags, "-u") || slices.Contains(group4.LDFlags, "_printf_float") {
			t.Errorf("Expected group4 LDFlags not to contain -u/_printf_float; this is configured by external target ldflags")
		}

		// Group 5: libm (default flags)
		group5 := config.Groups[5]
		expectedOutput5 := "libm-default-" + target + ".a"
		if group5.OutputFileName != expectedOutput5 {
			t.Errorf("Group5 OutputFileName expected '%s', got '%s'", expectedOutput5, group5.OutputFileName)
		}
		if slices.Contains(group5.CFlags, "-fbuiltin") {
			t.Errorf("Expected group5 CFlags not to contain -fbuiltin")
		}
		if slices.Contains(group5.CFlags, "-fno-math-errno") {
			t.Errorf("Expected group5 CFlags not to contain -fno-math-errno")
		}
		if slices.Contains(group5.LDFlags, "-u") || slices.Contains(group5.LDFlags, "_printf_float") {
			t.Errorf("Expected group5 LDFlags not to contain -u/_printf_float; this is configured by external target ldflags")
		}

		totalLibmFiles := len(group4.Files) + len(group5.Files)
		if totalLibmFiles != 410 {
			t.Errorf("Expected 410 libm files from riscv32-esp-elf build list, got %d", totalLibmFiles)
		}

		// libm list should follow the riscv32-esp-elf build selection and
		// must not include stale mathfp/legacy entries.
		allLibmFiles := append(append([]string{}, group4.Files...), group5.Files...)
		for _, disallow := range []string{
			filepath.Join(baseDir, "newlib", "libm", "mathfp", "s_sqrt.c"),
			filepath.Join(baseDir, "newlib", "libm", "common", "isgreater.c"),
			filepath.Join(baseDir, "newlib", "libm", "fenv", "fenv_stub.c"),
		} {
			for _, file := range allLibmFiles {
				if file == disallow {
					t.Errorf("Unexpected file '%s' found in group4 files", disallow)
				}
			}
		}
	}
}

func TestGetNewlibESP32ConfigXtensa(t *testing.T) {
	baseDir := "/test/base"
	target := "xtensa-esp32-elf"

	config := getNewlibESP32ConfigXtensa(baseDir, target)

	// Test LibcCFlags
	libcDir := filepath.Join(baseDir, "newlib", "libc")
	expectedCFlags := []string{
		"-I" + filepath.Join(libcDir, "include"),
		"-I" + filepath.Join(baseDir, "newlib"),
		"-I" + libcDir,
	}
	if len(config.ExportCFlags) != len(expectedCFlags) {
		t.Errorf("Expected %d LibcCFlags, got %d", len(expectedCFlags), len(config.ExportCFlags))
	} else {
		for i, expected := range expectedCFlags {
			if config.ExportCFlags[i] != expected {
				t.Errorf("ExportCFlags[%d] mismatch. Expected '%s', got '%s'", i, expected, config.ExportCFlags[i])
			}
		}
	}

	// Test Groups configuration
	if len(config.Groups) != 6 {
		t.Errorf("Expected 6 groups, got %d", len(config.Groups))
	} else {
		// Group 0: libcrt0
		group0 := config.Groups[0]
		expectedOutput0 := "libcrt0-" + target + ".a"
		if group0.OutputFileName != expectedOutput0 {
			t.Errorf("Group0 OutputFileName expected '%s', got '%s'", expectedOutput0, group0.OutputFileName)
		}

		// Check sample files in group0
		sampleFiles0 := []string{
			filepath.Join(baseDir, "libgloss", "xtensa", "clibrary_init.c"),
			filepath.Join(baseDir, "libgloss", "xtensa", "crt1-boards.S"),
		}
		for _, sample := range sampleFiles0 {
			found := false
			for _, file := range group0.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group0 files", sample)
			}
		}

		// Group 1: libgloss + libc
		group1 := config.Groups[1]
		expectedOutput1 := "libgloss-" + target + ".a"
		if group1.OutputFileName != expectedOutput1 {
			t.Errorf("Group1 OutputFileName expected '%s', got '%s'", expectedOutput1, group1.OutputFileName)
		}

		// Check sample files in group1
		sampleFiles1 := []string{
			filepath.Join(baseDir, "libgloss", "libnosys", "close.c"),
		}
		for _, sample := range sampleFiles1 {
			found := false
			for _, file := range group1.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group1 files", sample)
			}
		}

		// Test CFlags for group1
		expectedCFlagsGroup1 := []string{
			"-D__NO_SYSCALLS__",
			"-D_NO_GETUT",
			"-DHAVE_CONFIG_H",
			"-D_LIBC",
			// ... (other expected flags)
		}
		for _, expectedFlag := range expectedCFlagsGroup1 {
			found := false
			for _, flag := range group1.CFlags {
				if flag == expectedFlag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected flag '%s' not found in group1 CFlags", expectedFlag)
			}
		}

		// Group 3: libm sources with -fbuiltin -fno-math-errno.
		group3 := config.Groups[3]
		expectedOutput3 := "libm-fbuiltin_fno_math_errno-" + target + ".a"
		if group3.OutputFileName != expectedOutput3 {
			t.Errorf("Group3 OutputFileName expected '%s', got '%s'", expectedOutput3, group3.OutputFileName)
		}
		for _, sample := range []string{
			filepath.Join(baseDir, "newlib", "libm", "common", "s_fpclassify.c"),
			filepath.Join(baseDir, "newlib", "libm", "common", "sf_fpclassify.c"),
		} {
			found := false
			for _, file := range group3.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group3 files", sample)
			}
		}
		if !slices.Contains(group3.CFlags, "-fbuiltin") || !slices.Contains(group3.CFlags, "-fno-math-errno") {
			t.Errorf("Expected group3 CFlags to contain -fbuiltin and -fno-math-errno")
		}

		// Group 4: default libm sources.
		group4 := config.Groups[4]
		expectedOutput4 := "libm-default-" + target + ".a"
		if group4.OutputFileName != expectedOutput4 {
			t.Errorf("Group4 OutputFileName expected '%s', got '%s'", expectedOutput4, group4.OutputFileName)
		}
		for _, sample := range []string{
			filepath.Join(baseDir, "newlib", "libm", "complex", "cabs.c"),
			filepath.Join(baseDir, "newlib", "libm", "math", "e_acos.c"),
		} {
			found := false
			for _, file := range group4.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group4 files", sample)
			}
		}
		if slices.Contains(group4.CFlags, "-fbuiltin") || slices.Contains(group4.CFlags, "-fno-math-errno") || slices.Contains(group4.CFlags, "-D_LIBM") {
			t.Errorf("Expected group4 CFlags to not contain -fbuiltin/-fno-math-errno/-D_LIBM")
		}

		// Group 5: machine/xtensa libm sources built with -D_LIBM.
		group5 := config.Groups[5]
		expectedOutput5 := "libm-machine_xtensa_d_libm-" + target + ".a"
		if group5.OutputFileName != expectedOutput5 {
			t.Errorf("Group5 OutputFileName expected '%s', got '%s'", expectedOutput5, group5.OutputFileName)
		}
		for _, sample := range []string{
			filepath.Join(baseDir, "newlib", "libm", "machine", "xtensa", "ef_sqrt.c"),
			filepath.Join(baseDir, "newlib", "libm", "machine", "xtensa", "fegetenv.c"),
		} {
			found := false
			for _, file := range group5.Files {
				if file == sample {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected file '%s' not found in group5 files", sample)
			}
		}
		if !slices.Contains(group5.CFlags, "-D_LIBM") {
			t.Errorf("Expected group5 CFlags to contain -D_LIBM")
		}
		if !slices.Contains(group5.CFlags, "-I"+filepath.Join(baseDir, "newlib", "libc", "machine", "xtensa", "include")) {
			t.Errorf("Expected group5 CFlags to contain xtensa machine include path")
		}
		if !slices.Contains(group5.CFlags, "-I"+filepath.Join(baseDir, "newlib", "libc", "xtensa")) {
			t.Errorf("Expected group5 CFlags to contain xtensa sys include path")
		}

		totalLibmFiles := len(group3.Files) + len(group4.Files) + len(group5.Files)
		if totalLibmFiles != 380 {
			t.Errorf("Expected 380 xtensa libm files from command-log build list, got %d", totalLibmFiles)
		}

		// Test LDFlags and CCFlags
		if len(group0.LDFlags) == 0 {
			t.Error("Expected non-empty LDFlags in group0")
		}
		if len(group0.CCFlags) == 0 {
			t.Error("Expected non-empty CCFlags in group0")
		}
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyBaseDir_RISCV", func(t *testing.T) {
		config := getNewlibESP32ConfigRISCV("", "test-target")
		libcDir := filepath.Join("", "newlib", "libc")

		// Check that paths are constructed correctly
		expected := "-isystem" + filepath.Join(libcDir, "include")
		if config.ExportCFlags[0] != expected {
			t.Errorf("Expected LibcCFlags[0] to be '%s', got '%s'", expected, config.ExportCFlags[0])
		}
	})

	t.Run("EmptyTarget_RISCV", func(t *testing.T) {
		config := getNewlibESP32ConfigRISCV("/test/base", "")

		// Check output file name formatting (first group is libsemihost)
		expectedOutput := "libsemihost-.a"
		if config.Groups[0].OutputFileName != expectedOutput {
			t.Errorf("Expected OutputFileName '%s', got '%s'", expectedOutput, config.Groups[0].OutputFileName)
		}
	})

	t.Run("EmptyBaseDir_Xtensa", func(t *testing.T) {
		config := getNewlibESP32ConfigXtensa("", "test-target")
		libcDir := filepath.Join("", "newlib", "libc")

		// Check that paths are constructed correctly
		expected := "-I" + filepath.Join(libcDir, "include")
		if config.ExportCFlags[0] != expected {
			t.Errorf("Expected LibcCFlags[0] to be '%s', got '%s'", expected, config.ExportCFlags[0])
		}
	})

	t.Run("EmptyTarget_Xtensa", func(t *testing.T) {
		config := getNewlibESP32ConfigXtensa("/test/base", "")

		// Check output file name formatting
		expectedOutput := "libcrt0-.a"
		if config.Groups[0].OutputFileName != expectedOutput {
			t.Errorf("Expected OutputFileName '%s', got '%s'", expectedOutput, config.Groups[0].OutputFileName)
		}
	})
}

func TestGroupConfiguration(t *testing.T) {
	baseDir := "/test/base"
	target := "test-target"

	t.Run("RISCV_GroupCount", func(t *testing.T) {
		config := getNewlibESP32ConfigRISCV(baseDir, target)
		if len(config.Groups) != 6 {
			t.Errorf("Expected 6 groups for RISCV, got %d", len(config.Groups))
		}
	})

	t.Run("Xtensa_GroupCount", func(t *testing.T) {
		config := getNewlibESP32ConfigXtensa(baseDir, target)
		if len(config.Groups) != 6 {
			t.Errorf("Expected 6 groups for Xtensa, got %d", len(config.Groups))
		}
	})

	t.Run("RISCV_GroupNames", func(t *testing.T) {
		config := getNewlibESP32ConfigRISCV(baseDir, target)
		expectedNames := []string{
			"libsemihost-" + target + ".a",
			"libcrt0-" + target + ".a",
			"libgloss-" + target + ".a",
			"libc-" + target + ".a",
			"libm-fbuiltin_fno_math_errno-" + target + ".a",
			"libm-default-" + target + ".a",
		}

		for i, group := range config.Groups {
			if group.OutputFileName != expectedNames[i] {
				t.Errorf("Group %d expected name '%s', got '%s'", i, expectedNames[i], group.OutputFileName)
			}
		}
	})

	t.Run("Xtensa_GroupNames", func(t *testing.T) {
		config := getNewlibESP32ConfigXtensa(baseDir, target)
		expectedNames := []string{
			"libcrt0-" + target + ".a",
			"libgloss-" + target + ".a",
			"libc-" + target + ".a",
			"libm-fbuiltin_fno_math_errno-" + target + ".a",
			"libm-default-" + target + ".a",
			"libm-machine_xtensa_d_libm-" + target + ".a",
		}

		for i, group := range config.Groups {
			if group.OutputFileName != expectedNames[i] {
				t.Errorf("Group %d expected name '%s', got '%s'", i, expectedNames[i], group.OutputFileName)
			}
		}
	})
}

func TestCompilerFlags(t *testing.T) {
	baseDir := "/test/base"
	target := "test-target"

	t.Run("RISCV_CFlags", func(t *testing.T) {
		config := getNewlibESP32ConfigRISCV(baseDir, target)
		group := config.Groups[3] // libc group (index 3: libsemihost=0, libcrt0=1, libgloss=2, libc=3)

		requiredFlags := []string{
			"-DHAVE_CONFIG_H",
			"-D_LIBC",
			"-D__NO_SYSCALLS__",
			"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
		}

		for _, flag := range requiredFlags {
			found := false
			for _, cflag := range group.CFlags {
				if cflag == flag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Required flag '%s' not found in RISCV CFlags", flag)
			}
		}
	})

	t.Run("Xtensa_CFlags", func(t *testing.T) {
		config := getNewlibESP32ConfigXtensa(baseDir, target)
		group := config.Groups[1] // libgloss+libc group

		requiredFlags := []string{
			"-D__NO_SYSCALLS__",
			"-DHAVE_CONFIG_H",
			"-D_LIBC",
			"-isystem" + filepath.Join(baseDir, "newlib", "libc", "include"),
		}

		for _, flag := range requiredFlags {
			found := false
			for _, cflag := range group.CFlags {
				if cflag == flag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Required flag '%s' not found in Xtensa CFlags", flag)
			}
		}
	})

	t.Run("CommonFlags", func(t *testing.T) {
		configRISCV := getNewlibESP32ConfigRISCV(baseDir, target)
		configXtensa := getNewlibESP32ConfigXtensa(baseDir, target)

		// Test LDFlags
		expectedLDFlags := []string{
			"-nostdlib",
			"-ffunction-sections",
			"-fdata-sections",
		}

		for _, group := range configRISCV.Groups {
			for _, expected := range expectedLDFlags {
				found := false
				for _, flag := range group.LDFlags {
					if flag == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Required LDFlag '%s' not found in RISCV group", expected)
				}
			}
		}

		for _, group := range configXtensa.Groups {
			for _, expected := range expectedLDFlags {
				found := false
				for _, flag := range group.LDFlags {
					if flag == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Required LDFlag '%s' not found in Xtensa group", expected)
				}
			}
		}

		// Test CCFlags
		expectedCCFlags := []string{
			"-Oz",
			"-fno-builtin",
			"-ffreestanding",
		}

		for _, group := range configRISCV.Groups {
			for _, expected := range expectedCCFlags {
				found := false
				for _, flag := range group.CCFlags {
					if flag == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Required CCFlag '%s' not found in RISCV group", expected)
				}
			}
		}

		for _, group := range configXtensa.Groups {
			for _, expected := range expectedCCFlags {
				found := false
				for _, flag := range group.CCFlags {
					if flag == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Required CCFlag '%s' not found in Xtensa group", expected)
				}
			}
		}
	})
}

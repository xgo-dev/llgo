package mquickjs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/goplus/llgo/internal/crosscompile/compile"
)

// GetMquickjsConfig returns the configuration for downloading mquickjs
func GetMquickjsConfig() compile.LibConfig {
	return compile.LibConfig{
		Name:           "mquickjs",
		Version:        "v1.0.0",
		Url:            "https://github.com/luoliwoshang/mquickjs/archive/refs/tags/v1.0.0.tar.gz",
		ResourceSubDir: "mquickjs-1.0.0",
	}
}

// GenerateHeaders generates mquickjs_atom.h and mqjs_stdlib_llgo32.h by compiling
// the mqjs_stdlib tool for the HOST machine and running it with -m32 flag.
// It also generates stdlib_export.c to export js_stdlib symbol.
// This must be called after downloading the mquickjs source and before compiling.
func GenerateHeaders(baseDir string) error {
	atomHeader := filepath.Join(baseDir, "mquickjs_atom.h")
	stdlibHeader := filepath.Join(baseDir, "mqjs_stdlib_llgo32.h")
	stdlibExport := filepath.Join(baseDir, "stdlib_export.c")

	// Check if all generated files already exist
	if _, err := os.Stat(atomHeader); err == nil {
		if _, err := os.Stat(stdlibHeader); err == nil {
			if _, err := os.Stat(stdlibExport); err == nil {
				// All files exist, skip generation
				return nil
			}
		}
	}

	fmt.Fprintln(os.Stderr, "Generating mquickjs headers...")

	// Source files needed for the generator tool
	generatorSources := []string{
		filepath.Join(baseDir, "mqjs_stdlib.c"),
		filepath.Join(baseDir, "mquickjs_build.c"),
	}

	// Check if source files exist
	for _, src := range generatorSources {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("mquickjs generator source file not found: %s", src)
		}
	}

	// Output binary path (in the same directory)
	generatorBin := filepath.Join(baseDir, "mqjs_stdlib")

	// Check if generator binary already exists
	needCompile := true
	if _, err := os.Stat(generatorBin); err == nil {
		needCompile = false
	}

	if needCompile {
		// Compile the generator tool for HOST machine using system cc
		// We use "cc" which should work on most systems (maps to gcc/clang)
		compileArgs := []string{
			"-o", generatorBin,
			"-I" + baseDir,
			"-lm", // link math library
		}
		compileArgs = append(compileArgs, generatorSources...)

		fmt.Fprintf(os.Stderr, "  Compiling mqjs_stdlib tool...\n")
		cmd := exec.Command("cc", compileArgs...)
		cmd.Dir = baseDir
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to compile mqjs_stdlib generator: %w", err)
		}
	}

	// Generate mquickjs_atom.h with -m32 -a flags
	fmt.Fprintf(os.Stderr, "  Generating mquickjs_atom.h...\n")
	atomFile, err := os.Create(atomHeader)
	if err != nil {
		return fmt.Errorf("failed to create mquickjs_atom.h: %w", err)
	}
	defer atomFile.Close()

	cmd := exec.Command(generatorBin, "-m32", "-a")
	cmd.Dir = baseDir
	cmd.Stdout = atomFile
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate mquickjs_atom.h: %w", err)
	}

	// Generate mqjs_stdlib_llgo32.h with -m32 flag
	fmt.Fprintf(os.Stderr, "  Generating mqjs_stdlib_llgo32.h...\n")
	stdlibFile, err := os.Create(stdlibHeader)
	if err != nil {
		return fmt.Errorf("failed to create mqjs_stdlib_llgo32.h: %w", err)
	}
	defer stdlibFile.Close()

	cmd = exec.Command(generatorBin, "-m32")
	cmd.Dir = baseDir
	cmd.Stdout = stdlibFile
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate mqjs_stdlib_llgo32.h: %w", err)
	}

	// Generate stdlib_export.c to export js_stdlib symbol
	fmt.Fprintf(os.Stderr, "  Generating stdlib_export.c...\n")
	exportContent := `// Auto-generated file to export js_stdlib symbol for embedded use
#include <stddef.h>
#include <stdint.h>
#include <math.h>
#include "mquickjs.h"

// __fpclassifyd implementation for embedded platforms
// This function classifies a double-precision floating point number
// Required by mquickjs for isnan/isinf/isfinite macros
int __fpclassifyd(double a) {
    union { double d; uint64_t u; } u;
    u.d = a;
    uint32_t h = u.u >> 32;
    uint32_t l = (uint32_t)u.u;

    h &= 0x7fffffff;
    if (h >= 0x7ff00000) {
        if (h == 0x7ff00000 && l == 0)
            return FP_INFINITE;
        else
            return FP_NAN;
    } else if (h < 0x00100000) {
        if (h == 0 && l == 0)
            return FP_ZERO;
        else
            return FP_SUBNORMAL;
    } else {
        return FP_NORMAL;
    }
}

// Stub implementations for platform-specific functions
// Users should provide their own implementations using __attribute__((weak))
__attribute__((weak))
JSValue js_print(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_UNDEFINED;
}

__attribute__((weak))
JSValue js_gc(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_UNDEFINED;
}

__attribute__((weak))
JSValue js_load(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_UNDEFINED;
}

__attribute__((weak))
JSValue js_setTimeout(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_UNDEFINED;
}

__attribute__((weak))
JSValue js_clearTimeout(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_UNDEFINED;
}

__attribute__((weak))
JSValue js_date_now(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_NewInt32(ctx, 0); // Returns 0 as timestamp
}

__attribute__((weak))
JSValue js_performance_now(JSContext *ctx, JSValue *this_val, int argc, JSValue *argv) {
    (void)ctx; (void)this_val; (void)argc; (void)argv;
    return JS_NewFloat64(ctx, 0.0); // Returns 0 as timestamp
}

#include "mqjs_stdlib_llgo32.h"
`
	if err := os.WriteFile(stdlibExport, []byte(exportContent), 0644); err != nil {
		return fmt.Errorf("failed to create stdlib_export.c: %w", err)
	}

	fmt.Fprintln(os.Stderr, "  Headers generated successfully.")
	return nil
}

// mquickjs compile flags for embedded targets
var _mquickjsCCFlags = []string{
	"-fno-jump-tables", // Required for ESP32 IRAM/Flash architecture
	"-ffunction-sections",
	"-fdata-sections",
	"-fno-math-errno",
	"-fno-trapping-math",
	"-std=gnu17",
}

var _mquickjsLDFlags = []string{
	"-Wl,--gc-sections",
}

// GetMquickjsCompileConfig returns configuration for compiling mquickjs
func GetMquickjsCompileConfig(baseDir, target string) compile.CompileConfig {
	return compile.CompileConfig{
		ExportCFlags: []string{
			"-I" + baseDir,
		},
		Groups: []compile.CompileGroup{
			{
				OutputFileName: fmt.Sprintf("libmquickjs-%s.a", target),
				Files: []string{
					filepath.Join(baseDir, "mquickjs.c"),
					filepath.Join(baseDir, "cutils.c"),
					filepath.Join(baseDir, "dtoa.c"),
					filepath.Join(baseDir, "libm.c"),
					filepath.Join(baseDir, "stdlib_export.c"), // exports js_stdlib symbol
				},
				CFlags: []string{
					"-DCONFIG_VERSION=\"1.0.0\"",
					"-D_GNU_SOURCE",
					"-nostdlib",
					"-I" + baseDir,
				},
				CCFlags: _mquickjsCCFlags,
				LDFlags: _mquickjsLDFlags,
			},
		},
	}
}

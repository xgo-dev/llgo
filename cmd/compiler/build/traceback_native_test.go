//go:build darwin || linux || windows

package build

import (
	stdcontext "context"
	"debug/elf"
	"debug/macho"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNativeDebuggerRegistrySurvivesLTO(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix visibility/section retention; Windows uses dllexport")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal("clang is required for the native traceback test:", err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "registry.c")
	// No application code references the registry: only a debugger looking it
	// up by name can keep a reason to retain this symbol after whole-program LTO.
	if err := os.WriteFile(source, []byte("#include \"traceback_unix.c\"\nint main(void) { return 0; }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "registry")
	args := []string{"-std=c11", "-O2", "-flto", "-fvisibility=hidden",
		"-ffunction-sections", "-fdata-sections", "-Wall", "-Wextra", "-Werror",
		"-I../../../runtime/internal/stacktrace/_wrap", "-pthread", source, "-o", bin}
	if runtime.GOOS == "darwin" {
		args = append(args, "-Wl,-dead_strip")
	} else {
		args = append(args, "-fuse-ld=lld", "-Wl,--gc-sections")
	}
	if out, err := exec.Command(clang, args...).CombinedOutput(); err != nil {
		t.Fatalf("link registry under LTO and section GC: %v\n%s", err, out)
	}
	const name = "llgo_debugger_threads_v1"
	if runtime.GOOS == "darwin" {
		file, err := macho.Open(bin)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if file.Symtab != nil {
			for _, symbol := range file.Symtab.Syms {
				if symbol.Name == "_"+name && symbol.Sect != 0 && symbol.Type&0x11 == 1 {
					return // N_EXT, not N_PEXT: external and not private-external.
				}
			}
		}
	} else {
		file, err := elf.Open(bin)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		symbols, err := file.Symbols()
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range symbols {
			if symbol.Name == name && symbol.Section != elf.SHN_UNDEF &&
				elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL && elf.ST_VISIBILITY(symbol.Other) == elf.STV_DEFAULT {
				return
			}
		}
	}
	t.Fatalf("%s is missing or no longer externally visible after LTO and section GC", name)
}

func TestNativeTracebackCaptureFaultAndTimeout(t *testing.T) {
	testNativeTraceback(t, "main.c")
}

func TestNativeTracebackFaultBufferCapacity(t *testing.T) {
	if runtime.GOOS == "windows" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		t.Skip("dynamic Unix unwinder requires Darwin/Linux amd64/arm64")
	}
	testNativeTraceback(t, "capacity.c")
}

func testNativeTraceback(t *testing.T, source string) {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal("clang is required for the native traceback test:", err)
	}
	bin := filepath.Join(t.TempDir(), "traceback-native")
	args := []string{"-std=c11", "-O2", "-fno-omit-frame-pointer", "-Wall", "-Wextra", "-Werror", "-I../../../runtime/internal/stacktrace/_wrap"}
	if runtime.GOOS == "windows" {
		bin += ".exe"
		if target := os.Getenv("LLGO_WINDOWS_TARGET_TRIPLE"); target != "" {
			args = append(args, "--target="+target)
		}
		args = append(args, "-fuse-ld=lld", "testdata/tracebacknative/windows.c",
			"../../../runtime/internal/runtime/_wrap/setjmp_windows_amd64.c",
			"../../../runtime/internal/runtime/_wrap/setjmp_windows_arm64.c")
	} else {
		args = append(args, "-pthread", filepath.Join("testdata/tracebacknative", source))
		if runtime.GOOS == "linux" {
			args = append(args, "-ldl")
		}
	}
	cmd := exec.Command(clang, append(args, "-o", bin)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile native transport: %v\n%s", err, out)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 10*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, bin).CombinedOutput(); err != nil {
		t.Fatalf("native transport: %v\n%s", err, out)
	}
}

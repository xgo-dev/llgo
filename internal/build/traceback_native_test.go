//go:build !llgo && (darwin || linux || windows)

package build

import (
	stdcontext "context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

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
	args := []string{"-std=c11", "-O2", "-fno-omit-frame-pointer", "-Wall", "-Wextra", "-Werror", "-I../../runtime/internal/stacktrace/_wrap"}
	if runtime.GOOS == "windows" {
		bin += ".exe"
		if target := os.Getenv("LLGO_WINDOWS_TARGET_TRIPLE"); target != "" {
			args = append(args, "--target="+target)
		}
		args = append(args, "-fuse-ld=lld", "testdata/tracebacknative/windows.c",
			"../../runtime/internal/runtime/_wrap/setjmp_windows_amd64.c",
			"../../runtime/internal/runtime/_wrap/setjmp_windows_arm64.c")
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

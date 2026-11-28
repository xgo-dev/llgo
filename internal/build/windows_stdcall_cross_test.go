//go:build go1.27 && llgo_windows_cross

package build

import (
	"os"
	"testing"
)

// Explicit cross-owner role: the caller must activate its actual ARM64 Windows
// toolchain/sysroot first. Pure COFF and host tests need no Windows headers.
func TestWindowsStdCallARM64CompletePackage(t *testing.T) {
	// Do not export target CC/CXX while cmd/go builds this host test binary.
	// The explicit cross role activates them only for the LLGo invocation.
	cc, cxx := os.Getenv("LLGO_WINDOWS_CROSS_CC"), os.Getenv("LLGO_WINDOWS_CROSS_CXX")
	if cc == "" || cxx == "" {
		t.Fatal("the explicit Windows cross owner requires its real CC/CXX and sysroot")
	}
	t.Setenv("CC", cc)
	t.Setenv("CXX", cxx)
	windowsStdCallCompletePackage(t, "arm64")
}

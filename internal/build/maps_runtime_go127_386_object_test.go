//go:build go1.27 && ((windows && 386) || llgo_windows_386_cross)

package build

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// Compile the complete actual maps package, including the indexed masks/shifts
// in memhash_386.s. Cross compilation does not claim native AES execution.
func TestMapsRuntime386CompletePackage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "386" {
		cc, cxx := os.Getenv("LLGO_WINDOWS_CROSS_CC"), os.Getenv("LLGO_WINDOWS_CROSS_CXX")
		if cc == "" || cxx == "" {
			t.Fatal("the explicit Windows 386 cross role requires its real CC/CXX and sysroot")
		}
		// The test runner is built for the host before selecting these tools.
		t.Setenv("CC", cc)
		t.Setenv("CXX", cxx)
	}
	archive := mapsRuntimeCompletePackage(t, "windows", "386")
	_, readobj := windowsStdCallTools(t)
	headers := windowsStdCallCommand(t, readobj, "--file-headers", archive)
	formats, machines := 0, 0
	for _, line := range strings.Split(headers, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Format:") {
			formats++
			if line != "Format: COFF-i386" {
				t.Fatal("the complete maps archive contains a non-386 object", line)
			}
		}
		if strings.HasPrefix(line, "Machine:") {
			machines++
			if line != "Machine: IMAGE_FILE_MACHINE_I386 (0x14C)" {
				t.Fatal("the complete maps archive has the wrong machine", line)
			}
		}
	}
	if formats == 0 || machines != formats {
		t.Fatal("the complete maps archive lacks actual I386 COFF member headers", headers)
	}
}

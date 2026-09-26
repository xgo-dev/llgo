//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestShallowPanicTracebackBuildModes(t *testing.T) {
	for _, test := range []struct {
		name  string
		opt   optlevel.Level
		dwarf DWARFMode
	}{
		{name: "O0_DWARF", opt: optlevel.O0, dwarf: DWARFPreserve},
		{name: "O0_no_DWARF", opt: optlevel.O0, dwarf: DWARFOmit},
		{name: "O2_DWARF", opt: optlevel.O2, dwarf: DWARFPreserve},
		{name: "O2_no_DWARF", opt: optlevel.O2, dwarf: DWARFOmit},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "shallow-panic"
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			bin := filepath.Join(t.TempDir(), name)
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel = test.opt
			conf.LinkOptions.DWARF = test.dwarf
			conf.OutFile = bin
			if _, err := Do([]string{"./testdata/shallowpanic"}, conf); err != nil {
				t.Fatalf("build shallow panic fixture: %v", err)
			}
			output, err := exec.Command(bin).CombinedOutput()
			if err == nil {
				t.Fatalf("panic fixture unexpectedly succeeded:\n%s", output)
			}
			for _, want := range []string{
				"panic: shallow-panic",
				"goroutine 1 [running]:",
				"main.panicSite(",
				"main.main(",
			} {
				if !strings.Contains(string(output), want) {
					t.Fatalf("panic traceback is missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestDeepTestPanicTraceback(t *testing.T) {
	library := filepath.Join(t.TempDir(), "callback.so")
	args := []string{"-O2", "-fomit-frame-pointer", "-shared"}
	if runtime.GOOS == "windows" {
		library += ".dll"
		if target := os.Getenv("LLGO_WINDOWS_TARGET_TRIPLE"); target != "" {
			args = append(args, "--target="+target)
		}
	} else {
		args = append(args, "-fPIC")
	}
	args = append(args, "testdata/deeppaniclibrary/callback.c", "-o", library)
	if out, err := exec.Command("clang", args...).CombinedOutput(); err != nil {
		t.Fatalf("build native callback library: %v\n%s", err, out)
	}
	bin := filepath.Join(t.TempDir(), "deeppanic.test")
	conf := NewDefaultConf(ModeTest)
	conf.CompileOnly = true
	conf.OutFile = bin
	if _, err := Do([]string{"./testdata/deeppanic"}, conf); err != nil {
		t.Fatalf("build deep panic test fixture: %v", err)
	}
	for _, tc := range []struct {
		name string
		want []string
	}{
		{name: "TestMediumPanic", want: []string{"deeppanic.deepCall("}},
		{name: "TestDeepPanic", want: []string{"deeppanic.deepCall(", " frames elided..."}},
		{name: "TestVeryDeepPanic", want: []string{"deeppanic.deepCall(", " frames elided..."}},
		{name: "TestCCallbackPanic", want: []string{"deeppanic.panic_callback(", "deeppanic.callCGoCallback("}},
		{name: "TestDeepCCallbackPanic", want: []string{"deeppanic.panic_callback(", "deeppanic.callCGoCallback(", "deeppanic.deepCCallback(", " frames elided..."}},
		{name: "TestVeryDeepCCallbackPanic", want: []string{"deeppanic.panic_callback(", "deeppanic.callCGoCallback(", "deeppanic.deepCCallback(", " frames elided..."}},
		{name: "TestDeepInsideCCallbackPanic", want: []string{"deeppanic.deepCall(", "deeppanic.panic_callback(", "deeppanic.callCGoCallback(", " frames elided..."}},
		{name: "TestDeepFault", want: []string{"deeppanic.faultSite(", "deeppanic.deepFault(", " frames elided..."}},
		{name: "TestVeryDeepFault", want: []string{"deeppanic.faultSite(", "deeppanic.deepFault(", " frames elided..."}},
		{name: "TestDeepCallbackFault", want: []string{"deeppanic.faultSite(", "deeppanic.deepFault(", "deeppanic.callCGoCallback(", " frames elided..."}},
		{name: "TestForeignCallbackPanic", want: []string{"deeppanic.panic_callback(", " frames elided..."}},
		{name: "TestForeignCallbackFault", want: []string{"deeppanic.faultSite(", "deeppanic.panic_callback(", " frames elided..."}},
		{name: "TestDynamicCallbackPanic", want: []string{"deeppanic.panic_callback(", "deeppanic.callDynamicLibraryCallback(", " frames elided..."}},
		{name: "TestDynamicCallbackFault", want: []string{"deeppanic.faultSite(", "deeppanic.callDynamicLibraryCallback(", " frames elided..."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bin, "-test.run=^"+tc.name+"$")
			cmd.Env = append(os.Environ(), "GOTRACEBACK=single", "LLGO_TRACEBACK_CALLBACK_LIBRARY="+library)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("panic test fixture unexpectedly succeeded:\n%s", output)
			}
			common := []string{"panic:"}
			if !strings.Contains(tc.name, "Foreign") {
				common = append(common, "testing.tRunner(")
			}
			if !strings.Contains(tc.name, "Fault") {
				common = append(common, "panic: deep-panic", "deeppanic.panicSite(")
			}
			for _, want := range append(common, tc.want...) {
				if !strings.Contains(string(output), want) {
					t.Fatalf("panic traceback is missing %q:\n%s", want, output)
				}
			}
			switch tc.name {
			case "TestMediumPanic":
				got := strings.Count(string(output), "(...)\n")
				if got <= 50 || got >= 100 || strings.Contains(string(output), " frames elided...") {
					t.Fatalf("medium panic traceback printed %d frames:\n%s", got, output)
				}
			case "TestDeepFault", "TestVeryDeepFault", "TestDeepCallbackFault", "TestDeepPanic", "TestVeryDeepPanic", "TestDeepCCallbackPanic", "TestVeryDeepCCallbackPanic", "TestDeepInsideCCallbackPanic", "TestForeignCallbackPanic", "TestForeignCallbackFault", "TestDynamicCallbackPanic", "TestDynamicCallbackFault":
				if got := strings.Count(string(output), "(...)\n"); got != 100 {
					t.Fatalf("panic traceback printed %d frames, want 100:\n%s", got, output)
				}
			}
		})
	}
}

package build

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestWindowsAMD64StackReserveScope(t *testing.T) {
	for _, tc := range []struct {
		os, arch string
		mode     BuildMode
		abi      crosscompile.PlatformABI
		want     string
	}{
		{"windows", "amd64", BuildModeExe, crosscompile.PlatformABIMsvc, "-Wl,/stack:8388608"},
		{"windows", "amd64", BuildModeExe, crosscompile.PlatformABIGNU, "-Wl,--stack,8388608"},
		{"windows", "amd64", BuildModeExe, crosscompile.PlatformABIUnknown, ""},
		{"windows", "386", BuildModeExe, crosscompile.PlatformABIMsvc, ""},
		{"windows", "arm64", BuildModeExe, crosscompile.PlatformABIMsvc, ""},
		{"windows", "amd64", BuildModeCShared, crosscompile.PlatformABIMsvc, ""},
		{"windows", "amd64", BuildModeCArchive, crosscompile.PlatformABIGNU, ""},
		{"linux", "amd64", BuildModeExe, crosscompile.PlatformABIGNU, ""},
	} {
		conf := &Config{Goos: tc.os, Goarch: tc.arch, BuildMode: tc.mode}
		export := crosscompile.Export{Toolchain: crosscompile.NativeToolchain{ABI: tc.abi, ObjectFormat: crosscompile.ObjectFormatCOFF}, LDFLAGS: []string{"user-flags"}}
		applyWindowsAMD64StackReserve(conf, &export)
		want := []string{"user-flags"}
		if tc.want != "" {
			want = append([]string{tc.want}, want...)
		}
		if !slices.Equal(export.LDFLAGS, want) {
			t.Errorf("%+v: flags=%v, want %v", tc, export.LDFLAGS, want)
		}
	}
}

func TestWindowsAMD64StackReservePEHeader(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, triple, entry, override string
		abi                           crosscompile.PlatformABI
	}{
		{"msvc", "x86_64-pc-windows-msvc", "-Wl,/entry:entry,/subsystem:console", "-Wl,/stack:2097152", crosscompile.PlatformABIMsvc},
		{"mingw", "x86_64-w64-windows-gnu", "-Wl,-e,entry,--subsystem,console", "-Wl,--stack,2097152", crosscompile.PlatformABIGNU},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var defaultCommit uint64
			for _, variant := range []string{"baseline", "default", "override"} {
				dir := t.TempDir()
				source, binary := filepath.Join(dir, "entry.c"), filepath.Join(dir, "entry.exe")
				if err := os.WriteFile(source, []byte("int entry(void) { return 0; }\n"), 0600); err != nil {
					t.Fatal(err)
				}
				export := crosscompile.Export{Toolchain: crosscompile.NativeToolchain{ABI: tc.abi, ObjectFormat: crosscompile.ObjectFormatCOFF}, LDFLAGS: []string{tc.entry}}
				want := uint64(windowsAMD64StackReserve)
				if variant == "override" {
					export.LDFLAGS = append(export.LDFLAGS, tc.override)
					want = 2 << 20
				}
				if variant != "baseline" {
					applyWindowsAMD64StackReserve(&Config{Goos: "windows", Goarch: "amd64", BuildMode: BuildModeExe}, &export)
				}
				args := []string{"--target=" + tc.triple, "-fuse-ld=lld", "-nostdlib", source, "-o", binary}
				if out, err := exec.Command(clang, append(args, export.LDFLAGS...)...).CombinedOutput(); err != nil {
					t.Fatalf("link PE: %v\n%s", err, out)
				}
				file, err := pe.Open(binary)
				if err != nil {
					t.Fatal(err)
				}
				header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
				if !ok {
					file.Close()
					t.Fatal("missing PE32+ optional header")
				}
				reserve, commit := header.SizeOfStackReserve, header.SizeOfStackCommit
				file.Close()
				if variant == "baseline" {
					defaultCommit = commit
					continue
				}
				if reserve != want || commit != defaultCommit {
					t.Fatalf("%s: stack reserve/commit=%d/%d, want %d/%d", variant, reserve, commit, want, defaultCommit)
				}
			}
		})
	}
}

func TestMainThreadStack(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH != "amd64" {
		t.Skip("the Windows stack-reserve change is limited to amd64")
	}
	if testing.Short() {
		t.Skip("builds and runs the standalone stack/GC regression")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "mainstack"+defaultAppExt(conf))
	if _, err := Do([]string{"../../test/go/testdata/mainstack"}, conf); err != nil {
		t.Fatal(err)
	}
	if got := strings.ReplaceAll(runBinary(t, conf.OutFile), "\r\n", "\n"); got != "main-stack-ok\n" {
		t.Fatalf("stack/GC regression: %q", got)
	}
}

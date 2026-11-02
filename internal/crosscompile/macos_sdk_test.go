package crosscompile

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/clang"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func writeMacOSSDK(t *testing.T, path, version string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(path, "SDKSettings.json"), []byte(`{"Version":"`+version+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestSelectMacOSSDK(t *testing.T) {
	root := t.TempDir()
	old := writeMacOSSDK(t, filepath.Join(root, "MacOSX26.9.sdk"), "26.9")
	latest := writeMacOSSDK(t, filepath.Join(root, "MacOSX26.10.sdk"), "26.10")
	unsupported := writeMacOSSDK(t, filepath.Join(root, "MacOSX27.sdk"), "27.0")
	future := writeMacOSSDK(t, filepath.Join(root, "MacOSX28.sdk"), "28.0")
	unknown := writeMacOSSDK(t, filepath.Join(root, "MacOSX.sdk"), "")
	alias := filepath.Join(root, "alias.sdk")
	if err := os.Symlink(latest, alias); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, explicit, preferred string
		paths                     []string
		want, wantErr             string
	}{
		{name: "keep active older SDK", preferred: old, paths: []string{latest}, want: old},
		{name: "skip SDK 27 and sort numerically", preferred: unsupported, paths: []string{old, unsupported, alias, latest, future}, want: latest},
		{name: "missing active SDK", preferred: filepath.Join(root, "missing.sdk"), paths: []string{old}, want: old},
		{name: "xcrun unavailable", paths: []string{old}, want: old},
		{name: "explicit older SDK", explicit: old, preferred: unsupported, want: old},
		{name: "explicit SDK 27 fails", explicit: unsupported, paths: []string{latest}, wantErr: "SDKROOT"},
		{name: "explicit missing SDK fails", explicit: filepath.Join(root, "missing.sdk"), paths: []string{latest}, wantErr: "SDKROOT"},
		{name: "only unsupported SDKs", preferred: unsupported, paths: []string{unsupported, future, unknown}, wantErr: "install an SDK older than 27"},
		{name: "no SDKs", wantErr: "install an SDK older than 27"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			discovered := false
			sdk, err := selectMacOSSDK(tc.explicit, tc.preferred, func() []string { discovered = true; return tc.paths })
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				want, _ := filepath.EvalSymlinks(tc.want)
				if sdk.path != want {
					t.Fatalf("SDK = %q, want %q", sdk.path, want)
				}
			}
			if (tc.explicit != "" || tc.preferred == old) && discovered {
				t.Fatal("unexpected alternative SDK search")
			}
		})
	}
}

func TestReadMacOSSDKVersion(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name, version string
		ok            bool
	}{
		{"MacOSX.sdk", "26.5", true},
		{"MacOSX26.5.sdk", "", true},
		{"MacOSX26.5.sdk", "27.0", false}, // Metadata is authoritative.
		{"MacOSX27.sdk", "", false},
		{"MacOSX.sdk", "", false},
		{"MacOSX.sdk", "garbage", false},
		{"MacOSX.sdk", "26.bad", false},
		{"MacOSX.sdk", "+26", false},
		{"MacOSX.sdk", "0", false},
		{"MacOSX.sdk", "9999999999999999999999999", false},
	} {
		t.Run(tc.name+tc.version, func(t *testing.T) {
			path := writeMacOSSDK(t, filepath.Join(t.TempDir(), tc.name), tc.version)
			_, err := readMacOSSDK(path)
			if (err == nil) != tc.ok {
				t.Fatalf("readMacOSSDK error = %v, want success %v", err, tc.ok)
			}
		})
	}
	bad := writeMacOSSDK(t, filepath.Join(root, "MacOSX26.sdk"), "")
	if err := os.WriteFile(filepath.Join(bad, "SDKSettings.json"), []byte("invalid JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readMacOSSDK(bad); err == nil {
		t.Fatal("accepted malformed metadata")
	}
	if _, err := readMacOSSDK(filepath.Join(bad, "SDKSettings.json")); err == nil {
		t.Fatal("accepted a file as an SDK")
	}
}

func TestInstalledMacOSSDKs(t *testing.T) {
	root := t.TempDir()
	developer := filepath.Join(root, "Custom Xcode.app")
	home := filepath.Join(root, "home")
	paths := []string{
		filepath.Join(root, "active", "MacOSX26.sdk"),
		filepath.Join(developer, "Contents", "Developer", "Platforms", "MacOSX.platform", "Developer", "SDKs", "MacOSX26.sdk"),
		filepath.Join(home, "Applications", "Other.app", "Contents", "Developer", "Platforms", "MacOSX.platform", "Developer", "SDKs", "MacOSX26.sdk"),
	}
	for _, path := range paths {
		writeMacOSSDK(t, path, "26.0")
	}
	found := installedMacOSSDKs(filepath.Join(root, "active", "MacOSX27.sdk"), developer, home)
	for _, path := range paths {
		if !slices.Contains(found, path) {
			t.Errorf("SDK not discovered: %s", path)
		}
	}
	clt := filepath.Join(root, "CommandLineTools")
	path := writeMacOSSDK(t, filepath.Join(clt, "SDKs", "MacOSX26.sdk"), "26.0")
	if !slices.Contains(installedMacOSSDKs("", clt, ""), path) {
		t.Fatal("Command Line Tools SDK not discovered")
	}
}

func TestNativeMacOSSDKFlags(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native Darwin toolchain")
	}
	writeWasmTargetFixture(t, "", "")
	path := writeMacOSSDK(t, filepath.Join(t.TempDir(), "SDK with spaces.sdk"), "26.5")
	t.Setenv("SDKROOT", path)
	for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
		export, err := use(runtime.GOOS, runtime.GOARCH, false, optlevel.O2, mode, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, flags := range [][]string{export.CCFLAGS, export.LDFLAGS} {
			i := slices.Index(flags, "-isysroot")
			if i < 0 || i+1 == len(flags) {
				t.Fatalf("missing sysroot: %v", flags)
			}
			got, err := filepath.EvalSymlinks(flags[i+1])
			want, _ := filepath.EvalSymlinks(path)
			if err != nil || got != want {
				t.Fatalf("sysroot = %q, want %q: %v", got, want, err)
			}
		}
		if export.Toolchain.SDKVersion != "26.5" {
			t.Fatalf("SDK version = %q", export.Toolchain.SDKVersion)
		}
	}
	t.Setenv("SDKROOT", writeMacOSSDK(t, filepath.Join(t.TempDir(), "MacOSX27.sdk"), "27.0"))
	if _, err := use(runtime.GOOS, runtime.GOARCH, false, optlevel.O2, lto.Off, false); err == nil {
		t.Fatal("native build accepted SDK 27")
	}
}

func TestMacOSSDKVersionOrdering(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		writeMacOSSDK(t, filepath.Join(root, "short.sdk"), "26"),
		writeMacOSSDK(t, filepath.Join(root, "a.sdk"), "26.1"),
		writeMacOSSDK(t, filepath.Join(root, "b.sdk"), "26.1.0"),
		writeMacOSSDK(t, filepath.Join(root, "patch.sdk"), "26.1.1"),
	}
	for _, order := range [][]string{paths, {paths[3], paths[2], paths[1], paths[0]}} {
		sdk, err := selectMacOSSDK("", "", func() []string { return order })
		want, _ := filepath.EvalSymlinks(paths[3])
		if err != nil || sdk.path != want {
			t.Fatalf("SDK = %q, want %q: %v", sdk.path, want, err)
		}
	}
	// Numerically equal versions use a stable path tie-breaker, regardless of
	// the order in which Xcode and Command Line Tools SDKs are discovered.
	for _, order := range [][]string{{paths[1], paths[2]}, {paths[2], paths[1]}} {
		sdk, err := selectMacOSSDK("", "", func() []string { return order })
		want, _ := filepath.EvalSymlinks(paths[1])
		if err != nil || sdk.path != want {
			t.Fatalf("SDK = %q, want %q: %v", sdk.path, want, err)
		}
	}
}

func TestMacOSSDKUnreadableMetadata(t *testing.T) {
	path := writeMacOSSDK(t, filepath.Join(t.TempDir(), "MacOSX26.sdk"), "")
	// A directory in place of the metadata file must report the read error;
	// falling back to the directory name would hide a broken SDK installation.
	if err := os.Mkdir(filepath.Join(path, "SDKSettings.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := readMacOSSDK(path); err == nil {
		t.Fatal("accepted unreadable SDK metadata")
	}
}

func TestGetMacOSSDKEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("xcrun and xcode-select are Unix tools")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, script := range map[string]string{
		"xcrun":        "#!/bin/sh\n[ \"$*\" = '--sdk macosx --show-sdk-path' ] || exit 99\n[ -n \"$LLGO_TEST_ACTIVE_SDK\" ] || exit 1\nprintf '%s\\n' \"$LLGO_TEST_ACTIVE_SDK\"\n",
		"xcode-select": "#!/bin/sh\n[ \"$*\" = '-p' ] || exit 99\nprintf '%s\\n' \"$LLGO_TEST_DEVELOPER\"\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := writeMacOSSDK(t, filepath.Join(root, "active-old", "MacOSX26.sdk"), "26.0")
	unsupported := writeMacOSSDK(t, filepath.Join(root, "active-new", "MacOSX27.sdk"), "27.0")
	developer := filepath.Join(root, "Developer with spaces")
	// A high pre-27 minor version makes this fixture win over any SDKs on the
	// test machine while exercising the real discovery and selection path.
	fallback := writeMacOSSDK(t, filepath.Join(developer, "SDKs", "MacOSX26.999.sdk"), "26.999")
	for _, tc := range []struct{ name, explicit, active, developer, want string }{
		{"active older SDK", "", old, "", old},
		{"SDK 27 with DEVELOPER_DIR", "", unsupported, developer, fallback},
		{"SDK 27 with xcode-select", "", unsupported, "", fallback},
		{"xcrun fails", "", "", developer, fallback},
		{"explicit SDK without xcrun", old, "", "", old},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SDKROOT", tc.explicit)
			t.Setenv("DEVELOPER_DIR", tc.developer)
			t.Setenv("LLGO_TEST_ACTIVE_SDK", tc.active)
			t.Setenv("LLGO_TEST_DEVELOPER", developer)
			sdk, err := getMacOSSDK()
			want, _ := filepath.EvalSymlinks(tc.want)
			if err != nil || sdk.path != want {
				t.Fatalf("SDK = %q, want %q: %v", sdk.path, want, err)
			}
		})
	}
}

func TestNativeMacOSSDKOverridesClangConfig(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("native Darwin toolchain")
	}
	compiler, err := exec.LookPath("clang++")
	if err != nil {
		t.Skip("clang++ is unavailable")
	}
	linker, err := exec.LookPath("ld64.lld")
	if err != nil {
		t.Skip("ld64.lld is unavailable")
	}
	sdk, err := getMacOSSDK()
	if err != nil {
		t.Fatal(err)
	}
	writeWasmTargetFixture(t, "", "")
	t.Setenv("SDKROOT", sdk.path)
	for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS"} {
		t.Setenv(name, "")
	}
	root := t.TempDir()
	badSDK := writeMacOSSDK(t, filepath.Join(root, "SDK with spaces", "MacOSX27.sdk"), "27.0")
	for name, contents := range map[string]string{
		"usr/include/stdio.h":   "#error compilation selected the wrong SDK\n",
		"usr/lib/libSystem.tbd": "invalid SDK 27 stub: linking selected the wrong SDK\n",
	} {
		path := filepath.Join(badSDK, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(root, "probe.c")
	if err := os.WriteFile(source, []byte("#include <stdio.h>\nint main(void) { return puts(\"selected SDK\") < 0; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, option := range []string{"--sysroot=", "-isysroot "} {
		for _, mode := range []lto.Mode{lto.Off, lto.Thin, lto.Full} {
			t.Run(option+mode.String(), func(t *testing.T) {
				dir := t.TempDir()
				configPath := filepath.Join(dir, "clang.cfg")
				if err := os.WriteFile(configPath, []byte(option+strconv.Quote(badSDK)+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				export, err := use(runtime.GOOS, runtime.GOARCH, false, optlevel.O2, mode, false)
				if err != nil {
					t.Fatal(err)
				}
				config := clang.NewConfig(compiler, export.CCFLAGS, export.CFLAGS, export.LDFLAGS, "")
				config.CCArgs = []string{"--no-default-config", "--config=" + configPath}
				config.LDFLAGS = append(config.LDFLAGS, "--ld-path="+linker)
				var output bytes.Buffer
				cc, ld := clang.NewCompiler(config), clang.NewLinker(config)
				cc.Stdout, cc.Stderr = &output, &output
				ld.Stdout, ld.Stderr = &output, &output
				object, binary := filepath.Join(dir, "probe.o"), filepath.Join(dir, "probe")
				if err := cc.Compile("-x", "c", "-c", source, "-o", object); err != nil {
					t.Fatalf("compile with configured %s: %v\n%s", option, err, output.String())
				}
				output.Reset()
				if err := ld.Link(object, "-o", binary); err != nil {
					t.Fatalf("link with configured %s: %v\n%s", option, err, output.String())
				}
				if out, err := exec.Command(binary).CombinedOutput(); err != nil || string(out) != "selected SDK\n" {
					t.Fatalf("run: %v\n%s", err, out)
				}
			})
		}
	}
}

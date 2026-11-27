package crosscompile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestResolveWindowsToolchainDefaultsToMSVC(t *testing.T) {
	probe := fixedToolProbe(t, map[string]toolIdentity{
		commandKey("clang"): {target: "x86_64-pc-windows-msvc", version: "clang version 19.1.7"},
		commandKey("clang++", "--target=x86_64-pc-windows-msvc", "-fms-runtime-lib=dll"): {target: "x86_64-pc-windows-msvc", version: "clang version 19.1.7"},
	})
	input := NativeToolchainInput{
		ExternalFlags: []string{"/incremental:no"},
		Environ: []string{
			"WindowsSDKVersion=10.0.26100.0\\",
			"UCRTVersion=10.0.26100.0",
			"VCToolsVersion=14.44.35207\\",
		},
	}
	export, err := resolveWindowsToolchain("amd64", input, probe)
	if err != nil {
		t.Fatal(err)
	}
	wantToolchain := NativeToolchain{
		ABI:            PlatformABIMsvc,
		ObjectFormat:   ObjectFormatCOFF,
		Driver:         DriverFlavorClangGNU,
		Linker:         LinkerFlavorCOFFLLD,
		TargetTriple:   "x86_64-pc-windows-msvc",
		CRT:            CRTFlavorUCRT,
		CXXRuntime:     CXXRuntimeMSVC,
		SDKVersion:     "10.0.26100.0",
		CRTVersion:     "10.0.26100.0",
		ToolsetVersion: "14.44.35207",
	}
	if export.Toolchain != wantToolchain {
		t.Fatalf("Toolchain = %+v, want %+v", export.Toolchain, wantToolchain)
	}
	if export.CC != "clang" || !slices.Equal(export.CCArgs, []string{"--target=x86_64-pc-windows-msvc", "-fms-runtime-lib=dll"}) {
		t.Fatalf("CC command = %q %q", export.CC, export.CCArgs)
	}
	if export.CXX != "clang++" || !slices.Equal(export.CXXArgs, []string{"--target=x86_64-pc-windows-msvc", "-fms-runtime-lib=dll"}) {
		t.Fatalf("CXX command = %q %q", export.CXX, export.CXXArgs)
	}
	if export.CCIdentity != "clang version 19.1.7" || export.CXXIdentity != "clang version 19.1.7" {
		t.Fatalf("compiler identities = %q, %q", export.CCIdentity, export.CXXIdentity)
	}
	if !slices.Equal(export.LDFLAGS, input.ExternalFlags) {
		t.Fatalf("LDFLAGS = %q, want %q", export.LDFLAGS, input.ExternalFlags)
	}
	if len(export.BuildTags) != 0 {
		t.Fatalf("MSVC BuildTags = %q, want none", export.BuildTags)
	}
}

func TestResolveWindowsToolchainDefaultsToGNUProfile(t *testing.T) {
	probe := fixedToolProbe(t, map[string]toolIdentity{
		commandKey("clang"): {target: "x86_64-w64-mingw32", version: "clang version 19.1.7"},
		commandKey("clang++", "--target=x86_64-w64-windows-gnu"): {target: "x86_64-w64-windows-gnu", version: "clang version 19.1.7"},
	})
	export, err := resolveWindowsToolchain("amd64", NativeToolchainInput{
		Environ: []string{
			"MSYSTEM=CLANG64",
			"WindowsSDKVersion=10.0.26100.0\\",
			"UCRTVersion=10.0.26100.0",
			"VCToolsVersion=14.44.35207\\",
		},
	}, probe)
	if err != nil {
		t.Fatal(err)
	}
	if export.Toolchain.ABI != PlatformABIGNU || export.Toolchain.TargetTriple != "x86_64-w64-windows-gnu" {
		t.Fatalf("GNU toolchain = %+v", export.Toolchain)
	}
	if export.Toolchain.SDKVersion != "" || export.Toolchain.CRTVersion != "" || export.Toolchain.ToolsetVersion != "" {
		t.Fatalf("GNU profile inherited the host MSVC environment: %+v", export.Toolchain)
	}
	if export.CC != "clang" || !slices.Equal(export.CCArgs, []string{"--target=x86_64-w64-windows-gnu"}) {
		t.Fatalf("CC command = %q %q", export.CC, export.CCArgs)
	}
	if export.CXX != "clang++" || !slices.Equal(export.CXXArgs, []string{"--target=x86_64-w64-windows-gnu"}) {
		t.Fatalf("CXX command = %q %q", export.CXX, export.CXXArgs)
	}
	if !slices.Equal(export.BuildTags, []string{windowsGNUABIBuildTag}) {
		t.Fatalf("GNU BuildTags = %q, want %q", export.BuildTags, windowsGNUABIBuildTag)
	}
}

func TestResolveWindowsToolchainImplicitlyCrossesWindowsArchitecture(t *testing.T) {
	for _, test := range []struct {
		goarch string
		triple string
	}{
		{goarch: "386", triple: "i686-pc-windows-msvc"},
		{goarch: "arm64", triple: "aarch64-pc-windows-msvc"},
	} {
		t.Run(test.goarch, func(t *testing.T) {
			probe := fixedToolProbe(t, map[string]toolIdentity{
				commandKey("clang"): {target: "x86_64-pc-windows-msvc", version: "clang version 19.1.7"},
				commandKey("clang++", "--target="+test.triple, "-fms-runtime-lib=dll"): {
					target: test.triple, version: "clang version 19.1.7",
				},
			})
			export, err := resolveWindowsToolchain(test.goarch, NativeToolchainInput{}, probe)
			if err != nil {
				t.Fatal(err)
			}
			if got := export.Toolchain.TargetTriple; got != test.triple {
				t.Fatalf("TargetTriple = %q, want %q", got, test.triple)
			}
			if !slices.Equal(export.CCArgs, []string{"--target=" + test.triple, "-fms-runtime-lib=dll"}) {
				t.Fatalf("CCArgs = %q", export.CCArgs)
			}
		})
	}
}

func TestWindowsEnvironmentValueUsesLastCaseInsensitiveEntry(t *testing.T) {
	environ := []string{"UCRTVersion=old", "other=value", "ucrtversion=10.0.26100.0\\"}
	if got, want := windowsEnvironmentValue(environ, "UCRTVersion"), "10.0.26100.0"; got != want {
		t.Fatalf("windowsEnvironmentValue() = %q, want %q", got, want)
	}
}

func TestResolveWindowsToolchainExplicitGNUProfile(t *testing.T) {
	cc := []string{"wrapper", "clang", "--target=x86_64-w64-windows-gnu"}
	cxx := []string{"clang++", "--target=x86_64-w64-windows-gnu"}
	extld := []string{"clang", "--target=x86_64-unknown-windows-gnu"}
	probe := fixedToolProbe(t, map[string]toolIdentity{
		commandKey(cc...):    {target: "x86_64-w64-mingw32", version: "clang version 19.1.7"},
		commandKey(cxx...):   {target: "x86_64-w64-windows-gnu", version: "clang version 19.1.7"},
		commandKey(extld...): {target: "x86_64-unknown-windows-gnu", version: "clang version 19.1.7"},
	})
	export, err := resolveWindowsToolchain("amd64", NativeToolchainInput{
		CC:             cc,
		ExternalLinker: extld,
		Environ: []string{
			"WindowsSDKVersion=10.0.26100.0\\",
			"UCRTVersion=10.0.26100.0",
			"VCToolsVersion=14.44.35207\\",
		},
	}, probe)
	if err != nil {
		t.Fatal(err)
	}
	if export.Toolchain.ABI != PlatformABIGNU || export.Toolchain.TargetTriple != "x86_64-w64-windows-gnu" {
		t.Fatalf("GNU toolchain = %+v", export.Toolchain)
	}
	if export.Toolchain.CRT != CRTFlavorUnknown || export.Toolchain.CXXRuntime != CXXRuntimeUnknown {
		t.Fatalf("ambiguous GNU runtime was guessed: %+v", export.Toolchain)
	}
	if export.Toolchain.SDKVersion != "" || export.Toolchain.CRTVersion != "" || export.Toolchain.ToolsetVersion != "" {
		t.Fatalf("GNU profile inherited the host MSVC environment: %+v", export.Toolchain)
	}
	if export.CXX != cxx[0] || !slices.Equal(export.CXXArgs, cxx[1:]) {
		t.Fatalf("derived CXX = %q %q, want %q", export.CXX, export.CXXArgs, cxx)
	}
	if export.Linker != extld[0] || !slices.Equal(export.LinkerArgs, extld[1:]) {
		t.Fatalf("external linker = %q %q, want %q", export.Linker, export.LinkerArgs, extld)
	}
	if !slices.Equal(export.BuildTags, []string{windowsGNUABIBuildTag}) {
		t.Fatalf("GNU BuildTags = %q, want %q", export.BuildTags, windowsGNUABIBuildTag)
	}
}

func TestResolveWindowsToolchainRejectsIncompatibleInputs(t *testing.T) {
	clang := toolIdentity{target: "x86_64-pc-windows-msvc", version: "clang version 19.1.7"}
	for _, test := range []struct {
		name    string
		goarch  string
		input   NativeToolchainInput
		ids     map[string]toolIdentity
		probe   toolProbe
		wantErr string
	}{
		{
			name:    "unsupported GOARCH",
			goarch:  "arm",
			wantErr: `unsupported Windows GOARCH "arm"`,
		},
		{
			name:    "wrong architecture",
			goarch:  "arm64",
			input:   NativeToolchainInput{CC: []string{"clang"}},
			ids:     map[string]toolIdentity{commandKey("clang"): clang},
			wantErr: "want GOARCH=arm64",
		},
		{
			name:   "implicit compiler unsupported architecture",
			goarch: "amd64",
			ids: map[string]toolIdentity{
				commandKey("clang"): {target: "mips64-pc-windows-msvc", version: "clang version 19.1.7"},
			},
			wantErr: "unsupported Windows architecture",
		},
		{
			name:   "Cygwin ABI",
			goarch: "amd64",
			input:  NativeToolchainInput{CC: []string{"clang"}},
			ids: map[string]toolIdentity{
				commandKey("clang"): {target: "x86_64-pc-cygwin", version: "clang version 19.1.7"},
			},
			wantErr: "unsupported MSYS/Cygwin",
		},
		{
			name:    "clang-cl",
			goarch:  "amd64",
			input:   NativeToolchainInput{CC: []string{"clang-cl"}},
			ids:     map[string]toolIdentity{commandKey("clang-cl"): clang},
			wantErr: "clang-cl flag syntax",
		},
		{
			name:   "non-Clang driver",
			goarch: "amd64",
			input:  NativeToolchainInput{CC: []string{"gcc"}},
			ids: map[string]toolIdentity{
				commandKey("gcc"): {target: "x86_64-pc-windows-msvc", version: "gcc version 15"},
			},
			wantErr: "unsupported driver",
		},
		{
			name:   "CXX ABI mismatch",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:  []string{"clang"},
				CXX: []string{"clang++"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):   clang,
				commandKey("clang++"): {target: "x86_64-w64-windows-gnu", version: "clang version 19.1.7"},
			},
			wantErr: "CC and CXX are incompatible",
		},
		{
			name:   "CXX probe error",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:  []string{"clang"},
				CXX: []string{"missing-cxx"},
			},
			ids:     map[string]toolIdentity{commandKey("clang"): clang},
			wantErr: "probe CXX",
		},
		{
			name:   "CXX non-Windows target",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:  []string{"clang"},
				CXX: []string{"clang++"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):   clang,
				commandKey("clang++"): {target: "x86_64-unknown-linux-gnu", version: "clang version 19.1.7"},
			},
			wantErr: "not an explicit Windows",
		},
		{
			name:   "CXX unsupported driver",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:  []string{"clang"},
				CXX: []string{"g++"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"): clang,
				commandKey("g++"):   {target: "x86_64-pc-windows-msvc", version: "gcc version 15"},
			},
			wantErr: "CXX \"g++\" uses an unsupported driver",
		},
		{
			name:   "external linker mismatch",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:             []string{"clang"},
				CXX:            []string{"clang++"},
				ExternalLinker: []string{"other-clang"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):       clang,
				commandKey("clang++"):     clang,
				commandKey("other-clang"): {target: "x86_64-w64-windows-gnu", version: "clang version 19.1.7"},
			},
			wantErr: "-extld is incompatible with CC",
		},
		{
			name:   "external linker probe error",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:             []string{"clang"},
				CXX:            []string{"clang++"},
				ExternalLinker: []string{"missing-linker"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):   clang,
				commandKey("clang++"): clang,
			},
			wantErr: "probe -extld",
		},
		{
			name:   "external linker non-Windows target",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:             []string{"clang"},
				CXX:            []string{"clang++"},
				ExternalLinker: []string{"other-clang"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):       clang,
				commandKey("clang++"):     clang,
				commandKey("other-clang"): {target: "x86_64-unknown-linux-gnu", version: "clang version 19.1.7"},
			},
			wantErr: "not an explicit Windows",
		},
		{
			name:   "external linker unsupported driver",
			goarch: "amd64",
			input: NativeToolchainInput{
				CC:             []string{"clang"},
				CXX:            []string{"clang++"},
				ExternalLinker: []string{"gcc"},
			},
			ids: map[string]toolIdentity{
				commandKey("clang"):   clang,
				commandKey("clang++"): clang,
				commandKey("gcc"):     {target: "x86_64-pc-windows-msvc", version: "gcc version 15"},
			},
			wantErr: "-extld \"gcc\" uses an unsupported driver",
		},
		{
			name:   "probe error",
			goarch: "amd64",
			input:  NativeToolchainInput{CC: []string{"missing"}},
			probe: func([]string, NativeToolchainInput) (toolIdentity, error) {
				return toolIdentity{}, errors.New("not found")
			},
			wantErr: "probe CC",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			probe := test.probe
			if probe == nil {
				probe = fixedToolProbe(t, test.ids)
			}
			_, err := resolveWindowsToolchain(test.goarch, test.input, probe)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestWindowsTargetTriple(t *testing.T) {
	for _, test := range []struct {
		goarch string
		abi    PlatformABI
		want   string
	}{
		{"386", PlatformABIMsvc, "i686-pc-windows-msvc"},
		{"amd64", PlatformABIMsvc, "x86_64-pc-windows-msvc"},
		{"arm64", PlatformABIMsvc, "aarch64-pc-windows-msvc"},
		{"386", PlatformABIGNU, "i686-w64-windows-gnu"},
		{"amd64", PlatformABIGNU, "x86_64-w64-windows-gnu"},
		{"arm64", PlatformABIGNU, "aarch64-w64-windows-gnu"},
	} {
		got, err := windowsTargetTriple(test.goarch, test.abi)
		if err != nil || got != test.want {
			t.Errorf("windowsTargetTriple(%q, %q) = %q, %v; want %q", test.goarch, test.abi, got, err, test.want)
		}
	}
	if _, err := windowsTargetTriple("amd64", PlatformABIUnknown); err == nil {
		t.Fatal("windowsTargetTriple accepted an unknown ABI")
	}
}

func TestWindowsToolchainForTripleErrorsAndAliases(t *testing.T) {
	for _, test := range []struct {
		name    string
		goarch  string
		target  string
		wantABI PlatformABI
		wantErr string
	}{
		{name: "empty", goarch: "amd64", wantErr: "empty target triple"},
		{name: "non-Windows", goarch: "amd64", target: "x86_64-unknown-linux-gnu", wantErr: "not an explicit Windows"},
		{name: "unsupported arch", goarch: "amd64", target: "mips64-pc-windows-msvc", wantErr: "unsupported Windows architecture"},
		{name: "i386 alias", goarch: "386", target: "i386-pc-windows-msvc", wantABI: PlatformABIMsvc},
		{name: "amd64 alias", goarch: "amd64", target: "amd64-pc-windows-msvc", wantABI: PlatformABIMsvc},
		{name: "arm64 alias", goarch: "arm64", target: "arm64-w64-windows-gnu", wantABI: PlatformABIGNU},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := windowsToolchainForTriple(test.goarch, test.target)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.ABI != test.wantABI {
				t.Fatalf("ABI = %q, want %q", got.ABI, test.wantABI)
			}
		})
	}
}

func TestProbeNativeTool(t *testing.T) {
	if os.Getenv("LLGO_NATIVE_TOOL_PROBE_HELPER") == "1" {
		if got, want := canonicalTestPath(mustGetwd()), canonicalTestPath(os.Getenv("LLGO_NATIVE_TOOL_PROBE_DIR")); got != want {
			os.Exit(2)
		}
		arg := os.Args[len(os.Args)-1]
		switch arg {
		case "-dumpmachine":
			if slices.Contains(os.Args, "--fail-dumpmachine") || slices.Contains(os.Args, "--fail-all-targets") {
				os.Exit(3)
			}
			_, _ = os.Stdout.WriteString("x86_64-pc-windows-msvc\r\n")
		case "--print-target-triple":
			if slices.Contains(os.Args, "--fail-all-targets") {
				os.Exit(5)
			}
			_, _ = os.Stdout.WriteString("x86_64-pc-windows-msvc\n")
		case "--version":
			if slices.Contains(os.Args, "--fail-version") {
				os.Exit(6)
			}
			_, _ = os.Stdout.WriteString("clang version 19.1.7\nsecond line\n")
		default:
			os.Exit(4)
		}
		os.Exit(0)
	}

	dir := t.TempDir()
	environ := append(slices.Clone(os.Environ()),
		"LLGO_NATIVE_TOOL_PROBE_HELPER=1",
		"LLGO_NATIVE_TOOL_PROBE_DIR="+dir,
	)
	for _, command := range [][]string{
		{os.Args[0], "-test.run=^TestProbeNativeTool$", "--"},
		{os.Args[0], "-test.run=^TestProbeNativeTool$", "--", "--fail-dumpmachine"},
	} {
		identity, err := probeNativeTool(command, NativeToolchainInput{Dir: dir, Environ: environ})
		if err != nil {
			t.Fatal(err)
		}
		want := toolIdentity{target: "x86_64-pc-windows-msvc", version: "clang version 19.1.7"}
		if identity != want {
			t.Fatalf("identity = %+v, want %+v", identity, want)
		}
	}
	for _, test := range []struct {
		name    string
		marker  string
		wantErr string
	}{
		{name: "target", marker: "--fail-all-targets", wantErr: "read target triple"},
		{name: "version", marker: "--fail-version", wantErr: "read compiler identity"},
	} {
		t.Run(test.name+" error", func(t *testing.T) {
			command := []string{os.Args[0], "-test.run=^TestProbeNativeTool$", "--", test.marker}
			_, err := probeNativeTool(command, NativeToolchainInput{Dir: dir, Environ: environ})
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func fixedToolProbe(t *testing.T, identities map[string]toolIdentity) toolProbe {
	t.Helper()
	return func(command []string, _ NativeToolchainInput) (toolIdentity, error) {
		identity, ok := identities[commandKey(command...)]
		if !ok {
			return toolIdentity{}, errors.New("unexpected command: " + strings.Join(command, " "))
		}
		return identity, nil
	}
}

func commandKey(command ...string) string {
	return strings.Join(command, "\x00")
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}

func canonicalTestPath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		panic(err)
	}
	return filepath.Clean(resolved)
}

func TestCommandOrDefaultDoesNotAliasInput(t *testing.T) {
	input := []string{"clang", "--target=x86_64-pc-windows-msvc"}
	got := commandOrDefault(input, "unused")
	got[0] = "changed"
	if reflect.DeepEqual(got, input) || input[0] != "clang" {
		t.Fatalf("commandOrDefault aliased input: got %q input %q", got, input)
	}
}

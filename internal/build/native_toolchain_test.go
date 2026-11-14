package build

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestUsesWindowsToolchainProfile(t *testing.T) {
	for _, test := range []struct {
		name   string
		hostOS string
		conf   Config
		want   bool
	}{
		{name: "native Windows", hostOS: "windows", conf: Config{Goos: "windows", Goarch: "amd64"}, want: true},
		{name: "cross-architecture Windows", hostOS: "windows", conf: Config{Goos: "windows", Goarch: "arm64"}, want: true},
		{name: "cross-host Windows build", hostOS: "darwin", conf: Config{Goos: "windows", Goarch: "arm64", Mode: ModeBuild}, want: true},
		{name: "cross-host Windows test", hostOS: "linux", conf: Config{Goos: "windows", Goarch: "386", Mode: ModeTest}, want: true},
		{name: "cross-host Windows IR", hostOS: "darwin", conf: Config{Goos: "windows", Goarch: "arm64", Mode: ModeGen}},
		{name: "named target", hostOS: "windows", conf: Config{Goos: "windows", Goarch: "amd64", Target: "esp32c3"}},
		{name: "cross OS", hostOS: "windows", conf: Config{Goos: "linux", Goarch: "amd64"}},
		{name: "non-Windows host", hostOS: "darwin", conf: Config{Goos: "darwin", Goarch: "arm64"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := usesWindowsToolchainProfile(test.hostOS, &test.conf); got != test.want {
				t.Fatalf("usesWindowsToolchainProfile() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestParseNativeToolchainInput(t *testing.T) {
	commands := commandEnv{
		dir: "/work",
		environ: []string{
			"CC=old-clang",
			`CC=clang '--target=x86_64-pc-windows-msvc'`,
			`CXX="C:/Program Files/LLVM/bin/clang++.exe" --target=x86_64-pc-windows-msvc`,
		},
	}
	options := LinkOptions{
		ExternalLinker:      `clang "--target=x86_64-pc-windows-msvc"`,
		ExternalLinkerFlags: `'/debug:dwarf' /incremental:no`,
	}
	got, err := parseNativeToolchainInput(commands, options, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Dir != commands.dir || !reflect.DeepEqual(got.Environ, commands.environ) {
		t.Fatalf("invocation state = dir %q env %q", got.Dir, got.Environ)
	}
	if !got.ResolveWindows {
		t.Fatal("ResolveWindows = false, want true")
	}
	for name, values := range map[string]struct {
		got  []string
		want []string
	}{
		"CC":          {got.CC, []string{"clang", "--target=x86_64-pc-windows-msvc"}},
		"CXX":         {got.CXX, []string{"C:/Program Files/LLVM/bin/clang++.exe", "--target=x86_64-pc-windows-msvc"}},
		"-extld":      {got.ExternalLinker, []string{"clang", "--target=x86_64-pc-windows-msvc"}},
		"-extldflags": {got.ExternalFlags, []string{"/debug:dwarf", "/incremental:no"}},
	} {
		if !reflect.DeepEqual(values.got, values.want) {
			t.Errorf("%s = %q, want %q", name, values.got, values.want)
		}
	}

	got.Environ[0] = "changed"
	if commands.environ[0] == "changed" {
		t.Fatal("parseNativeToolchainInput aliased the invocation environment")
	}
}

func TestParseNativeToolchainInputErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		environ []string
		options LinkOptions
		wantErr string
	}{
		{name: "unterminated CC", environ: []string{"CC='clang"}, wantErr: "could not parse CC"},
		{name: "empty CXX", environ: []string{"CXX=  "}, wantErr: "CXX requires a non-empty command"},
		{name: "unterminated extld", options: LinkOptions{ExternalLinker: `'clang`}, wantErr: "could not parse -extld"},
		{name: "unterminated extldflags", options: LinkOptions{ExternalLinkerFlags: `'/debug`}, wantErr: "could not parse -extldflags"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseNativeToolchainInput(commandEnv{environ: test.environ}, test.options, false)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestBuildRejectsMalformedNativeToolchainInput(t *testing.T) {
	t.Setenv("CC", "'clang")
	conf := NewDefaultConf(ModeBuild)
	conf.Goos = "windows"
	conf.Goarch = "amd64"
	conf.GOAMD64 = "v1"

	_, err := Build(Invocation{Config: conf, Dir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "could not parse CC") {
		t.Fatalf("Build() error = %v, want malformed CC error", err)
	}
}

func TestBuildRejectsMalformedNativeLinkOptions(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix native linker path")
	}
	for _, test := range []struct {
		name    string
		options LinkOptions
	}{
		{"-extld", LinkOptions{ExternalLinker: "'clang"}},
		{"-extldflags", LinkOptions{ExternalLinkerFlags: "'-Wl,option"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.LinkOptions = test.options
			_, err := Build(Invocation{Config: conf, Dir: t.TempDir()})
			if err == nil || !strings.Contains(err.Error(), "could not parse "+test.name) {
				t.Fatalf("Build() error = %v, want malformed %s error", err, test.name)
			}
		})
	}
}

func TestDebuggerABIRootArgsLinkerDriver(t *testing.T) {
	for _, test := range []struct {
		name, goos, target, linker string
		want                       []string
	}{
		{"Linux default", "linux", "", "", []string{"-Wl,--undefined=" + debugabi.LegacyMarkerSymbol, "-Wl,--undefined=" + debugabi.NativeRecordSymbol}},
		{"Linux explicit driver", "linux", "", "clang++", []string{"-Wl,--undefined=" + debugabi.LegacyMarkerSymbol, "-Wl,--undefined=" + debugabi.NativeRecordSymbol}},
		{"Linux raw target linker", "linux", "custom", "ld.lld", []string{"--undefined=" + debugabi.LegacyMarkerSymbol, "--undefined=" + debugabi.NativeRecordSymbol}},
		{"Darwin default", "darwin", "", "", []string{"-Wl,-u,_" + debugabi.LegacyMarkerSymbol, "-Wl,-u,_" + debugabi.NativeRecordSymbol}},
		{"Darwin explicit driver", "darwin", "", "clang++", []string{"-Wl,-u,_" + debugabi.LegacyMarkerSymbol, "-Wl,-u,_" + debugabi.NativeRecordSymbol}},
		{"Darwin raw target linker", "darwin", "custom", "ld64.lld", []string{"-u", "_" + debugabi.LegacyMarkerSymbol, "-u", "_" + debugabi.NativeRecordSymbol}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := &context{
				buildConf:       &Config{Goos: test.goos, Goarch: "amd64", Target: test.target},
				frontendOptions: cl.Options{Debug: true},
				crossCompile:    crosscompile.Export{Linker: test.linker},
			}
			if got := debuggerABIRootArgs(ctx); !slices.Equal(got, test.want) {
				t.Fatalf("debugger roots = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNativeExternalLinkerDebugBuild(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("Unix native linker path")
	}
	driver, err := exec.LookPath("clang++")
	if err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(t.TempDir(), "external-linker-debug")
	conf.OptLevel, conf.LTO = optlevel.O3, lto.Full
	conf.LinkOptions = LinkOptions{ExternalLinker: driver, DWARF: DWARFPreserve}
	if _, err := Do([]string{"./testdata/dwarf"}, conf); err != nil {
		t.Fatal(err)
	}
	if got := runBinary(t, conf.OutFile); got != "dwarf-ok\n" {
		t.Fatalf("debug fixture output = %q", got)
	}
	verifyStrippedDebuggerSymbols(t, conf.OutFile)
}

func TestParseNativeToolchainInputAcceptsEmptyLinkerFlags(t *testing.T) {
	got, err := parseNativeToolchainInput(commandEnv{}, LinkOptions{
		ExternalLinker:      "  ",
		ExternalLinkerFlags: "\t",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ExternalLinker) != 0 || len(got.ExternalFlags) != 0 {
		t.Fatalf("external inputs = %q, %q; want empty", got.ExternalLinker, got.ExternalFlags)
	}
}

func TestCommandEnvLookupUsesLastValue(t *testing.T) {
	environ := []string{"CC=first", "malformed", "CXX=clang++", "CC=last"}
	commands := commandEnv{environ: environ}
	if got := commands.lookup("CC"); got != "last" {
		t.Fatalf("lookup(CC) = %q, want last", got)
	}
	if got := commands.lookup("CXX"); got != "clang++" {
		t.Fatalf("lookup(CXX) = %q, want clang++", got)
	}
	if got := commands.lookup("MISSING"); got != "" {
		t.Fatalf("lookup(MISSING) = %q", got)
	}
}

func TestNativeToolchainCommands(t *testing.T) {
	if os.Getenv("LLGO_NATIVE_COMMAND_HELPER") == "1" {
		separator := slices.Index(os.Args, "--")
		if separator < 0 {
			os.Exit(2)
		}
		_, _ = fmt.Fprint(os.Stdout, strings.Join(os.Args[separator+1:], " "))
		os.Exit(0)
	}

	helper := func(kind string) []string {
		return []string{os.Args[0], "-test.run=^TestNativeToolchainCommands$", "--", kind}
	}
	cc := helper("cc")
	cxx := helper("cxx")
	ctx := &context{
		buildConf: &Config{},
		crossCompile: crosscompile.Export{
			CC:      cc[0],
			CCArgs:  cc[1:],
			CXX:     cxx[0],
			CXXArgs: cxx[1:],
		},
		commands: commandEnv{
			dir:     t.TempDir(),
			environ: append(os.Environ(), "LLGO_NATIVE_COMMAND_HELPER=1"),
		},
	}
	run := func(name, want string, command func(*bytes.Buffer) error) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			if err := command(&output); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != want {
				t.Fatalf("command arguments = %q, want %q", got, want)
			}
		})
	}
	run("C source", "cc input", func(output *bytes.Buffer) error {
		cmd := ctx.compilerForSource("source.c")
		cmd.Stdout, cmd.Stderr = output, output
		return cmd.Compile("input")
	})
	run("C++ source", "cxx input", func(output *bytes.Buffer) error {
		cmd := ctx.compilerForSource("source.cpp")
		cmd.Stdout, cmd.Stderr = output, output
		return cmd.Compile("input")
	})
	run("default linker", "cxx input", func(output *bytes.Buffer) error {
		cmd := ctx.linker()
		cmd.Stdout, cmd.Stderr = output, output
		return cmd.Link("input")
	})

	extld := helper("extld")
	ctx.crossCompile.Linker = extld[0]
	ctx.crossCompile.LinkerArgs = extld[1:]
	run("explicit external linker", "extld input", func(output *bytes.Buffer) error {
		cmd := ctx.linker()
		cmd.Stdout, cmd.Stderr = output, output
		return cmd.Link("input")
	})
}

func TestIRCompilerUsesMatchingLLVMForEmscriptenMemory64(t *testing.T) {
	ctx := &context{crossCompile: crosscompile.Export{
		CC:      "emcc",
		CCArgs:  []string{"--emcc-prefix"},
		CCFLAGS: []string{"-target", "wasm64-unknown-emscripten"},
	}}
	config := ctx.irClangConfig()
	if config.CC != "emcc" || !reflect.DeepEqual(config.CCArgs, []string{"--emcc-prefix"}) {
		t.Fatalf("default IR compiler = %q %q, want emcc command", config.CC, config.CCArgs)
	}

	ctx.crossCompile.WasmProfile = crosscompile.WasmProfileJ64
	ctx.crossCompile.WasmProvider = crosscompile.WasmProviderEmscripten
	config = ctx.irClangConfig()
	if config.CC != "clang" || len(config.CCArgs) != 0 {
		t.Fatalf("Memory64 IR compiler = %q %q, want matching LLVM clang", config.CC, config.CCArgs)
	}
	wantFlags := []string{
		"-target", "wasm64-unknown-emscripten",
		"-mllvm", "-combiner-global-alias-analysis=false",
		"-mllvm", "-enable-emscripten-sjlj",
		"-mllvm", "-disable-lsr",
	}
	if !reflect.DeepEqual(config.CCFLAGS, wantFlags) {
		t.Fatalf("Memory64 IR compiler flags = %q", config.CCFLAGS)
	}
}

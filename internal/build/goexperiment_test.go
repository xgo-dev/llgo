package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func experimentCommands(t *testing.T, experiment string) commandEnv {
	t.Helper()
	return commandEnv{dir: t.TempDir(), environ: withEnv(os.Environ(),
		"GOENV=off", "GOFLAGS=", "GOOS=linux", "GOARCH=amd64", "GOAMD64=v3", "GOEXPERIMENT="+experiment)}
}

func mustResolveSourceGo(t *testing.T, commands commandEnv, override string) sourceGoConfig {
	t.Helper()
	cfg, err := resolveSourceGoConfig(commands, override)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestResolveGOEXPERIMENT(t *testing.T) {
	for _, test := range []struct {
		experiment string
		override   string
		simd       bool
	}{
		{"simd", "", true},
		{"simd,nosimd", "", false},
		{"nosimd,simd", "", true},
		{"simd,none", "", false},
		{"none,simd", "", true},
		{"nosimd", "simd", true},
		{"simd", "nosimd", false},
	} {
		t.Run(test.experiment+"/"+test.override, func(t *testing.T) {
			commands := experimentCommands(t, test.experiment)
			cfg := mustResolveSourceGo(t, commands, test.override)
			if got := slices.Contains(cfg.toolTags, "goexperiment.simd"); got != test.simd {
				t.Fatalf("simd = %v, want %v: %v", got, test.simd, cfg.toolTags)
			}
			if !slices.Contains(cfg.toolTags, "amd64.v3") || slices.Contains(cfg.toolTags, "arm64.v8.0") {
				t.Fatalf("tool tags do not match selected target: %v", cfg.toolTags)
			}
			commands.environ = cfg.apply(commands.environ)
			roundTrip := mustResolveSourceGo(t, commands, "")
			if !reflect.DeepEqual(cfg, roundTrip) {
				t.Fatalf("effective configuration changed after canonicalization:\n%+v\n%+v", cfg, roundTrip)
			}
		})
	}
}

func TestGOEXPERIMENTRejectsUnknown(t *testing.T) {
	_, err := resolveSourceGoConfig(experimentCommands(t, "llgo_unknown_experiment"), "")
	if err == nil || !strings.Contains(err.Error(), "unknown GOEXPERIMENT") {
		t.Fatalf("unexpected invalid-experiment result: %v", err)
	}
}

func TestSourceGoToolTagsAcrossSIMDTargets(t *testing.T) {
	for _, target := range []struct{ os, arch, tag string }{
		{"linux", "amd64", "amd64.v3"},
		{"linux", "arm64", "arm64.v8.0"},
		{"wasip1", "wasm", "wasm.satconv"},
	} {
		t.Run(target.arch, func(t *testing.T) {
			commands := experimentCommands(t, "simd")
			commands.environ = withEnv(commands.environ, "GOOS="+target.os, "GOARCH="+target.arch,
				"GOARM64=v8.0", "GOWASM=satconv,signext")
			cfg := mustResolveSourceGo(t, commands, "")
			if !slices.Contains(cfg.toolTags, "goexperiment.simd") || !slices.Contains(cfg.toolTags, target.tag) {
				t.Fatalf("incorrect %s tool tags: %v", target.arch, cfg.toolTags)
			}
			for _, tag := range cfg.toolTags {
				for _, arch := range []string{"amd64", "arm64", "wasm"} {
					if arch != target.arch && strings.HasPrefix(tag, arch+".") {
						t.Fatalf("foreign target tag %s in %s configuration", tag, target.arch)
					}
				}
			}
		})
	}
}

func TestGOEXPERIMENTSnapshotFromGOENV(t *testing.T) {
	for _, initial := range []string{"", "simd"} {
		t.Run(initial, func(t *testing.T) {
			commands := experimentCommands(t, "")
			goenv := filepath.Join(t.TempDir(), "env")
			writeEnv := func(experiment string) {
				t.Helper()
				if err := os.WriteFile(goenv, []byte("GOEXPERIMENT="+experiment+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeEnv(initial)
			commands.environ = withEnv(commands.environ, "GOENV="+goenv)
			cfg := mustResolveSourceGo(t, commands, "")
			if got := slices.Contains(cfg.toolTags, "goexperiment.simd"); got != (initial == "simd") {
				t.Fatalf("GOENV was not respected: %v", cfg.toolTags)
			}
			override := mustResolveSourceGo(t, commands, "nosimd")
			if slices.Contains(override.toolTags, "goexperiment.simd") {
				t.Fatal("explicit configuration did not override GOENV")
			}
			writeEnv("simd")
			if initial == "simd" {
				writeEnv("nosimd")
			}
			commands.environ = cfg.apply(commands.environ)
			after := mustResolveSourceGo(t, commands, "")
			if !reflect.DeepEqual(cfg, after) {
				t.Fatalf("GOENV mutation changed the pinned source configuration:\n%+v\n%+v", cfg, after)
			}
		})
	}
}

func TestResolveSourceGoUsesInvocationDir(t *testing.T) {
	commands := experimentCommands(t, "")
	commands.environ = withEnv(commands.environ, "GOTOOLCHAIN=path")
	if err := os.WriteFile(filepath.Join(commands.dir, "go.mod"), []byte("module example.org/future\n\ngo 1.999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSourceGoConfig(commands, ""); err == nil || !strings.Contains(err.Error(), "go1.999") {
		t.Fatalf("source configuration did not use invocation module: %v", err)
	}
}

func TestSourceGoToolTagsInheritBuildEnvironment(t *testing.T) {
	for _, mode := range []string{"module-on", "module-off", "goflags-modfile", "buildflags-modfile", "workspace"} {
		t.Run(mode, func(t *testing.T) {
			commands := experimentCommands(t, "")
			commands.environ = withEnv(commands.environ, "GOTOOLCHAIN=local", "GO111MODULE=on", "GOWORK=off")
			write := func(name, content string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(commands.dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write("go.mod", "module example.org/versioned\n\ngo 1.999\n")
			write("selected.mod", "module example.org/versioned\n\ngo 1.26\n")
			var flags []string
			switch mode {
			case "module-off":
				commands.environ = withEnv(commands.environ, "GO111MODULE=off")
			case "goflags-modfile":
				commands.environ = withEnv(commands.environ, "GOFLAGS=-modfile=selected.mod -p=1")
			case "buildflags-modfile":
				flags = []string{"-modfile=selected.mod"}
			case "workspace":
				write("go.mod", "module example.org/versioned\n\ngo 1.26\n")
				write("go.work", "go 1.26\nuse .\n")
				// -mod=mod is incompatible with workspace mode. Retaining both
				// settings must report that conflict instead of hiding it.
				commands.environ = withEnv(commands.environ, "GOWORK="+filepath.Join(commands.dir, "go.work"), "GOFLAGS=-mod=mod")
			}
			cfg, err := resolveSourceGoConfig(commands, "", flags...)
			if mode == "module-on" {
				if err == nil || !strings.Contains(err.Error(), "1.999") {
					t.Fatalf("module minimum was ignored: %v", err)
				}
				return
			}
			if mode == "workspace" {
				if err == nil || !strings.Contains(err.Error(), "workspace") {
					t.Fatalf("workspace configuration was ignored: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(cfg.toolTags, "amd64.v3") {
				t.Fatalf("missing selected toolchain tags: %v", cfg.toolTags)
			}
		})
	}
}

func TestSourcePatchMatchesEffectiveToolTags(t *testing.T) {
	commands := experimentCommands(t, "simd")
	cfg := mustResolveSourceGo(t, commands, "")
	ctx, err := newSourcePatchMatchContext(cfg.GOROOT, sourcePatchBuildContext{
		goos: "linux", goarch: "amd64", goversion: cfg.GOVERSION, toolTags: cfg.toolTags,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		tag  string
		want bool
	}{
		{"goexperiment.simd && amd64.v3", true},
		{"!goexperiment.simd", false},
		{"arm64.v8.0", false},
		{"amd64.v4", false},
	} {
		if err := os.WriteFile(filepath.Join(commands.dir, "selected.go"), []byte("//go:build "+test.tag+"\n\npackage selected\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := ctx.MatchFile(commands.dir, "selected.go"); err != nil || got != test.want {
			t.Fatalf("MatchFile(%q) = %v, %v, want %v", test.tag, got, err, test.want)
		}
	}
}

func TestSourcePatchEmptyToolTags(t *testing.T) {
	ctx, err := newSourcePatchMatchContext("", sourcePatchBuildContext{toolTags: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.ToolTags) != 0 {
		t.Fatalf("explicit empty tool tags inherited the host configuration: %v", ctx.ToolTags)
	}
}

func TestGOEXPERIMENTSeparatesCacheFingerprints(t *testing.T) {
	commands := experimentCommands(t, "")
	manifest := func(source sourceGoConfig) (string, string) {
		conf := &Config{Goos: "linux", Goarch: "amd64", GOEXPERIMENT: source.GOEXPERIMENT,
			sourceGoVersion: source.GOVERSION, toolTags: source.toolTags}
		m := newManifestBuilder()
		ctx := &context{buildConf: conf, llvmVersion: "test"}
		ctx.collectEnvInputs(m)
		return m.Build(), m.Fingerprint()
	}
	enabled := mustResolveSourceGo(t, commands, "simd")
	disabled := mustResolveSourceGo(t, commands, "nosimd")
	_, on := manifest(enabled)
	_, off := manifest(disabled)
	if on == off {
		t.Fatal("GOEXPERIMENT=simd and nosimd share the same package cache fingerprint")
	}
	equivalent := mustResolveSourceGo(t, commands, "simd,nosimd")
	if _, got := manifest(equivalent); got != off {
		t.Fatal("equivalent experiment spellings do not share a cache fingerprint")
	}
	enabled.GOVERSION += ".different"
	if _, got := manifest(enabled); got == on {
		t.Fatal("source toolchain version does not separate cache fingerprints")
	}
	text, _ := manifest(disabled)
	for _, key := range []string{"GOEXPERIMENT:", "SOURCE_GO_VERSION:", "TOOL_TAGS:"} {
		if !strings.Contains(text, key) {
			t.Fatalf("manifest missing %s:\n%s", key, text)
		}
	}
}

func TestSourceGoConfigPinsChildCommands(t *testing.T) {
	commands := experimentCommands(t, "simd")
	cfg := mustResolveSourceGo(t, commands, "")
	commands.environ = cfg.apply(withEnv(commands.environ, "GOEXPERIMENT=nosimd"))
	cmd := commands.configure(exec.Command("go", "env", "GOEXPERIMENT", "GOROOT", "GOVERSION"))
	got, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{cfg.GOEXPERIMENT, cfg.GOROOT, cfg.GOVERSION, ""}, "\n")
	if string(got) != want {
		t.Fatalf("child configuration = %q, want %q", got, want)
	}
}

func TestBuildSelectsGOEXPERIMENTSources(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module example.com/experiments\n\ngo 1.27\n",
		"on.go":  "//go:build goexperiment.simd\n\npackage experiments\nfunc Selected() int { return 127 }\n",
		"off.go": "//go:build !goexperiment.simd\n\npackage experiments\nfunc Selected() int { return 126 }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(llgoBuildCache, "0")
	t.Setenv("GOEXPERIMENT", "nosimd")
	for _, test := range []struct{ experiment, result string }{{"simd", "127"}, {"nosimd", "126"}, {"simd,nosimd", "126"}} {
		t.Run(test.experiment, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.GOEXPERIMENT = test.experiment
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 {
				t.Fatalf("Build returned %d packages", len(pkgs))
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			if ir := pkgs[0].LPkg.String(); !strings.Contains(ir, "ret i64 "+test.result) && !strings.Contains(ir, "ret i32 "+test.result) {
				t.Fatalf("build selected wrong source for %s:\n%s", test.experiment, ir)
			}
			if conf.GOEXPERIMENT != test.experiment || conf.sourceGoVersion != "" || conf.toolTags != nil {
				t.Fatal("Build mutated caller configuration")
			}
		})
	}
}

// runGoConfigHelper runs in a copy of the test executable named go (or go.exe).
func runGoConfigHelper(mode string) {
	if mode == "package-driver" || mode == "toolchain-driver" {
		if os.Getenv("GO111MODULE") == "off" {
			fmt.Fprintln(os.Stderr, "old launcher cannot handle selected GOEXPERIMENT")
			os.Exit(19)
		}
		name := "go"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		root := runtime.GOROOT()
		if mode == "toolchain-driver" {
			tool, err := os.Executable()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			data, err := os.ReadFile(tool + ".root")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			root = string(data)
		}
		cmd := exec.Command(filepath.Join(root, "bin", name), os.Args[1:]...)
		cmd.Env = withEnv(os.Environ(), "GOROOT="+root)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "list" {
		fmt.Fprintln(os.Stderr, "tool tags unavailable")
		os.Exit(7)
	}
	switch mode {
	case "invalid-json":
		fmt.Println("invalid JSON")
	case "missing-root":
		fmt.Println(`{"GOVERSION":"go1.27.0"}`)
	case "missing-version":
		fmt.Println(`{"GOROOT":"test-root"}`)
	case "list-failure":
		json.NewEncoder(os.Stdout).Encode(sourceGoConfig{GOROOT: os.Getenv("LLGO_TEST_SOURCE_GOROOT"), GOVERSION: "go1.27.0"})
	case "selected-root", "invalid-goflags":
		cfg := sourceGoConfig{GOROOT: runtime.GOROOT(), GOVERSION: runtime.Version()}
		if mode == "invalid-goflags" {
			cfg.GOFLAGS = "'-modfile=unterminated"
		}
		json.NewEncoder(os.Stdout).Encode(cfg)
	}
}

func TestSourceGoConfigErrors(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	writeBuildTestTool(t, bin, "go")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct{ mode, want string }{
		{"invalid-json", "decode Go source configuration:"},
		{"missing-root", "Go source configuration is missing GOROOT or GOVERSION"},
		{"missing-version", "Go source configuration is missing GOROOT or GOVERSION"},
		{"list-failure", "resolve Go tool tags: tool tags unavailable:"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			commands := experimentCommands(t, "")
			commands.environ = withEnv(commands.environ, "LLGO_TEST_GO_CONFIG_HELPER="+tc.mode,
				"LLGO_TEST_SOURCE_GOROOT="+filepath.Dir(bin))
			_, err := resolveSourceGoConfig(commands, "")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("resolveSourceGoConfig() error = %v, want %q", err, tc.want)
			}
			if tc.mode == "list-failure" {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 7 {
					t.Fatalf("error does not preserve subprocess exit status: %v", err)
				}
			}
		})
	}
	t.Run("start-failure", func(t *testing.T) {
		commands := experimentCommands(t, "")
		commands.dir = filepath.Join(commands.dir, "missing")
		_, err := resolveSourceGoConfig(commands, "")
		if err == nil || !strings.Contains(err.Error(), "resolve Go source configuration:") || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected source configuration error preserving missing-directory cause, got %v", err)
		}
	})
}

// The launcher on PATH need not be the selected source compiler. In particular,
// GO111MODULE=off disables toolchain switching even with explicit GOTOOLCHAIN.
func TestSourceGoToolTagsUseResolvedCompiler(t *testing.T) {
	bin := t.TempDir()
	writeBuildTestTool(t, bin, "go")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	commands := experimentCommands(t, "")
	commands.environ = withEnv(commands.environ, "LLGO_TEST_GO_CONFIG_HELPER=selected-root")
	cfg := mustResolveSourceGo(t, commands, "")
	if !slices.Contains(cfg.toolTags, "goexperiment.jsonv2") {
		t.Fatalf("lost selected Go 1.27 default experiment: %v", cfg.toolTags)
	}
	commands.environ = cfg.apply(commands.environ)
	goExe := "go"
	if runtime.GOOS == "windows" {
		goExe += ".exe"
	}
	output, err := commands.configure(exec.Command(filepath.Join(cfg.GOROOT, "bin", goExe), "list", "encoding/json/v2", "encoding/json/jsontext")).CombinedOutput()
	if err != nil {
		t.Fatalf("selected JSON packages unavailable: %v\n%s", err, output)
	}
}

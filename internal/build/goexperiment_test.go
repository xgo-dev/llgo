package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	commands.environ = withEnv(commands.environ, "GOTOOLCHAIN=local")
	if err := os.WriteFile(filepath.Join(commands.dir, "go.mod"), []byte("module example.org/future\n\ngo 1.999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveSourceGoConfig(commands, ""); err == nil || !strings.Contains(err.Error(), "requires go >= 1.999") {
		t.Fatalf("source configuration did not use invocation module: %v", err)
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

package goroot

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestGOROOTJSRuntimeEnvironment(t *testing.T) {
	for _, profile := range []string{"", "J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI"} {
		t.Run(profile, func(t *testing.T) {
			withGOROOTWasmProfile(t, profile)
			env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo", "PATH=/bin", "HOME=/home/test", "TMPDIR=/tmp",
				"GODEBUG=panicnil=1", "GOEXPERIMENT=loopvar", "GOMEMLIMIT=128MiB", "GOGC=50", "GOTRACEBACK=all",
				"SystemRoot=C:\\Windows", "CGO_LDFLAGS=" + strings.Repeat("x", 16000), "GITHUB_EVENT_PATH=/ci/event"}
			original := append([]string(nil), env...)
			var baseline []string
			for _, llgo := range []bool{false, true} {
				_, _, got, err := gorootArtifactCommand(t.TempDir(), "out", llgo, env)
				if err != nil {
					t.Fatal(err)
				}
				js := strings.HasPrefix(profile, "J")
				if js {
					if envEntry(got, "CGO_LDFLAGS") != "" || envEntry(got, "GITHUB_EVENT_PATH") != "" {
						t.Fatal("JS runtime inherited build/CI environment")
					}
					for _, key := range []string{"GOROOT", "LLGO_ROOT", "PATH", "HOME", "TMPDIR", "SystemRoot", "GODEBUG", "GOEXPERIMENT", "GOMEMLIMIT", "GOGC", "GOTRACEBACK"} {
						if envEntry(got, key) != envEntry(env, key) {
							t.Fatalf("lost runtime variable %s", key)
						}
					}
				} else if envEntry(got, "CGO_LDFLAGS") != envEntry(env, "CGO_LDFLAGS") {
					t.Fatal("changed non-JS runtime environment")
				}
				if llgo && !reflect.DeepEqual(got, baseline) {
					t.Fatal("Go and LLGo have different runtime environments")
				}
				baseline = got
			}
			if !reflect.DeepEqual(env, original) {
				t.Fatal("changed compiler environment")
			}
		})
	}
}

func withGOROOTWasmProfile(t *testing.T, name string) {
	t.Helper()
	old := *flagWasmProfile
	*flagWasmProfile = name
	t.Cleanup(func() { *flagWasmProfile = old })
}

func TestGOROOTWasmProfiles(t *testing.T) {
	want := map[string]struct{ target, goos, suffix, runner string }{
		"J32-GoJS":       {"", "js", ".mjs", "emscripten-runner.mjs"},
		"J32-Emscripten": {"emscripten", "js", ".mjs", "emscripten-runner.mjs"},
		"J64-Emscripten": {"emscripten-memory64", "js", ".mjs", "emscripten-memory64-runner.mjs"},
		"W32-WASI":       {"wasi", "wasip1", ".wasm", "wasmer"},
	}
	for name, expected := range want {
		got, ok, err := selectGOROOTWasmProfile(name)
		if err != nil || !ok || got.target != expected.target || got.goos != expected.goos || got.llgoSuffix != expected.suffix || got.runner != expected.runner {
			t.Fatalf("%s: got %+v, %v, %v; want %+v", name, got, ok, err, expected)
		}
	}
	if _, ok, err := selectGOROOTWasmProfile(""); err != nil || ok {
		t.Fatalf("empty profile: ok=%v err=%v", ok, err)
	}
	if _, _, err := selectGOROOTWasmProfile("bad"); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestGOROOTWasmBuildAndRunCommands(t *testing.T) {
	withGOROOTWasmProfile(t, "J64-Emscripten")
	env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo", "GOOS=linux", "GOARCH=amd64"}
	if got := gorootArtifactPath("/tmp", "llgo", true); got != filepath.Join("/tmp", "llgo.mjs") {
		t.Fatal(got)
	}
	wantBuild := []string{"build", "-target", "emscripten-memory64", "-tags=x", "-o", "out.mjs", "."}
	if got := gorootBuildArgs(true, []string{"-tags=x"}, "out.mjs", "."); !reflect.DeepEqual(got, wantBuild) {
		t.Fatalf("build args: %v", got)
	}
	app, args, targetEnv, err := gorootArtifactCommand("/work", "out.mjs", true, env, "one")
	if err != nil || app != "node" || !reflect.DeepEqual(args, []string{"--stack-size=8192", filepath.Join("/llgo", "targets", "emscripten-memory64-runner.mjs"), "out.mjs", "one"}) {
		t.Fatalf("LLGo command: %q %v %v", app, args, err)
	}
	if envEntry(targetEnv, "GOOS") != "js" || envEntry(targetEnv, "GOARCH") != "wasm" || envEntry(targetEnv, "CGO_ENABLED") != "0" || envEntry(targetEnv, "GOMAXPROCS") != "1" {
		t.Fatalf("target env: %v", targetEnv)
	}
	app, args, _, err = gorootArtifactCommand("/work", "go.wasm", false, env, "two")
	if err != nil || app != filepath.Join("/go", "lib", "wasm", "go_js_wasm_exec") || !reflect.DeepEqual(args, []string{"go.wasm", "two"}) {
		t.Fatalf("Go command: %q %v %v", app, args, err)
	}
}

func TestGOROOTBuildArgsKeepGCBackendControlsOnBaseline(t *testing.T) {
	withGOROOTWasmProfile(t, "")
	for _, payload := range []string{"-l=4", "-d=ssa/check/on", "-d=softfloat", "-d=checkptr", "all=-d=checkptr"} {
		for _, flags := range [][]string{{"-gcflags", payload}, {"-gcflags=" + payload}} {
			baseline := append(append([]string{"build"}, flags...), "-o", "out", "source.go")
			if got := gorootBuildArgs(false, flags, "out", "source.go"); !reflect.DeepEqual(got, baseline) {
				t.Fatalf("Go baseline flags changed: %v", got)
			}
			want := []string{"build", "-o", "out", "source.go"}
			if got := gorootBuildArgs(true, flags, "out", "source.go"); !reflect.DeepEqual(got, want) {
				t.Fatalf("LLVM build retained gc-only backend controls: %v", got)
			}
		}
	}
	for _, flags := range [][]string{
		{"-gcflags=-N -l=4"},
		{"-gcflags", "-l=2"},
		{"-gcflags=-d=unknown"},
		{"-gcflags=-d=converthash=qy"},
		{"-gcflags=-d=checkptr -N"},
		{"-ldflags", "-gcflags=-l=4"},
		{"-tags", "-gcflags=-l=4"},
	} {
		want := append(append([]string{"build"}, flags...), "-o", "out", "source.go")
		if got := gorootBuildArgs(true, flags, "out", "source.go"); !reflect.DeepEqual(got, want) {
			t.Fatalf("unrelated payload changed: got %v, want %v", got, want)
		}
	}
}

func TestGOROOTGoJSRunCommandModelsBrowser(t *testing.T) {
	withGOROOTWasmProfile(t, "J32-GoJS")
	env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo"}
	app, args, _, err := gorootArtifactCommand("/work", "out.mjs", true, env, "arg")
	want := []string{"--stack-size=8192", filepath.Join("/llgo", "targets", "emscripten-runner.mjs"), "--browser-only", "out.mjs", "arg"}
	if err != nil || app != "node" || !reflect.DeepEqual(args, want) {
		t.Fatalf("GoJS command: %q %v %v", app, args, err)
	}
}

func TestGOROOTWasiRunCommand(t *testing.T) {
	withGOROOTWasmProfile(t, "W32-WASI")
	dir := t.TempDir()
	for _, llgo := range []bool{false, true} {
		for _, threads := range []string{"", "1"} {
			env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo", "LLGO_WASI_THREADS=" + threads, "RUST_LOG=warn", "PATH=/tools/bin:/usr/bin"}
			app, args, targetEnv, err := gorootArtifactCommand(dir, "out.wasm", llgo, env, "-test.v")
			workVolume, tempVolume, guestCwd := dir, "/tmp", dir
			if runtime.GOOS == "windows" {
				workVolume, tempVolume, guestCwd = dir+":/work", os.TempDir()+":/tmp", "/work"
			}
			rootVolume, guestRoot := "/go", "/go"
			if runtime.GOOS == "windows" {
				rootVolume, guestRoot = "/go:/goroot", "/goroot"
			}
			want := []string{"run", "--env=PATH=/tools/bin:/usr/bin", "--volume=" + rootVolume, "--env=GOROOT=" + guestRoot, "--enable-exceptions", "--enable-simd", "--stack-size=1048576",
				"--volume=" + workVolume, "--volume=" + tempVolume, "--env=PWD=" + guestCwd, "out.wasm", "--", "-test.v"}
			if err != nil || app != "wasmer" || !reflect.DeepEqual(args, want) || envEntry(targetEnv, "GOWASIRUNTIME") != "wasmtime" || envEntry(targetEnv, "RUST_LOG") != "off" {
				t.Fatalf("llgo=%v threads=%q: WASI command: %q %v %v %v", llgo, threads, app, args, targetEnv, err)
			}
		}
	}
}

func TestGOROOTWASIHostArgs(t *testing.T) {
	for _, tt := range []struct {
		name, cwd, temp string
		windows         bool
		want            []string
	}{
		{"unix", "/project with spaces", "/host-temp", false,
			[]string{"run", "--enable-exceptions", "--enable-simd", "--stack-size=1048576",
				"--volume=/project with spaces", "--volume=/tmp", "--env=PWD=/project with spaces", "out.wasm", "--"}},
		{"windows", `C:\project with spaces`, `D:\Temp dir`, true,
			[]string{"run", "--enable-exceptions", "--enable-simd", "--stack-size=1048576",
				`--volume=C:\project with spaces:/work`, `--volume=D:\Temp dir:/tmp`, "--env=PWD=/work", "out.wasm", "--"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := gorootWASIArgs(tt.cwd, tt.temp, "out.wasm", tt.windows); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("WASI arguments = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGOROOTWasmStackGrowthBudget(t *testing.T) {
	for _, profile := range []string{"", "J32-GoJS", "J32-Emscripten", "J64-Emscripten", "W32-WASI"} {
		t.Run(profile, func(t *testing.T) {
			withGOROOTWasmProfile(t, profile)
			for _, path := range []string{"uintptrescapes.go", "helloworld.go"} {
				want := []string{"-tags=x"}
				if profile != "" && path == "uintptrescapes.go" {
					want = append(want, "-goroutine-stack-size=8MB")
				}
				got := gorootWasmCaseBuildFlags(path, []string{"-tags=x"})
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: got %v, want %v", path, got, want)
				}
			}
		})
	}
}

package goroot

import (
	"path/filepath"
	"reflect"
	"testing"
)

func withGOROOTWasmProfile(t *testing.T, name string) {
	t.Helper()
	old := *flagWasmProfile
	*flagWasmProfile = name
	t.Cleanup(func() { *flagWasmProfile = old })
}

func TestGOROOTWasmProfiles(t *testing.T) {
	want := map[string]struct{ target, goos, suffix, runner string }{
		"EC32":  {"emscripten", "js", ".mjs", "emscripten-runner.mjs"},
		"EC64":  {"emscripten-memory64", "js", ".mjs", "emscripten-memory64-runner.mjs"},
		"WC32":  {"wasi", "wasip1", ".wasm", "wasmtime"},
		"GJS":   {"", "js", ".mjs", "emscripten-runner.mjs"},
		"GWASI": {"", "wasip1", ".wasm", "wasmtime"},
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
	withGOROOTWasmProfile(t, "EC64")
	env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo", "GOOS=linux", "GOARCH=amd64"}
	if got := gorootArtifactPath("/tmp", "llgo", true); got != filepath.Join("/tmp", "llgo.mjs") {
		t.Fatal(got)
	}
	wantBuild := []string{"build", "-target", "emscripten-memory64", "-tags=x", "-o", "out.mjs", "."}
	if got := gorootBuildArgs(true, []string{"-tags=x"}, "out.mjs", "."); !reflect.DeepEqual(got, wantBuild) {
		t.Fatalf("build args: %v", got)
	}
	app, args, targetEnv, err := gorootArtifactCommand("/work", "out.mjs", true, env, "one")
	if err != nil || app != "node" || !reflect.DeepEqual(args, []string{filepath.Join("/llgo", "targets", "emscripten-memory64-runner.mjs"), "out.mjs", "one"}) {
		t.Fatalf("LLGo command: %q %v %v", app, args, err)
	}
	if envEntry(targetEnv, "GOOS") != "js" || envEntry(targetEnv, "GOARCH") != "wasm" || envEntry(targetEnv, "CGO_ENABLED") != "0" {
		t.Fatalf("target env: %v", targetEnv)
	}
	app, args, _, err = gorootArtifactCommand("/work", "go.wasm", false, env, "two")
	if err != nil || app != filepath.Join("/go", "lib", "wasm", "go_js_wasm_exec") || !reflect.DeepEqual(args, []string{"go.wasm", "two"}) {
		t.Fatalf("Go command: %q %v %v", app, args, err)
	}
}

func TestGOROOTWasiRunCommand(t *testing.T) {
	withGOROOTWasmProfile(t, "GWASI")
	env := []string{"GOROOT=/go", "LLGO_ROOT=/llgo"}
	app, args, targetEnv, err := gorootArtifactCommand("/work", "out.wasm", true, env, "arg")
	want := []string{"run", "-W", "exceptions=y", "--dir=.", "out.wasm", "arg"}
	if err != nil || app != "wasmtime" || !reflect.DeepEqual(args, want) || envEntry(targetEnv, "GOWASIRUNTIME") != "wasmtime" {
		t.Fatalf("WASI command: %q %v %v %v", app, args, targetEnv, err)
	}
}

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRunCLICollectsEveryWasmProfile(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "benchmark", "binary_size", "println", "main.go")
	if err := os.MkdirAll(filepath.Dir(fixture), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "results")
	var calls int
	runner := func(_ context.Context, dir string, env []string, name string, args ...string) error {
		calls++
		if dir != root || (name != "fake-llgo" && name != "fake-go") {
			t.Fatalf("runner = (%q, %q), want (%q, fake-llgo or fake-go)", dir, name, root)
		}
		output := args[slices.Index(args, "-o")+1]
		if err := os.WriteFile(strings.TrimSuffix(output, filepath.Ext(output))+".wasm", []byte("\x00asmfixture"), 0o644); err != nil {
			return err
		}
		if filepath.Ext(output) == ".mjs" {
			return os.WriteFile(output, []byte("export default {}"), 0o644)
		}
		return nil
	}
	if code := runMain(context.Background(), io.Discard, []string{
		"-root", root,
		"-llgo", "fake-llgo",
		"-go", "fake-go",
		"-out", out,
		"-build-runs", "1",
	}, runner); code != 0 {
		t.Fatalf("runMain exit code = %d, want 0", code)
	}
	if want := len(wasmProfiles)*2 + len(goWasmProfiles); calls != want {
		t.Fatalf("build calls = %d, want %d", calls, want)
	}
	data, err := os.ReadFile(filepath.Join(out, "benchmark.txt"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, profile := range wasmProfiles {
		if !strings.Contains(text, "BenchmarkWasmSize/"+profile.name+"/LLGo ") ||
			!strings.Contains(text, "BenchmarkWasmBuild/"+profile.name+" ") {
			t.Errorf("result omits LLGo %s measurements:\n%s", profile.name, text)
		}
	}
	for _, profile := range goWasmProfiles {
		if !strings.Contains(text, "BenchmarkWasmSize/"+profile.name+"/Go ") {
			t.Errorf("result omits official Go %s size:\n%s", profile.name, text)
		}
	}
}

func TestMeasureGoProfile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	fixture := filepath.Join(root, "main.go")
	profile := goWasmProfile{name: "js", goos: "js"}
	var gotEnv []string
	result, err := measureGoProfile(context.Background(), func(_ context.Context, dir string, env []string, name string, args ...string) error {
		if dir != root || name != "fake-go" {
			t.Fatalf("runner = (%q, %q), want (%q, fake-go)", dir, name, root)
		}
		gotEnv = slices.Clone(env)
		output := args[slices.Index(args, "-o")+1]
		return os.WriteFile(output, []byte("\x00asmfixture"), 0o644)
	}, nil, root, "fake-go", out, fixture, profile)
	if err != nil {
		t.Fatal(err)
	}
	if result.name != "js" || result.moduleBytes != int64(len("\x00asmfixture")) {
		t.Fatalf("measurement = %+v", result)
	}
	if !slices.Contains(gotEnv, "GOOS=js") || !slices.Contains(gotEnv, "GOARCH=wasm") {
		t.Fatalf("Go WebAssembly environment = %v", gotEnv)
	}
}

func TestMeasureGoProfileReportsBuildFailure(t *testing.T) {
	want := errors.New("go build failed")
	_, err := measureGoProfile(context.Background(), func(context.Context, string, []string, string, ...string) error {
		return want
	}, nil, t.TempDir(), "fake-go", t.TempDir(), "main.go", goWasmProfile{name: "wasip1", goos: "wasip1"})
	if !errors.Is(err, want) {
		t.Fatalf("measureGoProfile error = %v, want %v", err, want)
	}
}

func TestMeasureGoProfileReportsOutputFailures(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "bin"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := goWasmProfile{name: "js", goos: "js"}
	if _, err := measureGoProfile(context.Background(), nil, nil, root, "fake-go", out, "main.go", profile); err == nil {
		t.Fatal("measureGoProfile unexpectedly accepted a blocked output directory")
	}

	if _, err := measureGoProfile(context.Background(), func(context.Context, string, []string, string, ...string) error {
		return nil
	}, nil, root, "fake-go", t.TempDir(), "main.go", profile); err == nil || !strings.Contains(err.Error(), "inspect wasm module") {
		t.Fatalf("missing-module error = %v", err)
	}
}

func TestWasmModuleSizeRejectsHostArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.wasm")
	if err := os.WriteFile(path, []byte("not wasm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wasmModuleSize(path); err == nil || !strings.Contains(err.Error(), "not a WebAssembly module") {
		t.Fatalf("wasmModuleSize error = %v", err)
	}
}

func TestWasmModuleSizeRejectsMissingAndTruncatedArtifacts(t *testing.T) {
	if _, err := wasmModuleSize(filepath.Join(t.TempDir(), "missing.wasm")); err == nil || !strings.Contains(err.Error(), "inspect wasm module") {
		t.Fatalf("missing wasmModuleSize error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "short.wasm")
	if err := os.WriteFile(path, []byte("wa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wasmModuleSize(path); err == nil || !strings.Contains(err.Error(), "read wasm module") {
		t.Fatalf("truncated wasmModuleSize error = %v", err)
	}
}

func TestMedianDuration(t *testing.T) {
	values := []time.Duration{9, 1, 5}
	if got := medianDuration(values); got != 5 {
		t.Fatalf("medianDuration = %v, want 5ns", got)
	}
	if !slices.Equal(values, []time.Duration{9, 1, 5}) {
		t.Fatalf("medianDuration mutated input: %v", values)
	}
}

func TestRunCLIRejectsInvalidBuildCount(t *testing.T) {
	err := runCLI(context.Background(), []string{"-build-runs", "0"}, nil)
	if err == nil || !strings.Contains(err.Error(), "must be positive") {
		t.Fatalf("runCLI error = %v", err)
	}
}

func TestRunCLIRejectsInvalidFlag(t *testing.T) {
	err := runCLI(context.Background(), []string{"-unknown"}, nil)
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("runCLI error = %v", err)
	}
}

func TestRunMainReportsFailure(t *testing.T) {
	var stderr strings.Builder
	if code := runMain(context.Background(), &stderr, []string{"-unknown"}, nil); code != 1 {
		t.Fatalf("runMain exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("runMain stderr = %q", stderr.String())
	}
}

func TestRunCLIReturnsBuildFailure(t *testing.T) {
	want := errors.New("compiler failed")
	err := runCLI(context.Background(), []string{
		"-root", t.TempDir(),
		"-out", filepath.Join(t.TempDir(), "out"),
		"-build-runs", "1",
	}, func(context.Context, string, []string, string, ...string) error {
		return want
	})
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "build "+wasmProfiles[0].name) {
		t.Fatalf("runCLI error = %v, want wrapped %v", err, want)
	}
}

func TestRunCLIReturnsOutputCleanupFailure(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runCLI(context.Background(), []string{
		"-root", t.TempDir(),
		"-out", filepath.Join(parent, "out"),
	}, nil)
	if err == nil {
		t.Fatal("runCLI unexpectedly accepted an output below a file")
	}
}

func TestMeasureProfileFailures(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	fixture := filepath.Join(root, "main.go")
	profile := wasmProfile{name: "test", outputExt: ".wasm"}
	want := errors.New("compiler failed")

	if _, err := measureProfile(context.Background(), func(context.Context, string, []string, string, ...string) error {
		return want
	}, nil, root, "llgo", out, fixture, profile, 1); !errors.Is(err, want) || !strings.Contains(err.Error(), "warm build") {
		t.Fatalf("warm-build error = %v", err)
	}

	calls := 0
	if _, err := measureProfile(context.Background(), func(context.Context, string, []string, string, ...string) error {
		calls++
		if calls == 1 {
			return nil
		}
		return want
	}, nil, root, "llgo", out, fixture, profile, 1); !errors.Is(err, want) || strings.Contains(err.Error(), "warm build") {
		t.Fatalf("measured-build error = %v", err)
	}

	jsProfile := wasmProfile{name: "js", outputExt: ".mjs", hasJSGlue: true}
	writeModule := func(_ context.Context, _ string, _ []string, _ string, args ...string) error {
		output := args[slices.Index(args, "-o")+1]
		return os.WriteFile(strings.TrimSuffix(output, ".mjs")+".wasm", []byte("\x00asmfixture"), 0o644)
	}
	if _, err := measureProfile(context.Background(), writeModule, nil, root, "llgo", out, fixture, jsProfile, 1); err == nil || !strings.Contains(err.Error(), "inspect JS glue") {
		t.Fatalf("missing-glue error = %v", err)
	}

	writeEmptyGlue := func(ctx context.Context, dir string, env []string, name string, args ...string) error {
		if err := writeModule(ctx, dir, env, name, args...); err != nil {
			return err
		}
		output := args[slices.Index(args, "-o")+1]
		return os.WriteFile(output, nil, 0o644)
	}
	if _, err := measureProfile(context.Background(), writeEmptyGlue, nil, root, "llgo", out, fixture, jsProfile, 1); err == nil || !strings.Contains(err.Error(), "generated empty JS glue") {
		t.Fatalf("empty-glue error = %v", err)
	}

	if _, err := measureProfile(context.Background(), func(context.Context, string, []string, string, ...string) error {
		return nil
	}, nil, root, "llgo", out, fixture, profile, 1); err == nil || !strings.Contains(err.Error(), "inspect wasm module") {
		t.Fatalf("missing-module error = %v", err)
	}

	blockedOut := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(blockedOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blockedOut, "bin"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := measureProfile(context.Background(), nil, nil, root, "llgo", blockedOut, fixture, profile, 1); err == nil || !strings.Contains(err.Error(), "warm build") {
		t.Fatalf("profile-cleanup error = %v", err)
	}
}

func TestWriteResultsReturnsFilesystemError(t *testing.T) {
	err := writeResults(filepath.Join(t.TempDir(), "missing", "benchmark.txt"), nil, nil)
	if err == nil {
		t.Fatal("writeResults unexpectedly succeeded")
	}
}

func TestRunCommand(t *testing.T) {
	if err := runCommand(context.Background(), t.TempDir(), os.Environ(), "go", "env", "GOOS"); err != nil {
		t.Fatal(err)
	}
}

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRunCLICollectsEveryExampleAndProfile(t *testing.T) {
	for _, test := range []struct {
		name  string
		flags []string
		runs  int
	}{
		{name: "default", runs: 3},
		{name: "one-sample", flags: []string{"-build-runs", "1"}, runs: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			out := filepath.Join(t.TempDir(), "results")
			// Specify the acceptance matrix independently of the production lists:
			// removing an example/profile there must make this test fail.
			profiles := []string{
				"j32-goos-js",
				"w32-goos-wasip1",
				"j32-emscripten",
				"j64-emscripten-memory64",
				"w32-wasi",
			}
			targets := map[string]string{
				"j32-emscripten":          "emscripten",
				"j64-emscripten-memory64": "emscripten-memory64",
				"w32-wasi":                "wasi",
			}
			gooses := map[string]string{
				"j32-goos-js":     "js",
				"w32-goos-wasip1": "wasip1",
			}
			wantCalls := make(map[string]int)
			wantMetrics := make(map[string]int)
			for _, example := range []string{"cprintf", "println", "fmtprintf", "reflectcall"} {
				fixtureRoot := filepath.Join("benchmark", "binary_size")
				if example == "reflectcall" {
					fixtureRoot = filepath.Join("benchmark", "wasm", "testdata")
				}
				fixture := filepath.Join(root, fixtureRoot, example, "main.go")
				if err := os.MkdirAll(filepath.Dir(fixture), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(fixture, []byte("fixture "+example), 0o644); err != nil {
					t.Fatal(err)
				}
				for _, profile := range profiles {
					metricName := profile
					if example != "println" {
						metricName = example + "/" + profile
					}
					wantCalls[example+"/"+profile+"/fake-llgo"] = 1
					wantMetrics["BenchmarkWasmSize/"+metricName+"/LLGo"] = 1
					if example == "println" || (example == "reflectcall" && profile == "w32-wasi") {
						wantCalls[example+"/"+profile+"/fake-llgo"] = test.runs + 1
						wantMetrics["BenchmarkWasmBuild/"+metricName] = 1
					}
					if example != "cprintf" && gooses[profile] != "" {
						wantCalls[example+"/"+profile+"/fake-go"] = 1
						wantMetrics["BenchmarkWasmSize/"+metricName+"/Go"] = 1
					}
				}
			}
			calls := make(map[string]int)
			artifacts := make(map[string]string)
			outputOwners := make(map[string]string)
			runner := func(_ context.Context, dir string, env []string, name string, args ...string) error {
				if dir != root || (name != "fake-llgo" && name != "fake-go") {
					t.Fatalf("runner = (%q, %q), want (%q, fake-llgo or fake-go)", dir, name, root)
				}
				fixture := args[len(args)-1]
				example := filepath.Base(filepath.Dir(fixture))
				data, err := os.ReadFile(fixture)
				if err != nil || string(data) != "fixture "+example {
					t.Fatalf("unexpected source %s: data=%q, error=%v", fixture, data, err)
				}
				output := args[slices.Index(args, "-o")+1]
				profile := strings.TrimPrefix(filepath.Base(filepath.Dir(output)), "go-")
				key := example + "/" + profile + "/" + name
				if _, ok := wantCalls[key]; !ok {
					t.Fatalf("unexpected compiler/example/profile combination %q", key)
				}
				calls[key]++
				if previous, ok := outputOwners[output]; ok && previous != key {
					t.Fatalf("artifact %s shared by %s and %s", output, previous, key)
				}
				outputOwners[output] = key
				profileDir := profile
				if name == "fake-go" {
					profileDir = "go-" + profile
				}
				ext := ".wasm"
				if name == "fake-llgo" && (profile == "j32-goos-js" || profile == "j32-emscripten" || profile == "j64-emscripten-memory64") {
					ext = ".mjs"
				}
				if want := filepath.Join(out, example, "bin", profileDir, "program"+ext); output != want {
					t.Fatalf("output = %s, want %s", output, want)
				}
				for _, setting := range []string{"LLGO_ROOT=" + root, "LLGO_BUILD_CACHE=off"} {
					if !slices.Contains(env, setting) {
						t.Errorf("%s environment omits %s", key, setting)
					}
				}
				targetIndex := slices.Index(args, "-target")
				if target := targets[profile]; target != "" {
					if targetIndex < 0 || args[targetIndex+1] != target {
						t.Fatalf("%s target arguments = %v, want %s", key, args, target)
					}
				} else if goos := gooses[profile]; goos == "" || targetIndex >= 0 || !slices.Contains(env, "GOOS="+goos) || !slices.Contains(env, "GOARCH=wasm") {
					t.Fatalf("%s compiler selection: args=%v, env=%v", key, args, env)
				}
				module := strings.TrimSuffix(output, ext) + ".wasm"
				artifacts[module] = "\x00asm" + key
				if err := os.WriteFile(module, []byte(artifacts[module]), 0o644); err != nil {
					return err
				}
				if ext == ".mjs" {
					artifacts[output] = "// " + key
					return os.WriteFile(output, []byte(artifacts[output]), 0o644)
				}
				return nil
			}
			args := append([]string{"-root", root, "-llgo", "fake-llgo", "-go", "fake-go", "-out", out}, test.flags...)
			var stderr strings.Builder
			if code := runMain(context.Background(), &stderr, args, runner); code != 0 {
				t.Fatalf("runMain exit code = %d: %s", code, stderr.String())
			}
			if !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("build calls = %v, want %v", calls, wantCalls)
			}
			for path, want := range artifacts {
				got, err := os.ReadFile(path)
				if err != nil || string(got) != want {
					t.Errorf("artifact overwritten/removed: %s: data=%q, error=%v", path, got, err)
				}
			}
			data, err := os.ReadFile(filepath.Join(out, "benchmark.txt"))
			if err != nil {
				t.Fatal(err)
			}
			metrics := make(map[string]int)
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "Benchmark") {
					metrics[strings.Fields(line)[0]]++
				}
			}
			if !reflect.DeepEqual(metrics, wantMetrics) {
				t.Fatalf("benchmark matrix = %v, want %v", metrics, wantMetrics)
			}
		})
	}
}

func TestMeasureGoProfile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	fixture := filepath.Join(root, "main.go")
	profile := goWasmProfile{name: "j32-goos-js", goos: "js"}
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
	if result.name != "j32-goos-js" || result.moduleBytes != int64(len("\x00asmfixture")) {
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
	if !errors.Is(err, want) || !strings.Contains(err.Error(), "build println/j32-goos-js") {
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
	}, nil, root, "llgo", out, fixture, profile, 0); !errors.Is(err, want) || strings.Contains(err.Error(), "warm build") {
		t.Fatalf("size-only build error = %v", err)
	}

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

func TestWriteResultsPreservesZeroBuildDuration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "benchmark.txt")
	// A fast build can take less than the clock resolution. Zero is a valid
	// sample, not an indication that this example was measured for size only.
	results := []measurement{
		{name: "timed-zero", buildMeasured: true},
		{name: "timed-positive", build: 17 * time.Nanosecond, buildMeasured: true},
		{name: "size-only"},
	}
	if err := writeResults(path, results, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var builds []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "BenchmarkWasmBuild/") {
			builds = append(builds, line)
		}
	}
	want := []string{
		"BenchmarkWasmBuild/timed-zero 1 0 build-ns",
		"BenchmarkWasmBuild/timed-positive 1 17 build-ns",
	}
	if !slices.Equal(builds, want) {
		t.Fatalf("build metrics = %v, want %v", builds, want)
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

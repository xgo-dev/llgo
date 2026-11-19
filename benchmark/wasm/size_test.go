package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWASISizeMatrix(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	out := filepath.Join(t.TempDir(), "output")
	t.Setenv("LLGO_BUILD_CACHE", "off")
	wantedModes := map[string][]string{
		"LLGo":           {},
		"LLGoDCE":        {"-deadcodedrop"},
		"LLGoDCEFullLTO": {"-deadcodedrop", "-globaldce=false", "-lto=full"},
	}
	calls := make(map[string]int)
	runner := func(_ context.Context, dir string, env []string, name string, args ...string) error {
		t.Helper()
		source, output := args[len(args)-1], args[slices.Index(args, "-o")+1]
		profile := filepath.Base(filepath.Dir(output))
		workload := filepath.Base(filepath.Dir(source))
		if dir != root || source != filepath.Join(other, "benchmark", "binary_size", workload, "main.go") {
			t.Fatalf("builder selected unmatched source or root: %q, %q", source, dir)
		}
		if !slices.Contains(env, "LLGO_BUILD_CACHE=on") || slices.Contains(env, "LLGO_BUILD_CACHE=off") {
			t.Fatalf("size build changed cache policy: %v", env)
		}
		switch name {
		case "dev-llgo":
			flags, ok := wantedModes[profile]
			if !ok || !slices.Equal(args[1:slices.Index(args, "-target")], append([]string{"-Oz", "-debug-artifact=none", "-ldflags=-s"}, flags...)) || !slices.Contains(args, "wasi") {
				t.Fatalf("wrong flags for %q: %v", profile, args)
			}
		case "go-ref":
			if workload == "cprintf" || !slices.Contains(args, "-ldflags=-s") || !slices.Contains(env, "GOOS=wasip1") {
				t.Fatalf("wrong Go reference for %q: %v", workload, args)
			}
		case "tinygo-ref":
			if workload == "cprintf" || !slices.Contains(args, "-opt=z") || !slices.Contains(args, "-no-debug") || !slices.Contains(args, "-target=wasi") {
				t.Fatalf("wrong TinyGo reference for %q: %v", workload, args)
			}
		default:
			t.Fatalf("unexpected compiler %q", name)
		}
		calls[workload]++
		return os.WriteFile(output, []byte("\x00asmtest"), 0o644)
	}
	args := []string{"-root", root, "-fixture-root", other, "-out", out,
		"-size-only", "-size-llgo", "dev-llgo", "-go", "go-ref", "-tinygo", "tinygo-ref"}
	if err := runCLI(context.Background(), args, runner); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls["cprintf"] != 3 || calls["println"] != 5 || calls["fmtprintf"] != 5 {
		t.Fatalf("wrong compiler matrix: %v", calls)
	}
	data, err := os.ReadFile(filepath.Join(out, "benchmark.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "BenchmarkWasmSize/wasi-size/") != 13 || strings.Contains(string(data), "BenchmarkWasmBuild/") {
		t.Fatalf("unexpected size-only metrics:\n%s", data)
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// These measurements use the same dev compiler for all three LLGo rows.
// Keep the cache configuration of the historical build-time series separate.
func measureWASISizeMatrix(ctx context.Context, runner commandRunner, env []string, root, fixtureRoot, out, llgo, goCommand, tinygo string) ([]measurement, error) {
	env = slices.DeleteFunc(slices.Clone(env), func(value string) bool {
		return strings.HasPrefix(value, "LLGO_BUILD_CACHE=")
	})
	env = append(env, "LLGO_BUILD_CACHE=on")
	// Include only the three fixed startup and formatting fixtures.
	var results []measurement
	for _, example := range wasmExamples {
		if example.name == "reflectcall" {
			continue
		}
		fixture := example.sourcePath(fixtureRoot)
		exampleOut := filepath.Join(out, "wasi-size", example.name)
		for _, mode := range []struct {
			name  string
			flags []string
		}{
			{"LLGo", nil},
			{"LLGoDCE", []string{"-deadcodedrop"}},
			{"LLGoDCEFullLTO", []string{"-deadcodedrop", "-globaldce=false", "-lto=full"}},
		} {
			flags := append([]string{"-Oz", "-debug-artifact=none", "-ldflags=-s"}, mode.flags...)
			profile := wasmProfile{name: mode.name, target: "wasi", outputExt: ".wasm", flags: flags}
			result, err := measureProfile(ctx, runner, env, root, llgo, exampleOut, fixture, profile, 0)
			if err != nil {
				return nil, fmt.Errorf("WASI size matrix %s/%s: %w", example.name, mode.name, err)
			}
			result.name = "wasi-size/" + example.name + "/" + mode.name
			results = append(results, result)
		}
		// The fixture exercises LLGo C FFI. Only provide references for
		// compilers that can build the identical Go source.
		if !example.goReference {
			continue
		}
		result, err := measureGoProfile(ctx, runner, env, root, goCommand, exampleOut, fixture,
			goWasmProfile{name: "Go", goos: "wasip1", flags: []string{"-ldflags=-s"}})
		if err != nil {
			return nil, fmt.Errorf("WASI size matrix %s/Go: %w", example.name, err)
		}
		result.name = "wasi-size/" + example.name + "/Go"
		results = append(results, result)

		tinyOut := filepath.Join(exampleOut, "bin", "tinygo")
		if err := os.MkdirAll(tinyOut, 0o755); err != nil {
			return nil, err
		}
		output := filepath.Join(tinyOut, "program.wasm")
		if err := runner(ctx, root, env, tinygo, "build", "-target=wasi", "-opt=z", "-no-debug", "-o", output, fixture); err != nil {
			return nil, fmt.Errorf("WASI size matrix %s/TinyGo: %w", example.name, err)
		}
		size, err := wasmModuleSize(output)
		if err != nil {
			return nil, err
		}
		results = append(results, measurement{name: "wasi-size/" + example.name + "/TinyGo", moduleBytes: size})
	}
	return results, nil
}

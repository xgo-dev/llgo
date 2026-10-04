package goroot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type gorootWasmProfile struct {
	name        string
	target      string
	goos        string
	llgoSuffix  string
	runner      string
	browserOnly bool
}

func selectGOROOTWasmProfile(name string) (gorootWasmProfile, bool, error) {
	switch name {
	case "":
		return gorootWasmProfile{}, false, nil
	case "J32-Emscripten":
		return gorootWasmProfile{name: name, target: "emscripten", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs"}, true, nil
	case "J64-Emscripten":
		return gorootWasmProfile{name: name, target: "emscripten-memory64", goos: "js", llgoSuffix: ".mjs", runner: "emscripten-memory64-runner.mjs"}, true, nil
	case "W32-WASI":
		return gorootWasmProfile{name: name, target: "wasi", goos: "wasip1", llgoSuffix: ".wasm", runner: "wasmer"}, true, nil
	case "J32-GoJS":
		return gorootWasmProfile{name: name, goos: "js", llgoSuffix: ".mjs", runner: "emscripten-runner.mjs", browserOnly: true}, true, nil
	default:
		return gorootWasmProfile{}, false, fmt.Errorf("unknown -wasm-profile=%q", name)
	}
}

func activeGOROOTWasmProfile() (gorootWasmProfile, bool) {
	p, ok, err := selectGOROOTWasmProfile(*flagWasmProfile)
	if err != nil {
		panic(err) // TestGoRootRunCases validates the flag before executing cases.
	}
	return p, ok
}

func gorootTargetEnv(env []string) []string {
	p, ok := activeGOROOTWasmProfile()
	if !ok {
		return env
	}
	out := append([]string{}, env...)
	out = upsertEnv(out, "GOOS="+p.goos)
	out = upsertEnv(out, "GOARCH=wasm")
	out = upsertEnv(out, "CGO_ENABLED=0")
	if p.goos == "wasip1" {
		out = upsertEnv(out, "GOWASIRUNTIME=wasmtime")
	}
	return out
}

func gorootRuntimeEnv(env []string) []string {
	out := gorootTargetEnv(env)
	if _, ok := activeGOROOTWasmProfile(); ok {
		// Keep the official Go baseline deterministic. The LLGo pthread backend
		// uses Wasmer and can still create Ms.
		out = upsertEnv(out, "GOMAXPROCS=1")
	}
	return out
}

func gorootArtifactPath(rootDir, label string, llgo bool) string {
	p, ok := activeGOROOTWasmProfile()
	if ok {
		if llgo {
			return filepath.Join(rootDir, label+p.llgoSuffix)
		}
		return filepath.Join(rootDir, label+".wasm")
	}
	out := filepath.Join(rootDir, label+".out")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	return out
}

func gorootBuildArgs(llgo bool, flags []string, output, target string) []string {
	args := []string{"build"}
	if llgo {
		if p, ok := activeGOROOTWasmProfile(); ok && p.target != "" {
			args = append(args, "-target", p.target)
		}
	}
	args = append(args, flags...)
	return append(args, "-o", output, target)
}

func envEntry(env []string, key string) string {
	prefix := key + "="
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

func runGOROOTArtifact(dir, artifact string, llgo bool, env []string, timeout time.Duration, programArgs ...string) ([]byte, []byte, int, time.Duration, error) {
	app, args, runEnv, err := gorootArtifactCommand(dir, artifact, llgo, env, programArgs...)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	return runProgram(dir, app, runEnv, timeout, args...)
}

func gorootArtifactCommand(dir, artifact string, llgo bool, env []string, programArgs ...string) (string, []string, []string, error) {
	p, ok := activeGOROOTWasmProfile()
	if !ok {
		return artifact, programArgs, env, nil
	}
	if p.runner == "wasmer" {
		cwd, err := filepath.Abs(dir)
		if err != nil {
			return "", nil, nil, fmt.Errorf("resolve Wasmer working directory: %w", err)
		}
		args := gorootWASIArgs(cwd, os.TempDir(), artifact, runtime.GOOS == "windows")
		// Suppress engine tracing at its source while preserving the module
		// cache and guest stdout/stderr, including log-shaped guest output.
		runEnv := upsertEnv(gorootRuntimeEnv(env), "RUST_LOG=off")
		return "wasmer", append(args, programArgs...), runEnv, nil
	}
	if !llgo {
		goroot := envEntry(env, "GOROOT")
		if goroot == "" {
			return "", nil, nil, errors.New("target Go execution requires GOROOT")
		}
		runner := filepath.Join(goroot, "lib", "wasm", "go_"+p.goos+"_wasm_exec")
		return runner, append([]string{artifact}, programArgs...), gorootRuntimeEnv(env), nil
	}
	root := envEntry(env, "LLGO_ROOT")
	if root == "" {
		return "", nil, nil, errors.New("target LLGo execution requires LLGO_ROOT")
	}
	runner := filepath.Join(root, "targets", p.runner)
	args := []string{runner}
	if p.browserOnly {
		args = append(args, "--browser-only")
	}
	args = append(args, artifact)
	args = append(args, programArgs...)
	return "node", args, gorootRuntimeEnv(env), nil
}

// uintptrescapes deliberately forces stack growth with 4096 recursive frames.
// LLGo's wasm goroutine stacks are fixed, so reserve a test-specific budget.
// Other cases retain the default to avoid multiplying every goroutine's memory.
func gorootWasmCaseBuildFlags(casePath string, flags []string) []string {
	if _, ok := activeGOROOTWasmProfile(); ok && casePath == "uintptrescapes.go" {
		return append(append([]string(nil), flags...), "-goroutine-stack-size=8MB")
	}
	return flags
}

// gorootWASIArgs applies the same host directory contract as the
// public WASI runner. The host is explicit so both contracts can be tested.
func gorootWASIArgs(cwd, tempDir, artifact string, windows bool) []string {
	workVolume, tempVolume, guestCwd := cwd, "/tmp", cwd
	if windows {
		// Map drive-letter paths to POSIX guest names, including Go's default
		// temporary directory. Wasmer selects its available backend itself.
		workVolume, tempVolume, guestCwd = cwd+":/work", tempDir+":/tmp", "/work"
	}
	return []string{"run", "--enable-exceptions", "--enable-simd", "--stack-size=1048576",
		"--volume=" + workVolume, "--volume=" + tempVolume, "--env=PWD=" + guestCwd, artifact, "--"}
}
